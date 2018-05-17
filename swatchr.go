package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"fmt"

	"sync"

	"code.cloudfoundry.org/bytefmt"
	"github.com/anacrolix/torrent"
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
)

var (
	logD *log.Logger
	logI *log.Logger
	logW *log.Logger
	logE *log.Logger
)

func initLoggers(debugMode bool) {
	debugHandle := ioutil.Discard
	if debugMode {
		debugHandle = os.Stdout
	}
	logD = log.New(debugHandle, "[D] ", log.Ldate|log.Ltime|log.Lshortfile)
	logI = log.New(os.Stdout, "[I] ", log.Ldate|log.Ltime|log.Lshortfile)
	logW = log.New(os.Stdout, "[W] ", log.Ldate|log.Ltime|log.Lshortfile)
	logE = log.New(os.Stderr, "[E] ", log.Ldate|log.Ltime)
}

const (
	changeTypeMovieAdded       = iota
	changeTypeGotInfo          = iota
	changeTypeProgressChanged  = iota
	changeTypeDownloadComplete = iota
)

const checkProgressTimeout = 1 * time.Second

type Change struct {
	Type     int
	Key      string // magnet link which is supposed to be unique
	Title    string
	Name     string
	SizeStr  string
	Progress int
	Estimate int // seconds
}

func (c Change) String() string {
	res, _ := json.Marshal(&c)
	return string(res)
}

var updates = make(chan Change)

type Listeners struct {
	chans []chan Change
	guard sync.Mutex
}

func (ls *Listeners) Add(c chan Change) {
	ls.guard.Lock()
	defer ls.guard.Unlock()
	ls.chans = append(ls.chans, c)
}

func (ls *Listeners) Notify(chg Change) {
	ls.guard.Lock()
	defer ls.guard.Unlock()
	for _, l := range ls.chans {
		l <- chg
	}
}

var listeners Listeners
var tclient, _ = torrent.NewClient(nil)

func main() {
	debugMode := flag.Bool("debug", false, "debug logging")
	configPath := flag.String("config", "/etc/swatchr/swatchr.conf", "config path")
	flag.Parse()

	initLoggers(*debugMode)

	// parse config
	conf := &Config{}
	if err := InitConfig(*configPath, conf); err != nil {
		logE.Fatalf("init config: %v", err)
	}

	catalog, err := syncCatalog(conf.Params.CatalogPath, conf.Params.StoragePath, conf.Params.Quota*1024*1024)
	if err != nil {
		logE.Fatalf("initialize catalog: %v", err)
	}

	go func() {
		for u := range updates {
			listeners.Notify(u)
		}
	}()
	//go composeUpdates(catalog)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", fs)
	http.Handle("/", newSwatchrHandler(handleIndex, catalog, conf))
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Access-Control-Allow-Origin", "*") })
	http.Handle("/add", newSwatchrHandler(handleAdd, catalog, conf))
	http.Handle("/updates", websocket.Handler(handleUpdates))
	logE.Fatalf("listen and serve: %v", http.ListenAndServe(fmt.Sprintf(":%d", conf.Params.Port), nil))
}

func prettifySize(size int64) string {
	return bytefmt.ByteSize(uint64(size))
}

func handleIndex(w http.ResponseWriter, r *http.Request, catalog *Catalog, conf *Config) {
	t, err := template.ParseFiles("static/html/index.html")
	if err != nil {
		logE.Printf("parse index.html: %v", err)
		return
	}

	t.Execute(w, &struct {
		Cat          *Catalog
		Conf         *ConfigParams
		PrettifySize func(int64) string
	}{catalog, &conf.Params, prettifySize})
}

func handleUpdates(ws *websocket.Conn) {
	l := make(chan Change)
	listeners.Add(l)
	for u := range l {
		msg, err := json.Marshal(&u)
		if err != nil {
			logE.Printf("encode new update: %v", err)
			continue
		}
		websocket.Message.Send(ws, string(msg))
	}
}

func handleAdd(w http.ResponseWriter, r *http.Request, catalog *Catalog, conf *Config) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println("body", string(body))
	var reqExpected struct {
		Title  string `json:"title"`
		Magnet string `json:"magnet"`
	}
	if err := json.Unmarshal(body, &reqExpected); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	errChan := make(chan error)
	go addFile(reqExpected.Title, reqExpected.Magnet, catalog, errChan)
	err = <-errChan
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err == nil {
		log.Println("successfully added")
		w.WriteHeader(http.StatusCreated)
		return
	}
	switch errors.Cause(err).(type) {
	case *alreadyExistsError:
		w.WriteHeader(http.StatusConflict)
	}
}

func addFile(title string, magnet string, catalog *Catalog, errChan chan error) {
	//defer tclient.Close()
	t, err := tclient.AddMagnet(magnet)
	if err != nil {
		logE.Printf("add magnet: %v", err)
		return
	}
	logD.Println("successfully added magnet")
	errChan <- nil

	catalog.AddMovie(Movie{Title: title, State: stateIndexing, Magnet: magnet})

	updates <- Change{
		Type:  changeTypeMovieAdded,
		Key:   magnet,
		Title: title,
	}

	<-t.GotInfo()
	sizeTotal := t.BytesCompleted() + t.BytesMissing()

	catalog.AddMovieInfo(magnet, t.Name(), sizeTotal)
	updates <- Change{
		Type:     changeTypeGotInfo,
		Key:      magnet,
		Name:     t.Name(),
		SizeStr:  prettifySize(sizeTotal),
		Progress: 0,
	}

	//logD.Println(t.Name())
	//logD.Println(t.Info().Files)
	//logD.Println(t.Info().Name)
	//logD.Println(t.Info().Length)

	//dprogress := make(chan int)

	go func(magnet string) {
		var velocities []int
		timePrev := time.Now()
		progPrev := 1
		log.Println("starting timer")
		ticker := time.NewTicker(checkProgressTimeout)
		for _ = range ticker.C {
			//log.Println("tick")
			//total := float64(t.BytesCompleted() + t.BytesMissing())
			//dprogress <- int(float64(t.BytesCompleted()) / sizeTotal * 100)
			progNew := progPrev + 1
			log.Printf("new progress: %v%%", progNew)
			if progNew-progPrev <= 0 {
				continue
			}

			var change Change
			change.Key = magnet
			if progNew >= 100 {
				log.Println("download complete")
				catalog.ChangeMovieState(magnet, stateDone)
				change.Type = changeTypeDownloadComplete
				change.Estimate = 0
				change.SizeStr = prettifySize(sizeTotal)
				updates <- change
				return
			}

			change.Type = changeTypeProgressChanged
			timeCur := time.Now()
			elapsed := timeCur.Second() - timePrev.Second()
			if elapsed > 0 {
				velocityRecent := elapsed / (progNew - progPrev) // seconds per 1%
				log.Printf("recent velocity: %dB/s", velocityRecent)
				if velocityRecent > 0 {
					velocities = append(velocities, velocityRecent)

					if len(velocities) >= 3 {
						velocityCur := avg(velocities[len(velocities)-3:])
						log.Printf("avg velocity: %d", velocityCur)
						change.Estimate = (100 - progNew) / velocityCur // s
					}
				}
			}

			change.Progress = progNew
			log.Printf("estimate: %d sec", change.Estimate)
			updates <- change

			//log.Println("new change", change)
			//dprogress <- start

			timePrev = timeCur
			progPrev = progNew
		}
	}(magnet)

	//t.DownloadAll()

	//tclient.WaitAll()
	logI.Print("done")
}

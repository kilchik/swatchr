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

	"github.com/anacrolix/torrent"
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

type Change struct {
	MovieName string
	Progress  int
}

var updates = make(chan Change)

func main() {
	debugMode := flag.Bool("debug", false, "debug logging")
	configPath := flag.String("config", "/etc/swatchr/swatchr.conf", "config path")
	flag.Parse()

	initLoggers(*debugMode)

	// parse config
	var conf configImpl
	if err := InitConfig(*configPath, &conf); err != nil {
		logE.Fatalf("init config: %v", err)
	}

	catalog, err := syncCatalog(conf.params.CatalogPath, conf.params.StoragePath, conf.params.Quota*1024*1024)
	if err != nil {
		logE.Fatalf("initialize catalog: %v", err)
	}

	go composeUpdates(catalog)

	http.Handle("/", newSwatchrHandler(handleIndex, catalog))
	http.Handle("/add", newSwatchrHandler(handleAdd, catalog))
	http.Handle("/updates", websocket.Handler(handleUpdates))
	logE.Fatalf("listen and serve: %v", http.ListenAndServe(conf.params.ListenAddr, nil))
}

func composeUpdates(c *Catalog) {
	t := time.NewTicker(3 * time.Second)
	for _ = range t.C {
		for _, mov := range c.Movies {
			if mov.State == stateActive {
				select {
				case pro := <-mov.progress:
					updates <- Change{
						MovieName: mov.Name,
						Progress:  pro,
					}
				default:
					continue
				}
			}
		}
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request, catalog *Catalog) {
	t, err := template.ParseFiles("static/html/index.html")
	if err != nil {
		logE.Printf("parse index.html: %v", err)
		return
	}

	t.Execute(w, catalog)
}

func handleUpdates(ws *websocket.Conn) {
	for u := range updates {
		msg, err := json.Marshal(&u)
		if err != nil {
			logE.Printf("encode new update: %v", err)
			continue
		}
		websocket.Message.Send(ws, string(msg))
	}
}

func handleAdd(w http.ResponseWriter, r *http.Request, catalog *Catalog) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var reqExpected struct {
		Magnet string `json:"magnet"`
	}
	if err := json.Unmarshal(body, &reqExpected); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	go addFile(reqExpected.Magnet, catalog)
}

func addFile(magnet string, catalog *Catalog) {
	tclient, err := torrent.NewClient(nil)
	if err != nil {
		logE.Printf("new client: %v", err)
		return
	}
	defer tclient.Close()
	t, err := tclient.AddMagnet(magnet)
	if err != nil {
		logE.Printf("add magnet: %v", err)
		return
	}
	<-t.GotInfo()
	logD.Println("got info")
	logD.Println(t.Name())
	logD.Println(t.Info().Files)
	logD.Println(t.Info().Name)
	logD.Println(t.Info().Length)

	dprogress := make(chan int)
	catalog.AddMovie(Movie{Name: t.Name(), Size: t.BytesMissing(), State: stateActive, Magnet: magnet, progress: dprogress})

	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for _ = range ticker.C {
			total := float64(t.BytesCompleted() + t.BytesMissing())
			dprogress <- int(float64(t.BytesCompleted()) / total * 100)
		}
	}()

	t.DownloadAll()

	tclient.WaitAll()
	logI.Print("done")
}

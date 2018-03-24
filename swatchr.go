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

type handleFunc func(w http.ResponseWriter, r *http.Request, catalog *Catalog)

type swatchrHandler struct {
	handle  handleFunc
	catalog *Catalog
}

func newSwatchrHandler(handle handleFunc, catalog *Catalog) *swatchrHandler {
	return &swatchrHandler{handle: handle, catalog: catalog}
}

func (sh swatchrHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sh.handle(w, r, sh.catalog)
}

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

	http.Handle("/", newSwatchrHandler(handleIndex, catalog))
	http.Handle("/add", newSwatchrHandler(handleAdd, catalog))
	logE.Fatalf("listen and serve: %v", http.ListenAndServe(conf.params.ListenAddr, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request, catalog *Catalog) {
	t, err := template.ParseFiles("static/html/index.html")
	if err != nil {
		logE.Printf("parse index.html: %v", err)
		return
	}
	t.Execute(w, catalog)
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

	go downloadFile(reqExpected.Magnet)
}

func downloadFile(magnet string) {
	c, _ := torrent.NewClient(nil)
	defer c.Close()
	t, _ := c.AddMagnet(magnet)
	<-t.GotInfo()
	logD.Println(t.Name())
	logD.Println(t.Info().Files)
	logD.Println(t.Info().Name)
	logD.Println(t.Info().Length)

	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for _ = range ticker.C {
			total := float64(t.BytesCompleted() + t.BytesMissing())

			logI.Printf("completed: %f%%", float64(t.BytesCompleted())/total*100)
		}
	}()

	t.DownloadAll()

	c.WaitAll()
	logI.Print("done")
}

package main

import (
	"net/http"
)

type handleFunc func(w http.ResponseWriter, r *http.Request, catalog *Catalog, conf *Config)

type swatchrHandler struct {
	handle  handleFunc
	catalog *Catalog
	conf    *Config
}

func newSwatchrHandler(handle handleFunc, catalog *Catalog, conf *Config) *swatchrHandler {
	return &swatchrHandler{handle: handle, catalog: catalog, conf: conf}
}

func (sh swatchrHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sh.handle(w, r, sh.catalog, sh.conf)
}

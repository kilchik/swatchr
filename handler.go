package main

import (
	"net/http"
)

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

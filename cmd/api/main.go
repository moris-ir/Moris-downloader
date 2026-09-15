package main

import (
	"github.com/moris3245/moris/internal/app"
	"github.com/moris3245/moris/internal/config"
	"github.com/moris3245/moris/internal/httpapi"
	"log"
	"net/http"
)

func main() {
	cfg := config.Load()
	a := app.New(cfg)
	log.Printf("MORIS API listening on %s", cfg.HTTPAddr)
	log.Fatal(http.ListenAndServe(cfg.HTTPAddr, httpapi.New(a).Handler()))
}

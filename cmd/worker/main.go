package main

import (
	"context"
	"github.com/moris3245/moris/internal/app"
	"github.com/moris3245/moris/internal/config"
	"log"
)

func main() {
	cfg := config.Load()
	a := app.New(cfg)
	log.Printf("MORIS worker starting with concurrency=%d", cfg.WorkerConcurrency)
	a.RunWorkers(context.Background())
	select {}
}

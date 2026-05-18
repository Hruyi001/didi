package main

import (
	"log"
	"time"

	api "didi/backend/internal/http"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
)

func main() {
	s := store.NewMemoryStore()
	store.SeedDemoData(s)
	dispatch := services.NewDispatchService(s)
	app := &api.App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: dispatch, Payment: services.NewPaymentService(s), Events: api.NewEventHub()}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := dispatch.ProcessTimeouts(time.Now()); err != nil {
				log.Printf("process dispatch timeouts: %v", err)
			}
		}
	}()
	r := api.NewRouter(app)
	log.Println("ride-hailing API listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

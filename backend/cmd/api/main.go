package main

import (
	"log"
	"os"
	"time"

	api "didi/backend/internal/http"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	s := buildStore()
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

func buildStore() store.Store {
	backend := os.Getenv("STORE_BACKEND")
	switch backend {
	case "", "memory":
		s := store.NewMemoryStore()
		store.SeedDemoData(s)
		return s
	case "mysql":
		dsn := os.Getenv("MYSQL_DSN")
		if dsn == "" {
			log.Fatal("MYSQL_DSN is required when STORE_BACKEND=mysql")
		}
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("open mysql store: %v", err)
		}
		s := store.NewMySQLStore(db)
		if os.Getenv("SEED_DEMO_DATA") == "true" {
			if err := store.SeedDemoDataForStore(s); err != nil {
				log.Fatalf("seed mysql demo data: %v", err)
			}
		}
		return s
	default:
		log.Fatalf("unsupported STORE_BACKEND: %s", backend)
		return nil
	}
}

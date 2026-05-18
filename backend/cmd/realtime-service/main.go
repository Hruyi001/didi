package main

import (
	"log"
	"time"

	"didi/backend/internal/realtime"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	hub := realtime.NewHub()
	realtime.RegisterHTTP(r, hub)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hub.Publish("heartbeat", []byte(`{"type":"HEARTBEAT"}`))
		}
	}()
	log.Fatal(r.Run(":18088"))
}

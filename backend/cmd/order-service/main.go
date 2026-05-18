package main

import (
	"log"

	"didi/backend/internal/order"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	order.RegisterHTTP(r, order.NewService(order.NewMemoryRepo(), nil))
	log.Fatal(r.Run(":18085"))
}

package main

import (
	"log"

	"didi/backend/internal/payment"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	repo := payment.NewMemoryRepo()
	repo.Create("order-1", 2800)
	payment.RegisterHTTP(r, payment.NewService(repo, nil))
	log.Fatal(r.Run(":18087"))
}

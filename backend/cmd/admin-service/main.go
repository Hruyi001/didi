package main

import (
	"log"

	"didi/backend/internal/admin"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	admin.RegisterHTTP(r, admin.NewService())
	log.Fatal(r.Run(":18089"))
}

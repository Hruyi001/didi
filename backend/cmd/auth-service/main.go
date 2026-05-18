package main

import (
	"log"

	"didi/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	auth.RegisterHTTP(r, auth.NewService(auth.NewMemoryRepo()))
	log.Fatal(r.Run(":18081"))
}

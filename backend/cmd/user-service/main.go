package main

import (
	"log"

	"didi/backend/internal/user"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	user.RegisterHTTP(r, user.NewService(user.NewMemoryRepo()))
	log.Fatal(r.Run(":18082"))
}

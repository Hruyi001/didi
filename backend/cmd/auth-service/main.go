package main

import (
	"log"

	"didi/backend/internal/auth"
	"didi/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	auth.RegisterHTTP(r, auth.NewService(auth.NewMemoryRepo(), services.NewRiskService(), services.NewAuthService()))
	log.Fatal(r.Run(":18081"))
}

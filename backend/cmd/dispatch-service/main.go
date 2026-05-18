package main

import (
	"log"

	"didi/backend/internal/dispatch"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	dispatch.RegisterHTTP(r, dispatch.NewService(dispatch.NewMemoryRepo(), nil))
	log.Fatal(r.Run(":18086"))
}

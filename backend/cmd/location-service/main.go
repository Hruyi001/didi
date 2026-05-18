package main

import (
	"log"

	"didi/backend/internal/location"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	repo := location.NewMemoryRepo()
	repo.Upsert("d1", 116.390, 39.900)
	repo.Upsert("d2", 116.401, 39.901)
	location.RegisterHTTP(r, location.NewService(repo))
	log.Fatal(r.Run(":18084"))
}

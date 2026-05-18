package main

import (
	"log"

	"didi/backend/internal/driver"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	repo := driver.NewMemoryRepo()
	repo.SeedApproved("driver-1", "13900000001")
	driver.RegisterHTTP(r, driver.NewService(repo))
	log.Fatal(r.Run(":18083"))
}

package main

import (
	"log"
	"os"

	"didi/backend/internal/admin"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	if os.Getenv("AUTH_TOKEN_SECRET") == "" {
		log.Fatal("AUTH_TOKEN_SECRET is required")
	}
	dsn := os.Getenv("ADMIN_MYSQL_DSN")
	if dsn == "" {
		log.Fatal("ADMIN_MYSQL_DSN is required")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("open admin mysql: %v", err)
	}
	r := gin.Default()
	admin.RegisterHTTP(r, admin.NewService(admin.NewMySQLDriverRepository(db)))
	log.Fatal(r.Run(":18089"))
}

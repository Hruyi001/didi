package config

import "os"

type ServiceConfig struct {
	ServiceName  string
	HTTPAddr     string
	MySQLDSN     string
	RedisAddr    string
	KafkaBrokers string
}

func Default(service string) ServiceConfig {
	return ServiceConfig{
		ServiceName:  service,
		HTTPAddr:     env("HTTP_ADDR", ":8080"),
		MySQLDSN:     env("MYSQL_DSN", "root:root@tcp(mysql:3306)/"+service+"?parseTime=true"),
		RedisAddr:    env("REDIS_ADDR", "redis:6379"),
		KafkaBrokers: env("KAFKA_BROKERS", "kafka:9092"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

package config

import "testing"

func TestServiceConfigDefaults(t *testing.T) {
	cfg := Default("order-service")
	if cfg.ServiceName != "order-service" {
		t.Fatalf("expected service name, got %q", cfg.ServiceName)
	}
	if cfg.HTTPAddr == "" {
		t.Fatal("expected default HTTP address")
	}
	if cfg.KafkaBrokers == "" {
		t.Fatal("expected kafka brokers default")
	}
}

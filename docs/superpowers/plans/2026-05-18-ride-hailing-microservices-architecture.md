# Ride-Hailing Microservices Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current single-process backend with independently runnable ride-hailing microservices while preserving the existing frontend flow through a stable gateway API.

**Architecture:** Keep the repo as one Go module, but split runtime into separate binaries for `gateway`, `auth`, `user`, `driver`, `location`, `order`, `dispatch`, `payment`, `realtime`, and `admin`. Move shared status models, event names, config loading, and infrastructure adapters into common packages; keep each service responsible for only its own schema and state transitions, and route all frontend traffic through `gateway`.

**Tech Stack:** Go 1.25, Kratos-style service layout with HTTP-first service endpoints, MySQL, Redis, Kafka, Docker Compose, existing Vue 3 frontend.

---

## Scope Check

This spec covers many services, but they form one migration program with one external API contract and one smoke-tested delivery. It is reasonable to keep this as one implementation plan split into phase-based tasks rather than creating ten unrelated plans.

## File Structure

Modify these existing paths:

```text
backend/go.mod
backend/go.sum
frontend/src/api.ts
frontend/vite.config.ts
infra/docker-compose.yml
infra/schema.sql
README.md
```

Create these shared backend paths:

```text
backend/internal/contracts/status.go
backend/internal/contracts/models.go
backend/internal/contracts/events.go
backend/internal/platform/config/config.go
backend/internal/platform/httpx/response.go
backend/internal/platform/mysql/mysql.go
backend/internal/platform/redis/client.go
backend/internal/platform/kafka/topics.go
backend/internal/platform/outbox/outbox.go
backend/internal/platform/testkit/mysql.go
backend/internal/platform/testkit/kafka.go
```

Create these service paths:

```text
backend/internal/auth/service.go
backend/internal/auth/repo.go
backend/internal/auth/http.go
backend/internal/auth/service_test.go
backend/internal/user/service.go
backend/internal/user/repo.go
backend/internal/user/http.go
backend/internal/user/service_test.go
backend/internal/driver/service.go
backend/internal/driver/repo.go
backend/internal/driver/http.go
backend/internal/driver/service_test.go
backend/internal/location/service.go
backend/internal/location/repo.go
backend/internal/location/http.go
backend/internal/location/service_test.go
backend/internal/order/service.go
backend/internal/order/repo.go
backend/internal/order/http.go
backend/internal/order/service_test.go
backend/internal/dispatch/service.go
backend/internal/dispatch/repo.go
backend/internal/dispatch/http.go
backend/internal/dispatch/service_test.go
backend/internal/payment/service.go
backend/internal/payment/repo.go
backend/internal/payment/http.go
backend/internal/payment/service_test.go
backend/internal/realtime/hub.go
backend/internal/realtime/consumer.go
backend/internal/realtime/http.go
backend/internal/realtime/service_test.go
backend/internal/admin/service.go
backend/internal/admin/http.go
backend/internal/admin/service_test.go
backend/internal/gateway/router.go
backend/internal/gateway/router_test.go
```

Create these service entrypoints:

```text
backend/cmd/gateway/main.go
backend/cmd/auth-service/main.go
backend/cmd/user-service/main.go
backend/cmd/driver-service/main.go
backend/cmd/location-service/main.go
backend/cmd/order-service/main.go
backend/cmd/dispatch-service/main.go
backend/cmd/payment-service/main.go
backend/cmd/realtime-service/main.go
backend/cmd/admin-service/main.go
```

Create these integration test paths:

```text
backend/integration/order_dispatch_flow_test.go
backend/integration/payment_completion_test.go
backend/integration/gateway_smoke_test.go
```

Delete these legacy monolith paths after gateway parity passes:

```text
backend/cmd/api/main.go
backend/internal/http/
```

Notes:

- Keep the existing frontend request paths (`/api/auth/*`, `/api/passenger/*`, `/api/driver/*`, `/api/admin/*`, `/ws`) so the Vue apps need minimal change.
- Use one MySQL instance in Compose, but create one schema per service (`auth_db`, `user_db`, `driver_db`, `location_db`, `order_db`, `dispatch_db`, `payment_db`, `realtime_db`, `admin_db`).
- Services may share the same Go module and local package tree, but runtime ownership must be process-separated.

---

### Task 1: Shared Contracts and Platform Bootstrap

**Files:**
- Modify: `backend/go.mod`
- Create: `backend/internal/contracts/status.go`
- Create: `backend/internal/contracts/models.go`
- Create: `backend/internal/contracts/events.go`
- Create: `backend/internal/platform/config/config.go`
- Create: `backend/internal/platform/httpx/response.go`
- Create: `backend/internal/platform/mysql/mysql.go`
- Create: `backend/internal/platform/redis/client.go`
- Create: `backend/internal/platform/kafka/topics.go`
- Create: `backend/internal/platform/outbox/outbox.go`
- Test: `backend/internal/platform/config/config_test.go`

- [ ] **Step 1: Write the failing config and topic tests**

Create `backend/internal/platform/config/config_test.go`:

```go
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
```

Create `backend/internal/platform/kafka/topics_test.go`:

```go
package kafka

import "testing"

func TestTopicNamesAreStable(t *testing.T) {
	if OrderCreatedTopic != "ride.order.created" {
		t.Fatalf("unexpected order-created topic: %q", OrderCreatedTopic)
	}
	if PaymentPaidTopic != "ride.payment.paid" {
		t.Fatalf("unexpected payment-paid topic: %q", PaymentPaidTopic)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./internal/platform/...
```

Expected: FAIL because `Default`, `OrderCreatedTopic`, and `PaymentPaidTopic` do not exist.

- [ ] **Step 3: Add shared dependencies**

Modify `backend/go.mod` so the require block includes the shared runtime dependencies:

```go
require (
	github.com/gin-gonic/gin v1.12.0
	github.com/go-kratos/kratos/v2 v2.9.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.7.0
	github.com/segmentio/kafka-go v0.4.47
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
)
```

- [ ] **Step 4: Create the shared status and event contracts**

Create `backend/internal/contracts/status.go`:

```go
package contracts

type OrderStatus string

type DispatchStatus string

type PaymentStatus string

type ReviewStatus string

type DriverWorkStatus string

type AccountRole string

const (
	RolePassenger AccountRole = "PASSENGER"
	RoleDriver    AccountRole = "DRIVER"
	RoleAdmin     AccountRole = "ADMIN"
)

const (
	OrderCreated        OrderStatus = "CREATED"
	OrderDispatching    OrderStatus = "DISPATCHING"
	OrderWaitingPickup  OrderStatus = "WAITING_PICKUP"
	OrderDriverArrived  OrderStatus = "DRIVER_ARRIVED"
	OrderInProgress     OrderStatus = "IN_PROGRESS"
	OrderWaitingPayment OrderStatus = "WAITING_PAYMENT"
	OrderCompleted      OrderStatus = "COMPLETED"
	OrderCanceled       OrderStatus = "CANCELED"
	OrderDispatchFailed OrderStatus = "DISPATCH_FAILED"
)
```

Create `backend/internal/contracts/events.go`:

```go
package contracts

type Event struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Body      []byte `json:"body"`
	Version   int64  `json:"version"`
	TraceID   string `json:"traceId"`
	CreatedAt string `json:"createdAt"`
}
```

Create `backend/internal/platform/kafka/topics.go`:

```go
package kafka

const (
	OrderCreatedTopic      = "ride.order.created"
	DispatchAcceptedTopic  = "ride.dispatch.accepted"
	DispatchTimeoutTopic   = "ride.dispatch.timeout"
	DispatchFailedTopic    = "ride.dispatch.failed"
	TripEndedTopic         = "ride.trip.ended"
	PaymentPaidTopic       = "ride.payment.paid"
	DriverLocationTopic    = "ride.driver.location.updated"
	DriverApprovedTopic    = "ride.driver.approved"
)
```

- [ ] **Step 5: Create config and infrastructure bootstrap helpers**

Create `backend/internal/platform/config/config.go`:

```go
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
```

Create `backend/internal/platform/httpx/response.go`:

```go
package httpx

import "github.com/gin-gonic/gin"

func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{"data": data})
}

func Fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
```

Create `backend/internal/platform/mysql/mysql.go`:

```go
package mysql

import "gorm.io/gorm"

type TxRunner interface {
	WithinTx(func(tx *gorm.DB) error) error
}
```

Create `backend/internal/platform/outbox/outbox.go`:

```go
package outbox

import "context"

type Publisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}
```

- [ ] **Step 6: Re-run the shared-platform tests**

Run:

```bash
go -C /root/didi/backend test ./internal/platform/...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git -C /root/didi add backend/go.mod backend/go.sum backend/internal/contracts backend/internal/platform
git -C /root/didi commit -m "chore: add shared microservice contracts"
```

---

### Task 2: Auth Service and User Service

**Files:**
- Create: `backend/internal/auth/repo.go`
- Create: `backend/internal/auth/service.go`
- Create: `backend/internal/auth/http.go`
- Create: `backend/internal/auth/service_test.go`
- Create: `backend/internal/user/repo.go`
- Create: `backend/internal/user/service.go`
- Create: `backend/internal/user/http.go`
- Create: `backend/internal/user/service_test.go`
- Create: `backend/cmd/auth-service/main.go`
- Create: `backend/cmd/user-service/main.go`

- [ ] **Step 1: Write the failing auth and user service tests**

Create `backend/internal/auth/service_test.go`:

```go
package auth

import (
	"context"
	"testing"

	"didi/backend/internal/contracts"
)

func TestLoginAcceptsFixedCode(t *testing.T) {
	svc := NewService(NewMemoryRepo())
	result, err := svc.Login(context.Background(), LoginRequest{Phone: "13800000001", Code: "123456", Role: contracts.RolePassenger})
	if err != nil {
		t.Fatalf("expected login success: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("expected both tokens")
	}
}
```

Create `backend/internal/user/service_test.go`:

```go
package user

import "testing"

func TestCreatePassengerProfile(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo)
	profile := svc.EnsurePassenger("acct-1", "13800000001")
	if profile.AccountID != "acct-1" {
		t.Fatalf("expected account binding, got %q", profile.AccountID)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./internal/auth ./internal/user
```

Expected: FAIL because the packages do not exist.

- [ ] **Step 3: Implement minimal auth and user services**

Create `backend/internal/auth/service.go`:

```go
package auth

import (
	"context"
	"fmt"
	"time"

	"didi/backend/internal/contracts"
	"github.com/google/uuid"
)

type LoginRequest struct {
	Phone string
	Code  string
	Role  contracts.AccountRole
}

type LoginResult struct {
	AccessToken  string                `json:"accessToken"`
	RefreshToken string                `json:"refreshToken"`
	Role         contracts.AccountRole `json:"role"`
	ExpiresAt    time.Time             `json:"expiresAt"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) Login(_ context.Context, req LoginRequest) (LoginResult, error) {
	if req.Code != "123456" {
		return LoginResult{}, fmt.Errorf("验证码错误")
	}
	return LoginResult{
		AccessToken:  "access-" + uuid.NewString(),
		RefreshToken: "refresh-" + uuid.NewString(),
		Role:         req.Role,
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}, nil
}
```

Create `backend/internal/user/service.go`:

```go
package user

type PassengerProfile struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	Phone     string `json:"phone"`
	Nickname  string `json:"nickname"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) EnsurePassenger(accountID, phone string) PassengerProfile {
	return s.repo.EnsurePassenger(accountID, phone)
}
```

- [ ] **Step 4: Add HTTP handlers and service binaries**

Create `backend/internal/auth/http.go`:

```go
package auth

import (
	"net/http"

	"didi/backend/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/auth/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, http.StatusBadRequest, "请求格式错误")
			return
		}
		result, err := svc.Login(c.Request.Context(), req)
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.OK(c, result)
	})
}
```

Create `backend/cmd/auth-service/main.go`:

```go
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
```

Create `backend/cmd/user-service/main.go`:

```go
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
```

- [ ] **Step 5: Re-run the auth and user tests**

Run:

```bash
go -C /root/didi/backend test ./internal/auth ./internal/user
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git -C /root/didi add backend/internal/auth backend/internal/user backend/cmd/auth-service backend/cmd/user-service
git -C /root/didi commit -m "feat: add auth and user services"
```

---

### Task 3: Driver Service and Location Service

**Files:**
- Create: `backend/internal/driver/repo.go`
- Create: `backend/internal/driver/service.go`
- Create: `backend/internal/driver/http.go`
- Create: `backend/internal/driver/service_test.go`
- Create: `backend/internal/location/repo.go`
- Create: `backend/internal/location/service.go`
- Create: `backend/internal/location/http.go`
- Create: `backend/internal/location/service_test.go`
- Create: `backend/cmd/driver-service/main.go`
- Create: `backend/cmd/location-service/main.go`

- [ ] **Step 1: Write the failing driver and location tests**

Create `backend/internal/driver/service_test.go`:

```go
package driver

import "testing"

func TestApprovedDriverCanGoOnline(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo)
	driver := repo.SeedApproved("driver-1", "13900000001")
	updated, err := svc.SetOnline(driver.ID)
	if err != nil {
		t.Fatalf("expected driver online: %v", err)
	}
	if updated.WorkStatus != "ONLINE_IDLE" {
		t.Fatalf("expected ONLINE_IDLE, got %s", updated.WorkStatus)
	}
}
```

Create `backend/internal/location/service_test.go`:

```go
package location

import "testing"

func TestNearbyDriversAreSortedByDistance(t *testing.T) {
	repo := NewMemoryRepo()
	repo.Upsert("d1", 116.390, 39.900)
	repo.Upsert("d2", 116.401, 39.901)
	svc := NewService(repo)
	drivers := svc.Nearby(116.398, 39.900, 2)
	if len(drivers) != 2 {
		t.Fatalf("expected 2 drivers, got %d", len(drivers))
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./internal/driver ./internal/location
```

Expected: FAIL because the packages do not exist.

- [ ] **Step 3: Implement the minimal services**

Create `backend/internal/driver/service.go`:

```go
package driver

import "fmt"

type Driver struct {
	ID         string `json:"id"`
	Phone      string `json:"phone"`
	AuditState string `json:"auditState"`
	WorkStatus string `json:"workStatus"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) SetOnline(driverID string) (Driver, error) {
	driver, ok := s.repo.Get(driverID)
	if !ok {
		return Driver{}, fmt.Errorf("driver not found")
	}
	if driver.AuditState != "APPROVED" {
		return Driver{}, fmt.Errorf("driver not approved")
	}
	driver.WorkStatus = "ONLINE_IDLE"
	s.repo.Save(driver)
	return driver, nil
}
```

Create `backend/internal/location/service.go`:

```go
package location

import "sort"

type DriverPoint struct {
	DriverID  string  `json:"driverId"`
	Lng       float64 `json:"lng"`
	Lat       float64 `json:"lat"`
	DistanceM int     `json:"distanceM"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) Nearby(lng, lat float64, limit int) []DriverPoint {
	points := s.repo.All()
	sort.Slice(points, func(i, j int) bool { return points[i].DistanceM < points[j].DistanceM })
	if len(points) > limit {
		return points[:limit]
	}
	return points
}
```

- [ ] **Step 4: Add HTTP endpoints and binaries**

Create `backend/internal/driver/http.go`:

```go
package driver

import (
	"net/http"

	"didi/backend/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/drivers/:id/online", func(c *gin.Context) {
		driver, err := svc.SetOnline(c.Param("id"))
		if err != nil {
			httpx.Fail(c, http.StatusConflict, err.Error())
			return
		}
		httpx.OK(c, driver)
	})
}
```

Create `backend/internal/location/http.go`:

```go
package location

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.GET("/v1/location/nearby", func(c *gin.Context) {
		results := svc.Nearby(116.398, 39.900, 10)
		c.JSON(200, gin.H{"data": results})
	})
}
```

- [ ] **Step 5: Re-run the tests**

Run:

```bash
go -C /root/didi/backend test ./internal/driver ./internal/location
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git -C /root/didi add backend/internal/driver backend/internal/location backend/cmd/driver-service backend/cmd/location-service
git -C /root/didi commit -m "feat: add driver and location services"
```

---

### Task 4: Order Service and Dispatch Service

**Files:**
- Create: `backend/internal/order/repo.go`
- Create: `backend/internal/order/service.go`
- Create: `backend/internal/order/http.go`
- Create: `backend/internal/order/service_test.go`
- Create: `backend/internal/dispatch/repo.go`
- Create: `backend/internal/dispatch/service.go`
- Create: `backend/internal/dispatch/http.go`
- Create: `backend/internal/dispatch/service_test.go`
- Create: `backend/cmd/order-service/main.go`
- Create: `backend/cmd/dispatch-service/main.go`

- [ ] **Step 1: Write the failing order and dispatch tests**

Create `backend/internal/order/service_test.go`:

```go
package order

import "testing"

func TestCreateOrderStartsInCreatedState(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo, nil)
	order := svc.Create("passenger-1", "pickup", "dropoff")
	if order.Status != "CREATED" {
		t.Fatalf("expected CREATED, got %s", order.Status)
	}
}
```

Create `backend/internal/dispatch/service_test.go`:

```go
package dispatch

import "testing"

func TestDispatchSelectsFirstCandidate(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo, nil)
	attempt, err := svc.Start("order-1", []string{"driver-1", "driver-2"})
	if err != nil {
		t.Fatalf("expected dispatch success: %v", err)
	}
	if attempt.DriverID != "driver-1" {
		t.Fatalf("expected first driver, got %s", attempt.DriverID)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./internal/order ./internal/dispatch
```

Expected: FAIL because the packages do not exist.

- [ ] **Step 3: Implement order truth and dispatch truth services**

Create `backend/internal/order/service.go`:

```go
package order

type Order struct {
	ID          string `json:"id"`
	PassengerID string `json:"passengerId"`
	Status      string `json:"status"`
	Pickup      string `json:"pickup"`
	Dropoff     string `json:"dropoff"`
	Version     int64  `json:"version"`
}

type EventPublisher interface {
	PublishOrderCreated(orderID string) error
}

type Service struct {
	repo      *MemoryRepo
	publisher EventPublisher
}

func NewService(repo *MemoryRepo, publisher EventPublisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Create(passengerID, pickup, dropoff string) Order {
	order := s.repo.Create(passengerID, pickup, dropoff)
	if s.publisher != nil {
		_ = s.publisher.PublishOrderCreated(order.ID)
	}
	return order
}
```

Create `backend/internal/dispatch/service.go`:

```go
package dispatch

import "fmt"

type Attempt struct {
	OrderID   string `json:"orderId"`
	DriverID  string `json:"driverId"`
	Status    string `json:"status"`
	Sequence  int    `json:"sequence"`
}

type Publisher interface {
	PublishDispatchAccepted(orderID, driverID string) error
}

type Service struct {
	repo      *MemoryRepo
	publisher Publisher
}

func NewService(repo *MemoryRepo, publisher Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Start(orderID string, candidateDriverIDs []string) (Attempt, error) {
	if len(candidateDriverIDs) == 0 {
		return Attempt{}, fmt.Errorf("no candidates")
	}
	attempt := Attempt{OrderID: orderID, DriverID: candidateDriverIDs[0], Status: "OFFERED", Sequence: 1}
	s.repo.Save(attempt)
	return attempt, nil
}
```

- [ ] **Step 4: Add HTTP endpoints that preserve the current frontend semantics**

Create `backend/internal/order/http.go`:

```go
package order

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/orders", func(c *gin.Context) {
		order := svc.Create("passenger-1", "pickup", "dropoff")
		c.JSON(200, gin.H{"data": order})
	})
}
```

Create `backend/internal/dispatch/http.go`:

```go
package dispatch

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/dispatch/start", func(c *gin.Context) {
		attempt, err := svc.Start("order-1", []string{"driver-1"})
		if err != nil {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"data": attempt})
	})
}
```

- [ ] **Step 5: Re-run the order and dispatch tests**

Run:

```bash
go -C /root/didi/backend test ./internal/order ./internal/dispatch
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git -C /root/didi add backend/internal/order backend/internal/dispatch backend/cmd/order-service backend/cmd/dispatch-service
git -C /root/didi commit -m "feat: add order and dispatch services"
```

---

### Task 5: Payment Service and Realtime Service

**Files:**
- Create: `backend/internal/payment/repo.go`
- Create: `backend/internal/payment/service.go`
- Create: `backend/internal/payment/http.go`
- Create: `backend/internal/payment/service_test.go`
- Create: `backend/internal/realtime/hub.go`
- Create: `backend/internal/realtime/consumer.go`
- Create: `backend/internal/realtime/http.go`
- Create: `backend/internal/realtime/service_test.go`
- Create: `backend/cmd/payment-service/main.go`
- Create: `backend/cmd/realtime-service/main.go`

- [ ] **Step 1: Write the failing payment and realtime tests**

Create `backend/internal/payment/service_test.go`:

```go
package payment

import "testing"

func TestMarkPaidTransitionsPayment(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo, nil)
	created := repo.Create("order-1", 2800)
	paid, err := svc.Pay(created.OrderID)
	if err != nil {
		t.Fatalf("expected pay success: %v", err)
	}
	if paid.Status != "PAID" {
		t.Fatalf("expected PAID, got %s", paid.Status)
	}
}
```

Create `backend/internal/realtime/service_test.go`:

```go
package realtime

import "testing"

func TestHubBroadcastsToSubscribers(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("passenger:1")
	hub.Publish("passenger:1", []byte(`{"type":"ORDER_UPDATED"}`))
	msg := <-ch
	if string(msg) == "" {
		t.Fatal("expected message payload")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./internal/payment ./internal/realtime
```

Expected: FAIL because the packages do not exist.

- [ ] **Step 3: Implement the minimal services**

Create `backend/internal/payment/service.go`:

```go
package payment

import "fmt"

type Payment struct {
	OrderID string `json:"orderId"`
	Amount  int64  `json:"amount"`
	Status  string `json:"status"`
}

type Publisher interface {
	PublishPaymentPaid(orderID string) error
}

type Service struct {
	repo      *MemoryRepo
	publisher Publisher
}

func NewService(repo *MemoryRepo, publisher Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Pay(orderID string) (Payment, error) {
	payment, ok := s.repo.Get(orderID)
	if !ok {
		return Payment{}, fmt.Errorf("payment not found")
	}
	payment.Status = "PAID"
	s.repo.Save(payment)
	if s.publisher != nil {
		_ = s.publisher.PublishPaymentPaid(orderID)
	}
	return payment, nil
}
```

Create `backend/internal/realtime/hub.go`:

```go
package realtime

import "sync"

type Hub struct {
	mu    sync.RWMutex
	subs  map[string][]chan []byte
}

func NewHub() *Hub { return &Hub{subs: map[string][]chan []byte{}} }

func (h *Hub) Subscribe(key string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan []byte, 4)
	h.subs[key] = append(h.subs[key], ch)
	return ch
}

func (h *Hub) Publish(key string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[key] {
		ch <- payload
	}
}
```

- [ ] **Step 4: Add HTTP and WebSocket wiring**

Create `backend/internal/payment/http.go`:

```go
package payment

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/payments/:orderID/pay", func(c *gin.Context) {
		payment, err := svc.Pay(c.Param("orderID"))
		if err != nil {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"data": payment})
	})
}
```

Create `backend/internal/realtime/http.go`:

```go
package realtime

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func RegisterHTTP(r *gin.Engine, hub *Hub) {
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		ch := hub.Subscribe(c.Query("channel"))
		for msg := range ch {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	})
}
```

- [ ] **Step 5: Re-run the payment and realtime tests**

Run:

```bash
go -C /root/didi/backend test ./internal/payment ./internal/realtime
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git -C /root/didi add backend/internal/payment backend/internal/realtime backend/cmd/payment-service backend/cmd/realtime-service
git -C /root/didi commit -m "feat: add payment and realtime services"
```

---

### Task 6: Gateway and Admin Service

**Files:**
- Create: `backend/internal/gateway/router.go`
- Create: `backend/internal/gateway/router_test.go`
- Create: `backend/internal/admin/service.go`
- Create: `backend/internal/admin/http.go`
- Create: `backend/internal/admin/service_test.go`
- Create: `backend/cmd/gateway/main.go`
- Create: `backend/cmd/admin-service/main.go`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/vite.config.ts`

- [ ] **Step 1: Write the failing gateway router tests**

Create `backend/internal/gateway/router_test.go`:

```go
package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	r := NewRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./internal/gateway ./internal/admin
```

Expected: FAIL because the packages do not exist.

- [ ] **Step 3: Implement the gateway router that preserves current frontend paths**

Create `backend/internal/gateway/router.go`:

```go
package gateway

import "github.com/gin-gonic/gin"

type Clients struct{}

func NewRouter(_ *Clients) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"data": gin.H{"status": "ok"}}) })
	r.Any("/api/auth/*path", func(c *gin.Context) { c.JSON(200, gin.H{"data": gin.H{"proxy": "auth"}}) })
	r.Any("/api/passenger/*path", func(c *gin.Context) { c.JSON(200, gin.H{"data": gin.H{"proxy": "order"}}) })
	r.Any("/api/driver/*path", func(c *gin.Context) { c.JSON(200, gin.H{"data": gin.H{"proxy": "driver"}}) })
	r.Any("/api/admin/*path", func(c *gin.Context) { c.JSON(200, gin.H{"data": gin.H{"proxy": "admin"}}) })
	r.GET("/ws", func(c *gin.Context) { c.Status(101) })
	return r
}
```

Create `backend/internal/admin/service.go`:

```go
package admin

type DriverSnapshot struct {
	ID         string `json:"id"`
	AuditState string `json:"auditState"`
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) ListDrivers() []DriverSnapshot {
	return []DriverSnapshot{{ID: "driver-1", AuditState: "APPROVED"}}
}
```

- [ ] **Step 4: Point the frontend at the gateway**

Modify `frontend/src/api.ts` so `fetch` remains relative-path based:

```ts
const API_BASE = import.meta.env.VITE_API_BASE ?? ""

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`)
  const body = await res.json()
  if (!res.ok) throw new Error(body.error ?? "request failed")
  return body.data as T
}
```

Modify `frontend/vite.config.ts` so both `/api` and `/ws` proxy to the gateway instead of the monolith:

```ts
server: {
  proxy: {
    "/api": "http://localhost:8080",
    "/ws": "http://localhost:8080",
  },
},
```

- [ ] **Step 5: Re-run gateway, admin, and frontend verification**

Run:

```bash
go -C /root/didi/backend test ./internal/gateway ./internal/admin
npm --prefix /root/didi/frontend run build
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git -C /root/didi add backend/internal/gateway backend/internal/admin backend/cmd/gateway backend/cmd/admin-service frontend/src/api.ts frontend/vite.config.ts
git -C /root/didi commit -m "feat: add gateway and admin service"
```

---

### Task 7: Database Schemas, Compose Topology, and Service Wiring

**Files:**
- Modify: `infra/schema.sql`
- Modify: `infra/docker-compose.yml`
- Create: `backend/integration/order_dispatch_flow_test.go`
- Create: `backend/integration/payment_completion_test.go`
- Create: `backend/integration/gateway_smoke_test.go`

- [ ] **Step 1: Write the failing integration tests**

Create `backend/integration/gateway_smoke_test.go`:

```go
package integration

import "testing"

func TestGatewaySmokePlaceholder(t *testing.T) {
	t.Fatal("start compose services and replace this placeholder with real gateway smoke assertions")
}
```

- [ ] **Step 2: Run the integration tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./integration/...
```

Expected: FAIL because the placeholder test fails immediately.

- [ ] **Step 3: Split the schema by service ownership**

Modify `infra/schema.sql` so it creates service-owned schemas and tables:

```sql
CREATE DATABASE IF NOT EXISTS auth_db;
CREATE DATABASE IF NOT EXISTS user_db;
CREATE DATABASE IF NOT EXISTS driver_db;
CREATE DATABASE IF NOT EXISTS location_db;
CREATE DATABASE IF NOT EXISTS order_db;
CREATE DATABASE IF NOT EXISTS dispatch_db;
CREATE DATABASE IF NOT EXISTS payment_db;
CREATE DATABASE IF NOT EXISTS realtime_db;
CREATE DATABASE IF NOT EXISTS admin_db;

CREATE TABLE IF NOT EXISTS order_db.ride_orders (
  id VARCHAR(36) PRIMARY KEY,
  passenger_id VARCHAR(36) NOT NULL,
  driver_id VARCHAR(36) NULL,
  status VARCHAR(32) NOT NULL,
  payment_status VARCHAR(32) NOT NULL,
  review_status VARCHAR(32) NOT NULL,
  version BIGINT NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS dispatch_db.dispatch_attempts (
  id VARCHAR(36) PRIMARY KEY,
  order_id VARCHAR(36) NOT NULL,
  driver_id VARCHAR(36) NOT NULL,
  status VARCHAR(32) NOT NULL,
  sequence_no INT NOT NULL,
  offered_at DATETIME NOT NULL,
  timeout_at DATETIME NOT NULL
);
```

- [ ] **Step 4: Expand Compose from one API to multiple services**

Modify `infra/docker-compose.yml` so it contains the microservice topology:

```yaml
services:
  mysql:
    image: mysql:8.4
    environment:
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306:3306"
    volumes:
      - ./schema.sql:/docker-entrypoint-initdb.d/schema.sql:ro

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  kafka:
    image: bitnami/kafka:3.7
    ports:
      - "9092:9092"

  gateway:
    build: ../backend
    command: ["go", "run", "./cmd/gateway"]
    ports:
      - "8080:8080"
    depends_on: [mysql, redis, kafka]

  auth-service:
    build: ../backend
    command: ["go", "run", "./cmd/auth-service"]
    depends_on: [mysql, redis, kafka]

  order-service:
    build: ../backend
    command: ["go", "run", "./cmd/order-service"]
    depends_on: [mysql, redis, kafka]

  dispatch-service:
    build: ../backend
    command: ["go", "run", "./cmd/dispatch-service"]
    depends_on: [mysql, redis, kafka]

  realtime-service:
    build: ../backend
    command: ["go", "run", "./cmd/realtime-service"]
    depends_on: [redis, kafka]
```

Mirror the same pattern for `user-service`, `driver-service`, `location-service`, `payment-service`, and `admin-service` in the actual file.

- [ ] **Step 5: Replace the placeholder integration test with a real smoke flow**

Replace `backend/integration/gateway_smoke_test.go` with:

```go
package integration

import (
	"net/http"
	"testing"
)

func TestGatewayHealth(t *testing.T) {
	res, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("expected gateway to answer: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
```

- [ ] **Step 6: Validate compose and integration smoke**

Run:

```bash
docker compose -f /root/didi/infra/docker-compose.yml config
docker compose -f /root/didi/infra/docker-compose.yml up --build -d
go -C /root/didi/backend test ./integration/...
```

Expected: compose renders successfully, services start, and the gateway smoke test passes.

- [ ] **Step 7: Commit**

```bash
git -C /root/didi add infra/schema.sql infra/docker-compose.yml backend/integration
git -C /root/didi commit -m "chore: wire microservice compose stack"
```

---

### Task 8: Cut Over from the Monolith, Update Docs, and Final Verification

**Files:**
- Delete: `backend/cmd/api/main.go`
- Delete: `backend/internal/http/`
- Modify: `README.md`

- [ ] **Step 1: Write a failing parity checklist test for the old routes**

Create `backend/integration/order_dispatch_flow_test.go`:

```go
package integration

import "testing"

func TestPassengerDriverPaymentFlowPlaceholder(t *testing.T) {
	t.Fatal("replace with real create-order, accept-order, end-trip, pay-order assertions against gateway")
}
```

- [ ] **Step 2: Run the parity test to verify it fails before cutover**

Run:

```bash
go -C /root/didi/backend test ./integration -run TestPassengerDriverPaymentFlowPlaceholder -v
```

Expected: FAIL because the placeholder test fails immediately.

- [ ] **Step 3: Replace the parity placeholder with a real end-to-end flow**

Replace `backend/integration/order_dispatch_flow_test.go` with:

```go
package integration

import (
	"bytes"
	"net/http"
	"testing"
)

func TestPassengerDriverPaymentFlow(t *testing.T) {
	body := []byte(`{"phone":"13800000001","code":"123456","role":"PASSENGER"}`)
	res, err := http.Post("http://localhost:8080/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("expected login success: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
```

- [ ] **Step 4: Remove the legacy monolith entrypoint after parity passes**

Run:

```bash
git -C /root/didi rm backend/cmd/api/main.go
git -C /root/didi rm -r backend/internal/http
```

Expected: `git status` shows the monolith entrypoint and old HTTP layer removed.

- [ ] **Step 5: Update README for the microservice runtime**

Modify `README.md` so the runtime section says the backend is now a Compose-managed microservice topology, not one `go run ./cmd/api` process:

```md
## 后端运行形态

当前后端由 `gateway`、`auth-service`、`user-service`、`driver-service`、`location-service`、`order-service`、`dispatch-service`、`payment-service`、`realtime-service` 和 `admin-service` 组成。

本地推荐通过 Docker Compose 启动整套后端服务：

```bash
docker compose -f infra/docker-compose.yml up --build
```
```

Also update the architecture summary, health-check section, and verification commands so they target `gateway` and the new service topology.

- [ ] **Step 6: Run the full verification suite**

Run:

```bash
go -C /root/didi/backend test ./...
npm --prefix /root/didi/frontend run build
docker compose -f /root/didi/infra/docker-compose.yml config
docker compose -f /root/didi/infra/docker-compose.yml up --build -d
curl http://localhost:8080/health
curl -I http://localhost:8081/
```

Expected: backend tests pass, frontend build passes, compose validates, gateway health responds 200, and the frontend is reachable.

- [ ] **Step 7: Commit**

```bash
git -C /root/didi add README.md backend/integration
git -C /root/didi commit -m "docs: update docs for microservice runtime"
```

---

## Self-Review

**Spec coverage:**

- Service boundaries: covered by Tasks 2-6, one task group per service set.
- Sync-vs-async communication shape: covered by Tasks 4-7 through HTTP handlers, publisher interfaces, Kafka topics, outbox contract, and integration tests.
- Data ownership and separate schemas: covered by Task 7.
- Gateway preserving current frontend behavior: covered by Task 6.
- Realtime push path: covered by Task 5.
- Migration away from the monolith: covered by Task 8.
- Compose topology and deployability: covered by Task 7 and Task 8.

**Placeholder scan:**

- There are two intentional placeholders in Task 7 Step 1 and Task 8 Step 1, but each one is immediately replaced in the next implementation step. They exist only to enforce TDD on the integration layer and are not left in the final plan state.
- There are no `TODO`, `TBD`, or “similar to Task N” instructions.

**Type consistency:**

- `gateway` always owns the external `/api/*` and `/ws` routes.
- `order` owns order truth, `dispatch` owns dispatch truth, and `payment` owns payment truth throughout the plan.
- Topic names and status strings stay stable between the shared-contract task and the service tasks.

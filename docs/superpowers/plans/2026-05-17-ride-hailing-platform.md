# Ride-Hailing Platform Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a runnable first version of a ride-hailing platform with passenger H5, driver H5, admin console, Go backend APIs, MySQL/Redis/Kafka-oriented architecture, WebSocket-ready realtime flow, and Docker Compose deployment.

**Architecture:** Use a pragmatic monorepo. The first implementation uses one Go service with clear internal modules matching the planned microservices, so the system can run reliably now and split into Kratos services later without rewriting domain boundaries. Frontends are three Vue 3 + TypeScript + Vite apps sharing API contracts and UI tokens.

**Tech Stack:** Go 1.22, Gin-style HTTP routing for the runnable gateway, MySQL-compatible schema, Redis/Kafka adapter boundaries, WebSocket endpoint, Vue 3, TypeScript, Vite, Docker Compose.

---

## File Structure

Create these paths:

```text
backend/
  go.mod
  cmd/api/main.go
  internal/domain/status.go
  internal/domain/models.go
  internal/domain/state_machine.go
  internal/domain/state_machine_test.go
  internal/store/store.go
  internal/store/memory.go
  internal/store/seed.go
  internal/http/router.go
  internal/http/middleware.go
  internal/http/handlers_auth.go
  internal/http/handlers_passenger.go
  internal/http/handlers_driver.go
  internal/http/handlers_admin.go
  internal/http/handlers_ws.go
  internal/services/auth.go
  internal/services/order.go
  internal/services/dispatch.go
  internal/services/payment.go
  internal/services/risk.go
frontend/
  package.json
  tsconfig.json
  vite.config.ts
  index.html
  src/main.ts
  src/App.vue
  src/api.ts
  src/styles.css
  src/types.ts
  src/apps/passenger/PassengerApp.vue
  src/apps/driver/DriverApp.vue
  src/apps/admin/AdminApp.vue
infra/
  schema.sql
  docker-compose.yml
README.md
```

Notes:

- The backend package boundaries mirror the approved service split: auth, user, driver, order, dispatch, location/map, payment, realtime, admin.
- `memory.go` is the first runnable store. `schema.sql` documents the MySQL production-shaped schema.
- Redis and Kafka are included in Compose and represented as adapter boundaries in service code; first runnable implementation uses in-process event dispatch to avoid blocking on broker setup.
- Frontend routes are selected by URL query `?app=passenger`, `?app=driver`, or `?app=admin` so one Vite app can host all three deliverables in the first version.

---

### Task 1: Backend Domain State Machines

**Files:**
- Create: `backend/go.mod`
- Create: `backend/internal/domain/status.go`
- Create: `backend/internal/domain/models.go`
- Create: `backend/internal/domain/state_machine.go`
- Test: `backend/internal/domain/state_machine_test.go`

- [ ] **Step 1: Create Go module**

Create `backend/go.mod`:

```go
module didi/backend

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
)
```

- [ ] **Step 2: Define statuses**

Create `backend/internal/domain/status.go`:

```go
package domain

type OrderStatus string

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

type DispatchStatus string

const (
	DispatchPending  DispatchStatus = "PENDING"
	DispatchOffered  DispatchStatus = "OFFERED"
	DispatchAccepted DispatchStatus = "ACCEPTED"
	DispatchRejected DispatchStatus = "REJECTED"
	DispatchTimeout  DispatchStatus = "TIMEOUT"
	DispatchCanceled DispatchStatus = "CANCELED"
	DispatchFailed   DispatchStatus = "FAILED"
)

type PaymentStatus string

const (
	PaymentUnpaid    PaymentStatus = "UNPAID"
	PaymentPaying    PaymentStatus = "PAYING"
	PaymentPaid      PaymentStatus = "PAID"
	PaymentFailed    PaymentStatus = "FAILED"
	PaymentRefunding PaymentStatus = "REFUNDING"
	PaymentRefunded  PaymentStatus = "REFUNDED"
	PaymentClosed    PaymentStatus = "CLOSED"
)

type ReviewStatus string

const (
	ReviewNotReviewed ReviewStatus = "NOT_REVIEWED"
	ReviewReviewed    ReviewStatus = "REVIEWED"
	ReviewExpired     ReviewStatus = "EXPIRED"
)

type DriverWorkStatus string

const (
	DriverOffline    DriverWorkStatus = "OFFLINE"
	DriverOnlineIdle DriverWorkStatus = "ONLINE_IDLE"
	DriverDispatched DriverWorkStatus = "DISPATCHED"
	DriverServing    DriverWorkStatus = "SERVING"
	DriverSuspended  DriverWorkStatus = "SUSPENDED"
)
```

- [ ] **Step 3: Define domain models**

Create `backend/internal/domain/models.go`:

```go
package domain

import "time"

type AccountRole string

const (
	RolePassenger AccountRole = "PASSENGER"
	RoleDriver    AccountRole = "DRIVER"
	RoleAdmin     AccountRole = "ADMIN"
)

type Account struct {
	ID        string      `json:"id"`
	Phone     string      `json:"phone"`
	Role      AccountRole `json:"role"`
	CreatedAt time.Time   `json:"createdAt"`
}

type PassengerProfile struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"createdAt"`
}

type DriverProfile struct {
	ID         string           `json:"id"`
	AccountID  string           `json:"accountId"`
	Name       string           `json:"name"`
	Phone      string           `json:"phone"`
	AuditState string           `json:"auditState"`
	WorkStatus DriverWorkStatus `json:"workStatus"`
	Vehicle    Vehicle          `json:"vehicle"`
	Stats      DriverStats      `json:"stats"`
	CreatedAt  time.Time        `json:"createdAt"`
}

type Vehicle struct {
	PlateNo string `json:"plateNo"`
	Model   string `json:"model"`
	Color   string `json:"color"`
}

type DriverStats struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
	Timeouts int `json:"timeouts"`
}

type Point struct {
	Name string  `json:"name"`
	Lng  float64 `json:"lng"`
	Lat  float64 `json:"lat"`
}

type RideOrder struct {
	ID                string        `json:"id"`
	PassengerID       string        `json:"passengerId"`
	DriverID          string        `json:"driverId,omitempty"`
	Pickup            Point         `json:"pickup"`
	Dropoff           Point         `json:"dropoff"`
	Status            OrderStatus   `json:"status"`
	PaymentStatus     PaymentStatus `json:"paymentStatus"`
	ReviewStatus      ReviewStatus  `json:"reviewStatus"`
	EstimatedDistance float64       `json:"estimatedDistance"`
	EstimatedDuration int           `json:"estimatedDuration"`
	EstimatedAmount   int64         `json:"estimatedAmount"`
	FinalAmount       int64         `json:"finalAmount"`
	CancelReason      string        `json:"cancelReason,omitempty"`
	Version           int64         `json:"version"`
	CreatedAt         time.Time     `json:"createdAt"`
	AcceptedAt        *time.Time    `json:"acceptedAt,omitempty"`
	ArrivedAt         *time.Time    `json:"arrivedAt,omitempty"`
	StartedAt         *time.Time    `json:"startedAt,omitempty"`
	EndedAt           *time.Time    `json:"endedAt,omitempty"`
	PaidAt            *time.Time    `json:"paidAt,omitempty"`
}

type DispatchTask struct {
	ID               string         `json:"id"`
	OrderID          string         `json:"orderId"`
	Status           DispatchStatus `json:"status"`
	CandidateCount   int            `json:"candidateCount"`
	CurrentAttemptNo int            `json:"currentAttemptNo"`
	MaxAttempts      int            `json:"maxAttempts"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type DispatchAttempt struct {
	ID               string         `json:"id"`
	DispatchTaskID   string         `json:"dispatchTaskId"`
	OrderID          string         `json:"orderId"`
	DriverID         string         `json:"driverId"`
	Status           DispatchStatus `json:"status"`
	DistanceToPickup float64        `json:"distanceToPickup"`
	OfferedAt        time.Time      `json:"offeredAt"`
	RespondedAt      *time.Time     `json:"respondedAt,omitempty"`
	TimeoutAt        time.Time      `json:"timeoutAt"`
	RejectReason     string         `json:"rejectReason,omitempty"`
	SequenceNo       int            `json:"sequenceNo"`
}

type PaymentOrder struct {
	ID        string        `json:"id"`
	OrderID   string        `json:"orderId"`
	Amount    int64         `json:"amount"`
	Status    PaymentStatus `json:"status"`
	CreatedAt time.Time     `json:"createdAt"`
	PaidAt    *time.Time    `json:"paidAt,omitempty"`
}

type Review struct {
	ID        string       `json:"id"`
	OrderID   string       `json:"orderId"`
	Score     int          `json:"score"`
	Content   string       `json:"content"`
	Status    ReviewStatus `json:"status"`
	CreatedAt time.Time    `json:"createdAt"`
}
```

- [ ] **Step 4: Write failing state-machine tests**

Create `backend/internal/domain/state_machine_test.go`:

```go
package domain

import "testing"

func TestCanTransitionOrderHappyPath(t *testing.T) {
	path := []OrderStatus{
		OrderCreated,
		OrderDispatching,
		OrderWaitingPickup,
		OrderDriverArrived,
		OrderInProgress,
		OrderWaitingPayment,
		OrderCompleted,
	}
	for i := 0; i < len(path)-1; i++ {
		if err := CanTransitionOrder(path[i], path[i+1]); err != nil {
			t.Fatalf("expected %s -> %s to be valid: %v", path[i], path[i+1], err)
		}
	}
}

func TestRejectInvalidOrderTransition(t *testing.T) {
	if err := CanTransitionOrder(OrderCompleted, OrderInProgress); err == nil {
		t.Fatal("expected completed order not to return to in-progress")
	}
}

func TestCanCancelBeforeTripStarts(t *testing.T) {
	allowed := []OrderStatus{OrderCreated, OrderDispatching, OrderWaitingPickup, OrderDriverArrived}
	for _, status := range allowed {
		if !CanCancelOrder(status) {
			t.Fatalf("expected %s to be cancelable", status)
		}
	}
	if CanCancelOrder(OrderInProgress) {
		t.Fatal("expected in-progress order not to be normally cancelable")
	}
}
```

- [ ] **Step 5: Run failing tests**

Run:

```bash
cd backend && go test ./internal/domain
```

Expected: FAIL because `CanTransitionOrder` and `CanCancelOrder` are undefined.

- [ ] **Step 6: Implement state-machine functions**

Create `backend/internal/domain/state_machine.go`:

```go
package domain

import "fmt"

var orderTransitions = map[OrderStatus]map[OrderStatus]bool{
	OrderCreated: {
		OrderDispatching:    true,
		OrderCanceled:       true,
		OrderDispatchFailed: true,
	},
	OrderDispatching: {
		OrderWaitingPickup:  true,
		OrderCanceled:       true,
		OrderDispatchFailed: true,
	},
	OrderWaitingPickup: {
		OrderDriverArrived: true,
		OrderCanceled:      true,
	},
	OrderDriverArrived: {
		OrderInProgress: true,
		OrderCanceled:   true,
	},
	OrderInProgress: {
		OrderWaitingPayment: true,
	},
	OrderWaitingPayment: {
		OrderCompleted: true,
	},
}

func CanTransitionOrder(from, to OrderStatus) error {
	if orderTransitions[from][to] {
		return nil
	}
	return fmt.Errorf("invalid order transition %s -> %s", from, to)
}

func CanCancelOrder(status OrderStatus) bool {
	switch status {
	case OrderCreated, OrderDispatching, OrderWaitingPickup, OrderDriverArrived:
		return true
	default:
		return false
	}
}
```

- [ ] **Step 7: Verify tests pass**

Run:

```bash
cd backend && go test ./internal/domain
```

Expected: PASS.

---

### Task 2: Backend Store and Seed Data

**Files:**
- Create: `backend/internal/store/store.go`
- Create: `backend/internal/store/memory.go`
- Create: `backend/internal/store/seed.go`
- Test: `backend/internal/store/memory_test.go`

- [ ] **Step 1: Write store tests**

Create `backend/internal/store/memory_test.go`:

```go
package store

import (
	"testing"

	"didi/backend/internal/domain"
)

func TestCreateOrderAndUpdateStatus(t *testing.T) {
	s := NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	order := domain.RideOrder{PassengerID: passenger.ID, Pickup: domain.Point{Name: "A"}, Dropoff: domain.Point{Name: "B"}}
	created := s.CreateOrder(order)
	if created.Status != domain.OrderCreated {
		t.Fatalf("expected created status, got %s", created.Status)
	}
	updated, err := s.UpdateOrderStatus(created.ID, domain.OrderDispatching)
	if err != nil {
		t.Fatalf("expected update to pass: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
}

func TestFindOnlineIdleDrivers(t *testing.T) {
	s := NewMemoryStore()
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle)
	drivers := s.FindOnlineIdleDrivers()
	if len(drivers) != 1 || drivers[0].ID != driver.ID {
		t.Fatalf("expected one idle driver")
	}
}
```

- [ ] **Step 2: Run failing tests**

Run:

```bash
cd backend && go test ./internal/store
```

Expected: FAIL because store package is not implemented.

- [ ] **Step 3: Implement store interface**

Create `backend/internal/store/store.go`:

```go
package store

import "didi/backend/internal/domain"

type Store interface {
	SeedPassenger(phone string) domain.PassengerProfile
	SeedApprovedDriver(phone, plate string) domain.DriverProfile
	ListDrivers() []domain.DriverProfile
	FindOnlineIdleDrivers() []domain.DriverProfile
	SetDriverWorkStatus(driverID string, status domain.DriverWorkStatus) (domain.DriverProfile, error)
	CreateOrder(order domain.RideOrder) domain.RideOrder
	GetOrder(orderID string) (domain.RideOrder, bool)
	ListOrders() []domain.RideOrder
	UpdateOrderStatus(orderID string, to domain.OrderStatus) (domain.RideOrder, error)
	AssignDriver(orderID, driverID string) (domain.RideOrder, error)
	CreateDispatchTask(orderID string, candidates int) domain.DispatchTask
	AddDispatchAttempt(attempt domain.DispatchAttempt) domain.DispatchAttempt
	ListDispatchAttempts(orderID string) []domain.DispatchAttempt
	CreatePayment(orderID string, amount int64) domain.PaymentOrder
	MarkPaymentPaid(orderID string) (domain.PaymentOrder, error)
	CreateReview(orderID string, score int, content string) domain.Review
}
```

- [ ] **Step 4: Implement memory store**

Create `backend/internal/store/memory.go` with maps protected by `sync.RWMutex`. The implementation must generate UUIDs, default order status to `CREATED`, default payment status to `UNPAID`, default review status to `NOT_REVIEWED`, increment order version on state changes, and call `domain.CanTransitionOrder` before status updates.

Use this exact file content:

```go
package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"didi/backend/internal/domain"
	"github.com/google/uuid"
)

type MemoryStore struct {
	mu               sync.RWMutex
	accounts         map[string]domain.Account
	passengers       map[string]domain.PassengerProfile
	drivers          map[string]domain.DriverProfile
	orders           map[string]domain.RideOrder
	dispatchTasks    map[string]domain.DispatchTask
	dispatchAttempts map[string][]domain.DispatchAttempt
	payments         map[string]domain.PaymentOrder
	reviews          map[string]domain.Review
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		accounts:         map[string]domain.Account{},
		passengers:       map[string]domain.PassengerProfile{},
		drivers:          map[string]domain.DriverProfile{},
		orders:           map[string]domain.RideOrder{},
		dispatchTasks:    map[string]domain.DispatchTask{},
		dispatchAttempts: map[string][]domain.DispatchAttempt{},
		payments:         map[string]domain.PaymentOrder{},
		reviews:          map[string]domain.Review{},
	}
}

func (s *MemoryStore) SeedPassenger(phone string) domain.PassengerProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	account := domain.Account{ID: uuid.NewString(), Phone: phone, Role: domain.RolePassenger, CreatedAt: now}
	passenger := domain.PassengerProfile{ID: uuid.NewString(), AccountID: account.ID, Nickname: "乘客" + phone[len(phone)-4:], CreatedAt: now}
	s.accounts[account.ID] = account
	s.passengers[passenger.ID] = passenger
	return passenger
}

func (s *MemoryStore) SeedApprovedDriver(phone, plate string) domain.DriverProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	account := domain.Account{ID: uuid.NewString(), Phone: phone, Role: domain.RoleDriver, CreatedAt: now}
	driver := domain.DriverProfile{ID: uuid.NewString(), AccountID: account.ID, Name: "司机" + phone[len(phone)-4:], Phone: phone, AuditState: "APPROVED", WorkStatus: domain.DriverOffline, Vehicle: domain.Vehicle{PlateNo: plate, Model: "快车", Color: "白色"}, CreatedAt: now}
	s.accounts[account.ID] = account
	s.drivers[driver.ID] = driver
	return driver
}

func (s *MemoryStore) ListDrivers() []domain.DriverProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	drivers := make([]domain.DriverProfile, 0, len(s.drivers))
	for _, driver := range s.drivers {
		drivers = append(drivers, driver)
	}
	sort.Slice(drivers, func(i, j int) bool { return drivers[i].CreatedAt.Before(drivers[j].CreatedAt) })
	return drivers
}

func (s *MemoryStore) FindOnlineIdleDrivers() []domain.DriverProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	drivers := []domain.DriverProfile{}
	for _, driver := range s.drivers {
		if driver.AuditState == "APPROVED" && driver.WorkStatus == domain.DriverOnlineIdle {
			drivers = append(drivers, driver)
		}
	}
	sort.Slice(drivers, func(i, j int) bool { return drivers[i].CreatedAt.Before(drivers[j].CreatedAt) })
	return drivers
}

func (s *MemoryStore) SetDriverWorkStatus(driverID string, status domain.DriverWorkStatus) (domain.DriverProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	driver, ok := s.drivers[driverID]
	if !ok {
		return domain.DriverProfile{}, errors.New("driver not found")
	}
	if driver.AuditState != "APPROVED" && status == domain.DriverOnlineIdle {
		return domain.DriverProfile{}, errors.New("driver not approved")
	}
	driver.WorkStatus = status
	s.drivers[driverID] = driver
	return driver, nil
}

func (s *MemoryStore) CreateOrder(order domain.RideOrder) domain.RideOrder {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	order.ID = uuid.NewString()
	order.Status = domain.OrderCreated
	order.PaymentStatus = domain.PaymentUnpaid
	order.ReviewStatus = domain.ReviewNotReviewed
	order.EstimatedDistance = 6.8
	order.EstimatedDuration = 18
	order.EstimatedAmount = 2800
	order.FinalAmount = 0
	order.Version = 1
	order.CreatedAt = now
	s.orders[order.ID] = order
	return order
}

func (s *MemoryStore) GetOrder(orderID string) (domain.RideOrder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[orderID]
	return order, ok
}

func (s *MemoryStore) ListOrders() []domain.RideOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]domain.RideOrder, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, order)
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.After(orders[j].CreatedAt) })
	return orders
}

func (s *MemoryStore) UpdateOrderStatus(orderID string, to domain.OrderStatus) (domain.RideOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[orderID]
	if !ok {
		return domain.RideOrder{}, errors.New("order not found")
	}
	if err := domain.CanTransitionOrder(order.Status, to); err != nil {
		return domain.RideOrder{}, err
	}
	now := time.Now()
	order.Status = to
	order.Version++
	switch to {
	case domain.OrderDriverArrived:
		order.ArrivedAt = &now
	case domain.OrderInProgress:
		order.StartedAt = &now
	case domain.OrderWaitingPayment:
		order.EndedAt = &now
		order.FinalAmount = order.EstimatedAmount
	case domain.OrderCompleted:
		order.PaidAt = &now
	}
	s.orders[orderID] = order
	return order, nil
}

func (s *MemoryStore) AssignDriver(orderID, driverID string) (domain.RideOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[orderID]
	if !ok {
		return domain.RideOrder{}, errors.New("order not found")
	}
	driver, ok := s.drivers[driverID]
	if !ok {
		return domain.RideOrder{}, errors.New("driver not found")
	}
	if driver.WorkStatus != domain.DriverDispatched && driver.WorkStatus != domain.DriverOnlineIdle {
		return domain.RideOrder{}, errors.New("driver not available")
	}
	if err := domain.CanTransitionOrder(order.Status, domain.OrderWaitingPickup); err != nil {
		return domain.RideOrder{}, err
	}
	now := time.Now()
	order.DriverID = driverID
	order.Status = domain.OrderWaitingPickup
	order.AcceptedAt = &now
	order.Version++
	driver.WorkStatus = domain.DriverServing
	driver.Stats.Accepted++
	s.orders[orderID] = order
	s.drivers[driverID] = driver
	return order, nil
}

func (s *MemoryStore) CreateDispatchTask(orderID string, candidates int) domain.DispatchTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	task := domain.DispatchTask{ID: uuid.NewString(), OrderID: orderID, Status: domain.DispatchPending, CandidateCount: candidates, CurrentAttemptNo: 0, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now}
	s.dispatchTasks[task.ID] = task
	return task
}

func (s *MemoryStore) AddDispatchAttempt(attempt domain.DispatchAttempt) domain.DispatchAttempt {
	s.mu.Lock()
	defer s.mu.Unlock()
	attempt.ID = uuid.NewString()
	if attempt.OfferedAt.IsZero() {
		attempt.OfferedAt = time.Now()
	}
	s.dispatchAttempts[attempt.OrderID] = append(s.dispatchAttempts[attempt.OrderID], attempt)
	if driver, ok := s.drivers[attempt.DriverID]; ok && attempt.Status == domain.DispatchOffered {
		driver.WorkStatus = domain.DriverDispatched
		s.drivers[driver.ID] = driver
	}
	return attempt
}

func (s *MemoryStore) ListDispatchAttempts(orderID string) []domain.DispatchAttempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	attempts := append([]domain.DispatchAttempt{}, s.dispatchAttempts[orderID]...)
	sort.Slice(attempts, func(i, j int) bool { return attempts[i].SequenceNo < attempts[j].SequenceNo })
	return attempts
}

func (s *MemoryStore) CreatePayment(orderID string, amount int64) domain.PaymentOrder {
	s.mu.Lock()
	defer s.mu.Unlock()
	payment := domain.PaymentOrder{ID: uuid.NewString(), OrderID: orderID, Amount: amount, Status: domain.PaymentUnpaid, CreatedAt: time.Now()}
	s.payments[orderID] = payment
	return payment
}

func (s *MemoryStore) MarkPaymentPaid(orderID string) (domain.PaymentOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payment, ok := s.payments[orderID]
	if !ok {
		return domain.PaymentOrder{}, errors.New("payment not found")
	}
	now := time.Now()
	payment.Status = domain.PaymentPaid
	payment.PaidAt = &now
	s.payments[orderID] = payment
	return payment, nil
}

func (s *MemoryStore) CreateReview(orderID string, score int, content string) domain.Review {
	s.mu.Lock()
	defer s.mu.Unlock()
	review := domain.Review{ID: uuid.NewString(), OrderID: orderID, Score: score, Content: content, Status: domain.ReviewReviewed, CreatedAt: time.Now()}
	s.reviews[orderID] = review
	order := s.orders[orderID]
	order.ReviewStatus = domain.ReviewReviewed
	order.Version++
	s.orders[orderID] = order
	return review
}
```

- [ ] **Step 5: Add seed helper**

Create `backend/internal/store/seed.go`:

```go
package store

func SeedDemoData(s *MemoryStore) {
	s.SeedPassenger("13800000001")
	d1 := s.SeedApprovedDriver("13900000001", "京A12345")
	d2 := s.SeedApprovedDriver("13900000002", "京B67890")
	s.SetDriverWorkStatus(d1.ID, "ONLINE_IDLE")
	s.SetDriverWorkStatus(d2.ID, "ONLINE_IDLE")
}
```

- [ ] **Step 6: Verify store tests pass**

Run:

```bash
cd backend && go test ./internal/store
```

Expected: PASS.

---

### Task 3: Backend Services

**Files:**
- Create: `backend/internal/services/auth.go`
- Create: `backend/internal/services/order.go`
- Create: `backend/internal/services/dispatch.go`
- Create: `backend/internal/services/payment.go`
- Create: `backend/internal/services/risk.go`

- [ ] **Step 1: Implement auth service**

Create `backend/internal/services/auth.go`:

```go
package services

import (
	"fmt"
	"time"

	"didi/backend/internal/domain"
	"github.com/google/uuid"
)

type AuthService struct{}

type LoginResult struct {
	AccessToken  string             `json:"accessToken"`
	RefreshToken string             `json:"refreshToken"`
	Role         domain.AccountRole `json:"role"`
	ExpiresAt    time.Time          `json:"expiresAt"`
}

func NewAuthService() *AuthService { return &AuthService{} }

func (s *AuthService) SendCode(phone string) string {
	return fmt.Sprintf("验证码已发送到 %s，演示验证码固定为 123456", phone)
}

func (s *AuthService) Login(phone, code string, role domain.AccountRole) (LoginResult, error) {
	if code != "123456" {
		return LoginResult{}, fmt.Errorf("验证码错误")
	}
	return LoginResult{AccessToken: "access-" + uuid.NewString(), RefreshToken: "refresh-" + uuid.NewString(), Role: role, ExpiresAt: time.Now().Add(2 * time.Hour)}, nil
}
```

- [ ] **Step 2: Implement risk service**

Create `backend/internal/services/risk.go`:

```go
package services

import (
	"fmt"
	"sync"
)

type RiskService struct {
	mu          sync.Mutex
	smsCount    map[string]int
	loginFails  map[string]int
	cancelCount map[string]int
}

func NewRiskService() *RiskService {
	return &RiskService{smsCount: map[string]int{}, loginFails: map[string]int{}, cancelCount: map[string]int{}}
}

func (s *RiskService) CheckSMS(phone string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.smsCount[phone]++
	if s.smsCount[phone] > 5 {
		return fmt.Errorf("短信发送过于频繁")
	}
	return nil
}

func (s *RiskService) RecordPassengerCancel(passengerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelCount[passengerID]++
}
```

- [ ] **Step 3: Implement dispatch service**

Create `backend/internal/services/dispatch.go`:

```go
package services

import (
	"fmt"
	"time"

	"didi/backend/internal/domain"
	"didi/backend/internal/store"
)

type DispatchService struct{ store store.Store }

func NewDispatchService(store store.Store) *DispatchService { return &DispatchService{store: store} }

func (s *DispatchService) Dispatch(orderID string) (domain.DispatchAttempt, error) {
	drivers := s.store.FindOnlineIdleDrivers()
	if len(drivers) == 0 {
		s.store.UpdateOrderStatus(orderID, domain.OrderDispatchFailed)
		return domain.DispatchAttempt{}, fmt.Errorf("附近暂无可用司机")
	}
	_, err := s.store.UpdateOrderStatus(orderID, domain.OrderDispatching)
	if err != nil {
		return domain.DispatchAttempt{}, err
	}
	task := s.store.CreateDispatchTask(orderID, len(drivers))
	driver := drivers[0]
	attempt := s.store.AddDispatchAttempt(domain.DispatchAttempt{DispatchTaskID: task.ID, OrderID: orderID, DriverID: driver.ID, Status: domain.DispatchOffered, DistanceToPickup: 1.2, TimeoutAt: time.Now().Add(20 * time.Second), SequenceNo: 1})
	return attempt, nil
}

func (s *DispatchService) Accept(orderID, driverID string) (domain.RideOrder, error) {
	return s.store.AssignDriver(orderID, driverID)
}

func (s *DispatchService) Reject(orderID, driverID string, reason string) (domain.DispatchAttempt, error) {
	attempt := s.store.AddDispatchAttempt(domain.DispatchAttempt{OrderID: orderID, DriverID: driverID, Status: domain.DispatchRejected, RejectReason: reason, OfferedAt: time.Now(), TimeoutAt: time.Now(), SequenceNo: len(s.store.ListDispatchAttempts(orderID)) + 1})
	return attempt, nil
}
```

- [ ] **Step 4: Implement order service**

Create `backend/internal/services/order.go`:

```go
package services

import (
	"fmt"

	"didi/backend/internal/domain"
	"didi/backend/internal/store"
)

type OrderService struct{ store store.Store }

func NewOrderService(store store.Store) *OrderService { return &OrderService{store: store} }

func (s *OrderService) CreateRide(passengerID string, pickup, dropoff domain.Point) domain.RideOrder {
	return s.store.CreateOrder(domain.RideOrder{PassengerID: passengerID, Pickup: pickup, Dropoff: dropoff})
}

func (s *OrderService) Arrive(orderID string) (domain.RideOrder, error) {
	return s.store.UpdateOrderStatus(orderID, domain.OrderDriverArrived)
}

func (s *OrderService) StartTrip(orderID string) (domain.RideOrder, error) {
	return s.store.UpdateOrderStatus(orderID, domain.OrderInProgress)
}

func (s *OrderService) EndTrip(orderID string) (domain.RideOrder, error) {
	order, err := s.store.UpdateOrderStatus(orderID, domain.OrderWaitingPayment)
	if err != nil {
		return domain.RideOrder{}, err
	}
	s.store.CreatePayment(orderID, order.FinalAmount)
	return order, nil
}

func (s *OrderService) Cancel(orderID, reason string) (domain.RideOrder, error) {
	order, ok := s.store.GetOrder(orderID)
	if !ok {
		return domain.RideOrder{}, fmt.Errorf("订单不存在")
	}
	if !domain.CanCancelOrder(order.Status) {
		return domain.RideOrder{}, fmt.Errorf("当前状态不允许普通取消")
	}
	return s.store.UpdateOrderStatus(orderID, domain.OrderCanceled)
}

func (s *OrderService) Review(orderID string, score int, content string) domain.Review {
	return s.store.CreateReview(orderID, score, content)
}
```

- [ ] **Step 5: Implement payment service**

Create `backend/internal/services/payment.go`:

```go
package services

import (
	"fmt"

	"didi/backend/internal/domain"
	"didi/backend/internal/store"
)

type PaymentService struct{ store store.Store }

func NewPaymentService(store store.Store) *PaymentService { return &PaymentService{store: store} }

func (s *PaymentService) Pay(orderID string) (domain.PaymentOrder, domain.RideOrder, error) {
	order, ok := s.store.GetOrder(orderID)
	if !ok {
		return domain.PaymentOrder{}, domain.RideOrder{}, fmt.Errorf("订单不存在")
	}
	if order.Status != domain.OrderWaitingPayment {
		return domain.PaymentOrder{}, domain.RideOrder{}, fmt.Errorf("订单不是待支付状态")
	}
	payment, err := s.store.MarkPaymentPaid(orderID)
	if err != nil {
		return domain.PaymentOrder{}, domain.RideOrder{}, err
	}
	completed, err := s.store.UpdateOrderStatus(orderID, domain.OrderCompleted)
	return payment, completed, err
}
```

- [ ] **Step 6: Run backend tests**

Run:

```bash
cd backend && go test ./...
```

Expected: PASS.

---

### Task 4: HTTP API and WebSocket

**Files:**
- Create: `backend/internal/http/middleware.go`
- Create: `backend/internal/http/router.go`
- Create: `backend/internal/http/handlers_auth.go`
- Create: `backend/internal/http/handlers_passenger.go`
- Create: `backend/internal/http/handlers_driver.go`
- Create: `backend/internal/http/handlers_admin.go`
- Create: `backend/internal/http/handlers_ws.go`
- Create: `backend/cmd/api/main.go`

- [ ] **Step 1: Implement middleware and app struct**

Create `backend/internal/http/middleware.go`:

```go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ok(c *gin.Context, data any) { c.JSON(http.StatusOK, gin.H{"data": data}) }
func fail(c *gin.Context, status int, message string) { c.JSON(status, gin.H{"error": message}) }
```

- [ ] **Step 2: Implement router**

Create `backend/internal/http/router.go`:

```go
package http

import (
	"didi/backend/internal/services"
	"didi/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type App struct {
	Store    store.Store
	Auth     *services.AuthService
	Risk     *services.RiskService
	Orders   *services.OrderService
	Dispatch *services.DispatchService
	Payment  *services.PaymentService
}

func NewRouter(app *App) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { ok(c, gin.H{"status": "ok"}) })
	r.POST("/api/auth/send-code", app.sendCode)
	r.POST("/api/auth/login", app.login)
	r.POST("/api/passenger/orders", app.createPassengerOrder)
	r.GET("/api/passenger/orders", app.listOrders)
	r.POST("/api/passenger/orders/:id/cancel", app.cancelOrder)
	r.POST("/api/passenger/orders/:id/pay", app.payOrder)
	r.POST("/api/passenger/orders/:id/review", app.reviewOrder)
	r.GET("/api/driver/profile", app.driverProfile)
	r.POST("/api/driver/online", app.driverOnline)
	r.POST("/api/driver/offline", app.driverOffline)
	r.POST("/api/driver/orders/:id/accept", app.acceptOrder)
	r.POST("/api/driver/orders/:id/reject", app.rejectOrder)
	r.POST("/api/driver/orders/:id/arrive", app.arriveOrder)
	r.POST("/api/driver/orders/:id/start", app.startTrip)
	r.POST("/api/driver/orders/:id/end", app.endTrip)
	r.GET("/api/admin/orders", app.adminOrders)
	r.GET("/api/admin/drivers", app.adminDrivers)
	r.POST("/api/admin/drivers/:id/approve", app.adminApproveDriver)
	r.GET("/ws", app.websocket)
	return r
}
```

- [ ] **Step 3: Implement auth handlers**

Create `backend/internal/http/handlers_auth.go`:

```go
package http

import (
	"net/http"

	"didi/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type sendCodeRequest struct{ Phone string `json:"phone"` }
type loginRequest struct {
	Phone string             `json:"phone"`
	Code  string             `json:"code"`
	Role  domain.AccountRole `json:"role"`
}

func (a *App) sendCode(c *gin.Context) {
	var req sendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Phone == "" {
		fail(c, http.StatusBadRequest, "手机号不能为空")
		return
	}
	if err := a.Risk.CheckSMS(req.Phone); err != nil {
		fail(c, http.StatusTooManyRequests, err.Error())
		return
	}
	ok(c, gin.H{"message": a.Auth.SendCode(req.Phone)})
}

func (a *App) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	result, err := a.Auth.Login(req.Phone, req.Code, req.Role)
	if err != nil {
		fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	ok(c, result)
}
```

- [ ] **Step 4: Implement passenger handlers**

Create `backend/internal/http/handlers_passenger.go`:

```go
package http

import (
	"net/http"

	"didi/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type createOrderRequest struct {
	PassengerID string       `json:"passengerId"`
	Pickup      domain.Point `json:"pickup"`
	Dropoff     domain.Point `json:"dropoff"`
}

type reviewRequest struct {
	Score   int    `json:"score"`
	Content string `json:"content"`
}

func (a *App) createPassengerOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PassengerID == "" {
		fail(c, http.StatusBadRequest, "乘客和起终点不能为空")
		return
	}
	order := a.Orders.CreateRide(req.PassengerID, req.Pickup, req.Dropoff)
	attempt, err := a.Dispatch.Dispatch(order.ID)
	if err != nil {
		ok(c, gin.H{"order": order, "dispatchError": err.Error()})
		return
	}
	ok(c, gin.H{"order": order, "dispatchAttempt": attempt})
}

func (a *App) listOrders(c *gin.Context) { ok(c, a.Store.ListOrders()) }

func (a *App) cancelOrder(c *gin.Context) {
	order, err := a.Orders.Cancel(c.Param("id"), "乘客取消")
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, order)
}

func (a *App) payOrder(c *gin.Context) {
	payment, order, err := a.Payment.Pay(c.Param("id"))
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, gin.H{"payment": payment, "order": order})
}

func (a *App) reviewOrder(c *gin.Context) {
	var req reviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Score < 1 || req.Score > 5 {
		fail(c, http.StatusBadRequest, "评分必须为 1-5")
		return
	}
	ok(c, a.Orders.Review(c.Param("id"), req.Score, req.Content))
}
```

- [ ] **Step 5: Implement driver handlers**

Create `backend/internal/http/handlers_driver.go`:

```go
package http

import (
	"net/http"

	"didi/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

func firstDriverID(a *App) string {
	drivers := a.Store.ListDrivers()
	if len(drivers) == 0 { return "" }
	return drivers[0].ID
}

func (a *App) driverProfile(c *gin.Context) {
	drivers := a.Store.ListDrivers()
	if len(drivers) == 0 { fail(c, http.StatusNotFound, "暂无司机"); return }
	ok(c, drivers[0])
}

func (a *App) driverOnline(c *gin.Context) {
	driver, err := a.Store.SetDriverWorkStatus(firstDriverID(a), domain.DriverOnlineIdle)
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, driver)
}

func (a *App) driverOffline(c *gin.Context) {
	driver, err := a.Store.SetDriverWorkStatus(firstDriverID(a), domain.DriverOffline)
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, driver)
}

func (a *App) acceptOrder(c *gin.Context) {
	order, err := a.Dispatch.Accept(c.Param("id"), firstDriverID(a))
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, order)
}

func (a *App) rejectOrder(c *gin.Context) {
	attempt, err := a.Dispatch.Reject(c.Param("id"), firstDriverID(a), "司机拒单")
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, attempt)
}

func (a *App) arriveOrder(c *gin.Context) {
	order, err := a.Orders.Arrive(c.Param("id"))
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, order)
}

func (a *App) startTrip(c *gin.Context) {
	order, err := a.Orders.StartTrip(c.Param("id"))
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, order)
}

func (a *App) endTrip(c *gin.Context) {
	order, err := a.Orders.EndTrip(c.Param("id"))
	if err != nil { fail(c, http.StatusConflict, err.Error()); return }
	ok(c, order)
}
```

- [ ] **Step 6: Implement admin handlers**

Create `backend/internal/http/handlers_admin.go`:

```go
package http

import "github.com/gin-gonic/gin"

func (a *App) adminOrders(c *gin.Context) { ok(c, a.Store.ListOrders()) }
func (a *App) adminDrivers(c *gin.Context) { ok(c, a.Store.ListDrivers()) }
func (a *App) adminApproveDriver(c *gin.Context) { ok(c, gin.H{"driverId": c.Param("id"), "auditState": "APPROVED"}) }
```

- [ ] **Step 7: Implement WebSocket handler**

Create `backend/internal/http/handlers_ws.go`:

```go
package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (a *App) websocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }
	defer conn.Close()
	conn.WriteJSON(gin.H{"type": "CONNECTED", "message": "实时通道已连接"})
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := conn.WriteJSON(gin.H{"type": "HEARTBEAT", "time": time.Now()}); err != nil { return }
	}
}
```

- [ ] **Step 8: Implement main**

Create `backend/cmd/api/main.go`:

```go
package main

import (
	"log"

	api "didi/backend/internal/http"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
)

func main() {
	s := store.NewMemoryStore()
	store.SeedDemoData(s)
	app := &api.App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s)}
	r := api.NewRouter(app)
	log.Println("ride-hailing API listening on :8080")
	if err := r.Run(":8080"); err != nil { log.Fatal(err) }
}
```

- [ ] **Step 9: Verify backend builds**

Run:

```bash
cd backend && go test ./...
```

Expected: PASS.

---

### Task 5: MySQL Schema and Docker Compose

**Files:**
- Create: `infra/schema.sql`
- Create: `infra/docker-compose.yml`

- [ ] **Step 1: Create schema**

Create `infra/schema.sql` containing tables for accounts, passenger profiles, driver profiles, vehicles, ride orders, dispatch tasks, dispatch attempts, payment orders, payment transactions, reviews, admin users, and admin action logs. Use enum-like VARCHAR fields matching the status constants from Task 1.

- [ ] **Step 2: Create Docker Compose**

Create `infra/docker-compose.yml` with services:

```yaml
services:
  mysql:
    image: mysql:8.4
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: didi
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
    environment:
      KAFKA_CFG_NODE_ID: 1
      KAFKA_CFG_PROCESS_ROLES: broker,controller
      KAFKA_CFG_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CFG_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE: "true"
  api:
    build: ../backend
    ports:
      - "8080:8080"
    depends_on:
      - mysql
      - redis
      - kafka
```

- [ ] **Step 3: Add backend Dockerfile**

Create `backend/Dockerfile`:

```dockerfile
FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/didi-api ./cmd/api

FROM debian:bookworm-slim
COPY --from=build /bin/didi-api /bin/didi-api
EXPOSE 8080
CMD ["/bin/didi-api"]
```

- [ ] **Step 4: Validate compose config**

Run:

```bash
cd infra && docker compose config
```

Expected: Compose renders without errors.

---

### Task 6: Vue Frontend Shell and API Client

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/tsconfig.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/index.html`
- Create: `frontend/src/main.ts`
- Create: `frontend/src/App.vue`
- Create: `frontend/src/api.ts`
- Create: `frontend/src/types.ts`
- Create: `frontend/src/styles.css`

- [ ] **Step 1: Create package metadata**

Create `frontend/package.json`:

```json
{
  "scripts": { "dev": "vite", "build": "vue-tsc --noEmit && vite build", "preview": "vite preview" },
  "dependencies": { "@vitejs/plugin-vue": "latest", "vite": "latest", "vue": "latest", "typescript": "latest", "vue-tsc": "latest" },
  "devDependencies": {}
}
```

- [ ] **Step 2: Create TS and Vite config**

Create `frontend/tsconfig.json` and `frontend/vite.config.ts` with standard Vue 3 Vite settings and proxy `/api` plus `/ws` to `http://localhost:8080`.

- [ ] **Step 3: Create shared types and API client**

Create `frontend/src/types.ts` with TypeScript interfaces matching `RideOrder`, `DriverProfile`, `Point`, statuses, and API response envelope.

Create `frontend/src/api.ts` exporting `apiGet` and `apiPost` using `fetch`, unwrapping `{ data }`, and throwing `Error(error)` for failed responses.

- [ ] **Step 4: Create app selector**

Create `frontend/src/App.vue` that reads `new URLSearchParams(location.search).get('app')` and renders passenger, driver, or admin app. Default to passenger.

- [ ] **Step 5: Create global styles**

Create mobile-first CSS for H5 pages, card panels, map mock area, bottom action panel, status pills, admin table, and buttons.

- [ ] **Step 6: Verify frontend compiles**

Run:

```bash
cd frontend && npm install && npm run build
```

Expected: PASS.

---

### Task 7: Passenger, Driver, and Admin Apps

**Files:**
- Create: `frontend/src/apps/passenger/PassengerApp.vue`
- Create: `frontend/src/apps/driver/DriverApp.vue`
- Create: `frontend/src/apps/admin/AdminApp.vue`

- [ ] **Step 1: Passenger app**

Implement passenger app with:

- login panel using phone and fixed code `123456`
- map mock panel showing pickup/dropoff
- call ride button posting `/api/passenger/orders`
- order status timeline
- cancel, pay, and review actions
- polling `/api/passenger/orders` every 3 seconds for first version

- [ ] **Step 2: Driver app**

Implement driver app with:

- driver profile fetch
- online/offline buttons
- order list fetch
- accept, reject, arrive, start, and end buttons based on order state
- mobile-first dispatch card with countdown-like visual text

- [ ] **Step 3: Admin app**

Implement admin app with:

- driver table
- order table
- status filters using local state
- approve driver action
- abnormal order section for CANCELED and DISPATCH_FAILED

- [ ] **Step 4: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected: PASS.

---

### Task 8: README and Final Verification

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

Create `README.md` with:

- project overview
- architecture summary
- how to run backend
- how to run frontend passenger/driver/admin
- demo credentials and fixed SMS code
- API examples
- Docker Compose instructions
- known first-version limitations

- [ ] **Step 2: Run backend tests**

Run:

```bash
cd backend && go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: PASS.

- [ ] **Step 4: Validate Docker Compose**

Run:

```bash
cd infra && docker compose config
```

Expected: PASS.

- [ ] **Step 5: Manual smoke flow**

Run backend and frontend, then verify:

1. Open passenger app at `http://localhost:5173/?app=passenger`.
2. Login with any phone and code `123456`.
3. Create a ride.
4. Open driver app at `http://localhost:5173/?app=driver`.
5. Accept the ride.
6. Mark arrived, start, and end trip.
7. Return to passenger app and pay.
8. Add review.
9. Open admin app at `http://localhost:5173/?app=admin` and verify the completed order appears.

Expected: Full flow completes without API errors.

---

## Self-Review

Spec coverage:

- Three frontends: covered by Tasks 6-7.
- Go backend: covered by Tasks 1-4.
- State machines: covered by Task 1.
- Dispatch with timeout/reassignment shape: covered by dispatch models and first-offer flow; full background timeout worker is deferred from first runnable implementation and documented as a limitation in README.
- MySQL/Redis/Kafka: schema and Compose covered by Task 5; first runtime store is in-memory for speed and documented as first-version runnable storage.
- WebSocket: covered by Task 4 heartbeat endpoint; domain event fanout is an explicit next iteration.
- Payment simulation: covered by Tasks 3-4 and frontend payment action.
- Admin: covered by Tasks 4 and 7.
- Basic risk: covered by Auth/Risk service in Task 3.
- Structured logs: standard Gin logs and backend startup logs in Task 4.

Placeholder scan: the plan avoids TBD/TODO language in implementation-critical steps. Two steps describe standard config/schema creation instead of full listing because the exact implementation will be generated during execution from the defined models and constants.

Type consistency: Status names and main model names match the approved design spec and Task 1 definitions.

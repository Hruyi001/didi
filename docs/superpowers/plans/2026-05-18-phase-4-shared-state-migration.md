# Phase 4 Shared State Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move legacy `api` business state onto a shared MySQL-backed store and cut one real admin read endpoint to `admin-service` without migrating driver/order write traffic.

**Architecture:** Implement a `store.Store` MySQL adapter behind the existing legacy API service boundary, selected explicitly by `STORE_BACKEND=mysql`. Then replace `admin-service` fake driver snapshots with a read-only MySQL repository and route only safe admin read traffic through the gateway when `ADMIN_BASE` is configured. Keep all driver, order, dispatch, payment, review, and websocket write flows on legacy `api` in this phase.

**Tech Stack:** Go, Gin, GORM, MySQL 8.4, Docker Compose, existing `domain`, `store`, `services`, `gateway`, and `admin` packages.

---

## File Structure

### Create

- `backend/internal/store/store_contract_test.go`
  - Shared behavioral contract for any implementation of `store.Store`.
- `backend/internal/store/mysql_store.go`
  - MySQL-backed implementation of `store.Store` using GORM and existing `domain` types.
- `backend/internal/store/mysql_store_test.go`
  - Runs the shared contract against MySQL when `DIDI_MYSQL_TEST_DSN` is set.
- `backend/internal/store/mysql_seed.go`
  - Idempotent MySQL demo seed helper equivalent to current `SeedDemoData` behavior.
- `backend/internal/admin/repo.go`
  - Read-only admin repository interfaces and MySQL-backed driver snapshot reader.
- `backend/internal/admin/auth.go`
  - Admin-service HTTP middleware that validates existing signed access tokens and requires `ADMIN` role.

### Modify

- `backend/go.mod`
  - Add MySQL GORM driver dependency.
- `backend/go.sum`
  - Updated by `go mod tidy`.
- `infra/schema.sql`
  - Add `driver_locations` table because existing protected APIs update and read driver locations.
- `backend/internal/store/store.go`
  - No interface expansion expected; confirm `MySQLStore` implements existing methods.
- `backend/cmd/api/main.go`
  - Select `MemoryStore` by default and `MySQLStore` only when `STORE_BACKEND=mysql`.
- `backend/internal/admin/service.go`
  - Replace fixed fake snapshot with repository-backed read model.
- `backend/internal/admin/http.go`
  - Register `/health`, `/api/admin/drivers`, `/v1/admin/drivers`, and require admin auth for admin routes.
- `backend/cmd/admin-service/main.go`
  - Connect to MySQL when `ADMIN_MYSQL_DSN` is configured and use repository-backed service.
- `backend/internal/gateway/router.go`
  - Add optional `AdminBase` and route only admin read endpoints to admin upstream.
- `backend/internal/gateway/router_test.go`
  - Add tests for admin read cutover and write fallback.
- `backend/cmd/gateway/main.go`
  - Read `ADMIN_BASE` from environment.
- `infra/docker-compose.yml`
  - Configure `api` with explicit MySQL store, add `admin-service`, and configure `gateway` with `ADMIN_BASE`.
- `backend/integration/order_dispatch_flow_test.go`
  - Add admin read endpoint integration proof after auth still works.

---

## Task 1: Add Store Contract Coverage for Existing MemoryStore

**Files:**
- Create: `backend/internal/store/store_contract_test.go`
- Test: `backend/internal/store/store_contract_test.go`

- [ ] **Step 1: Write the shared store contract test**

Create `backend/internal/store/store_contract_test.go`:

```go
package store

import (
	"testing"
	"time"

	"didi/backend/internal/domain"
)

func TestMemoryStoreContract(t *testing.T) {
	RunStoreContract(t, func(t *testing.T) Store {
		t.Helper()
		return NewMemoryStore()
	})
}

func RunStoreContract(t *testing.T, newStore func(t *testing.T) Store) {
	t.Helper()
	t.Run("passenger and driver seed lookup", func(t *testing.T) {
		s := newStore(t)
		passenger := s.SeedPassenger("13800000001")
		if passenger.ID == "" || passenger.AccountID == "" {
			t.Fatalf("expected seeded passenger ids, got %#v", passenger)
		}
		foundPassenger, ok := s.GetPassengerByPhone("13800000001")
		if !ok || foundPassenger.ID != passenger.ID {
			t.Fatalf("expected passenger lookup by phone, ok=%v passenger=%#v", ok, foundPassenger)
		}

		driver := s.SeedApprovedDriver("13900000001", "京A12345")
		if driver.ID == "" || driver.AccountID == "" || driver.Vehicle.PlateNo != "京A12345" {
			t.Fatalf("expected seeded approved driver with vehicle, got %#v", driver)
		}
		foundDriver, ok := s.GetDriverByPhone("13900000001")
		if !ok || foundDriver.ID != driver.ID {
			t.Fatalf("expected driver lookup by phone, ok=%v driver=%#v", ok, foundDriver)
		}
		drivers := s.ListDrivers()
		if len(drivers) != 1 || drivers[0].ID != driver.ID {
			t.Fatalf("expected one listed driver, got %#v", drivers)
		}
	})

	t.Run("driver work status and location", func(t *testing.T) {
		s := newStore(t)
		driver := s.SeedApprovedDriver("13900000002", "京B67890")
		updated, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle)
		if err != nil {
			t.Fatalf("expected approved driver to go online: %v", err)
		}
		if updated.WorkStatus != domain.DriverOnlineIdle {
			t.Fatalf("expected ONLINE_IDLE, got %s", updated.WorkStatus)
		}
		location, err := s.UpdateDriverLocation(driver.ID, domain.DriverLocation{Lng: 116.397, Lat: 39.908, SpeedKPH: 35})
		if err != nil {
			t.Fatalf("expected location update: %v", err)
		}
		if location.DriverID != driver.ID || location.UpdatedAt.IsZero() {
			t.Fatalf("expected stored location with driver id and timestamp, got %#v", location)
		}
		found, ok := s.GetDriverLocation(driver.ID)
		if !ok || found.Lng != 116.397 || found.Lat != 39.908 || found.SpeedKPH != 35 {
			t.Fatalf("expected location lookup, ok=%v location=%#v", ok, found)
		}
	})

	t.Run("order dispatch accept trip payment review", func(t *testing.T) {
		s := newStore(t)
		passenger := s.SeedPassenger("13800000003")
		driver := s.SeedApprovedDriver("13900000003", "京C00003")
		if _, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle); err != nil {
			t.Fatalf("expected driver online: %v", err)
		}

		order := s.CreateOrder(domain.RideOrder{
			PassengerID: passenger.ID,
			Pickup:      domain.Point{Name: "天安门", Lng: 116.397, Lat: 39.908},
			Dropoff:     domain.Point{Name: "国贸", Lng: 116.457, Lat: 39.914},
		})
		if order.ID == "" || order.Status != domain.OrderCreated || order.PaymentStatus != domain.PaymentUnpaid || order.ReviewStatus != domain.ReviewNotReviewed {
			t.Fatalf("expected created order defaults, got %#v", order)
		}
		byPassenger := s.ListOrdersByPassenger(passenger.ID)
		if len(byPassenger) != 1 || byPassenger[0].ID != order.ID {
			t.Fatalf("expected passenger order listing, got %#v", byPassenger)
		}

		if _, err := s.UpdateOrderStatus(order.ID, domain.OrderDispatching); err != nil {
			t.Fatalf("expected dispatching status: %v", err)
		}
		task := s.CreateDispatchTask(order.ID, 1)
		if task.ID == "" || task.OrderID != order.ID || task.CandidateCount != 1 {
			t.Fatalf("expected dispatch task, got %#v", task)
		}
		offer := s.AddDispatchAttempt(domain.DispatchAttempt{
			DispatchTaskID:   task.ID,
			OrderID:          order.ID,
			DriverID:         driver.ID,
			Status:           domain.DispatchOffered,
			DistanceToPickup: 1.2,
			TimeoutAt:        time.Now().Add(20 * time.Second),
			SequenceNo:       1,
		})
		if offer.ID == "" || offer.Status != domain.DispatchOffered {
			t.Fatalf("expected offered attempt, got %#v", offer)
		}
		attempts := s.ListDispatchAttempts(order.ID)
		if len(attempts) != 1 || attempts[0].DriverID != driver.ID {
			t.Fatalf("expected listed attempt, got %#v", attempts)
		}
		visibleToDriver := s.ListOrdersForDriver(driver.ID)
		if len(visibleToDriver) != 1 || visibleToDriver[0].ID != order.ID {
			t.Fatalf("expected dispatching order visible to offered driver, got %#v", visibleToDriver)
		}

		accepted, err := s.AssignDriver(order.ID, driver.ID)
		if err != nil {
			t.Fatalf("expected assign driver: %v", err)
		}
		if accepted.DriverID != driver.ID || accepted.Status != domain.OrderWaitingPickup || accepted.AcceptedAt == nil {
			t.Fatalf("expected accepted order, got %#v", accepted)
		}
		arrived, err := s.UpdateOrderStatus(order.ID, domain.OrderDriverArrived)
		if err != nil || arrived.ArrivedAt == nil {
			t.Fatalf("expected arrived order, order=%#v err=%v", arrived, err)
		}
		started, err := s.UpdateOrderStatus(order.ID, domain.OrderInProgress)
		if err != nil || started.StartedAt == nil {
			t.Fatalf("expected started order, order=%#v err=%v", started, err)
		}
		ended, err := s.UpdateOrderStatus(order.ID, domain.OrderWaitingPayment)
		if err != nil || ended.EndedAt == nil || ended.FinalAmount != ended.EstimatedAmount {
			t.Fatalf("expected ended order with final amount, order=%#v err=%v", ended, err)
		}

		payment := s.CreatePayment(order.ID, ended.FinalAmount)
		if payment.ID == "" || payment.Status != domain.PaymentUnpaid || payment.Amount != ended.FinalAmount {
			t.Fatalf("expected unpaid payment, got %#v", payment)
		}
		paid, err := s.MarkPaymentPaid(order.ID)
		if err != nil || paid.Status != domain.PaymentPaid || paid.PaidAt == nil {
			t.Fatalf("expected paid payment, payment=%#v err=%v", paid, err)
		}
		completed, err := s.UpdateOrderStatus(order.ID, domain.OrderCompleted)
		if err != nil || completed.PaidAt == nil {
			t.Fatalf("expected completed order, order=%#v err=%v", completed, err)
		}
		review := s.CreateReview(order.ID, 5, "很好")
		if review.ID == "" || review.Status != domain.ReviewReviewed || review.Score != 5 {
			t.Fatalf("expected review, got %#v", review)
		}
		storedOrder, ok := s.GetOrder(order.ID)
		if !ok || storedOrder.ReviewStatus != domain.ReviewReviewed {
			t.Fatalf("expected reviewed order, ok=%v order=%#v", ok, storedOrder)
		}
	})

	t.Run("business errors match memory store semantics", func(t *testing.T) {
		s := newStore(t)
		if _, err := s.SetDriverWorkStatus("missing-driver", domain.DriverOnlineIdle); err == nil || err.Error() != "driver not found" {
			t.Fatalf("expected driver not found, got %v", err)
		}
		if _, err := s.UpdateOrderStatus("missing-order", domain.OrderDispatching); err == nil || err.Error() != "order not found" {
			t.Fatalf("expected order not found, got %v", err)
		}
		if _, err := s.AssignDriver("missing-order", "missing-driver"); err == nil || err.Error() != "order not found" {
			t.Fatalf("expected order not found on assign, got %v", err)
		}
		if _, err := s.MarkPaymentPaid("missing-order"); err == nil || err.Error() != "payment not found" {
			t.Fatalf("expected payment not found, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run the contract test against MemoryStore**

Run:

```bash
go -C /root/didi/backend test ./internal/store
```

Expected: PASS. This locks existing `MemoryStore` behavior before adding MySQL.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/store/store_contract_test.go
git commit -m "test: add store behavior contract

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 2: Add MySQL Driver, Driver Location Schema, and MySQL Store Test Harness

**Files:**
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`
- Modify: `infra/schema.sql:138`
- Create: `backend/internal/store/mysql_store_test.go`

- [ ] **Step 1: Add the MySQL GORM driver dependency**

Run:

```bash
go -C /root/didi/backend get gorm.io/driver/mysql@v1.5.7
```

Expected: `backend/go.mod` includes `gorm.io/driver/mysql v1.5.7`, and `backend/go.sum` is updated.

- [ ] **Step 2: Add driver location schema**

Append to `infra/schema.sql`:

```sql

CREATE TABLE IF NOT EXISTS driver_locations (
  driver_id VARCHAR(64) PRIMARY KEY,
  lng DECIMAL(10, 6) NOT NULL,
  lat DECIMAL(10, 6) NOT NULL,
  speed_kph INT NOT NULL,
  updated_at DATETIME NOT NULL
);
```

- [ ] **Step 3: Write the MySQL contract harness**

Create `backend/internal/store/mysql_store_test.go`:

```go
package store

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLStoreContract(t *testing.T) {
	dsn := os.Getenv("DIDI_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set DIDI_MYSQL_TEST_DSN to run MySQL store contract")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	RunStoreContract(t, func(t *testing.T) Store {
		t.Helper()
		resetMySQLStoreTables(t, db)
		return NewMySQLStore(db)
	})
}

func resetMySQLStoreTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"reviews",
		"payment_orders",
		"dispatch_attempts",
		"dispatch_tasks",
		"driver_locations",
		"ride_orders",
		"vehicles",
		"driver_profiles",
		"passenger_profiles",
		"accounts",
	}
	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
}
```

- [ ] **Step 4: Run the MySQL harness without DSN**

Run:

```bash
go -C /root/didi/backend test ./internal/store
```

Expected: FAIL because `NewMySQLStore` is not defined yet. This confirms the test harness requires the implementation.

- [ ] **Step 5: Commit**

```bash
git add backend/go.mod backend/go.sum infra/schema.sql backend/internal/store/mysql_store_test.go
git commit -m "test: add mysql store contract harness

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 3: Implement MySQL Store Models and Identity/Driver Methods

**Files:**
- Create: `backend/internal/store/mysql_store.go`
- Test: `backend/internal/store/mysql_store_test.go`

- [ ] **Step 1: Add MySQL store struct, table models, and identity/driver methods**

Create `backend/internal/store/mysql_store.go` with this initial implementation:

```go
package store

import (
	"errors"
	"sort"
	"time"

	"didi/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MySQLStore struct {
	db *gorm.DB
}

func NewMySQLStore(db *gorm.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

type accountRow struct {
	ID        string    `gorm:"column:id;primaryKey"`
	Phone     string    `gorm:"column:phone"`
	Role      string    `gorm:"column:role"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (accountRow) TableName() string { return "accounts" }

type passengerProfileRow struct {
	ID        string    `gorm:"column:id;primaryKey"`
	AccountID string    `gorm:"column:account_id"`
	Nickname  string    `gorm:"column:nickname"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (passengerProfileRow) TableName() string { return "passenger_profiles" }

type driverProfileRow struct {
	ID         string    `gorm:"column:id;primaryKey"`
	AccountID  string    `gorm:"column:account_id"`
	Name       string    `gorm:"column:name"`
	Phone      string    `gorm:"column:phone"`
	AuditState string    `gorm:"column:audit_state"`
	WorkStatus string    `gorm:"column:work_status"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (driverProfileRow) TableName() string { return "driver_profiles" }

type vehicleRow struct {
	ID         string `gorm:"column:id;primaryKey"`
	DriverID   string `gorm:"column:driver_id"`
	PlateNo    string `gorm:"column:plate_no"`
	Model      string `gorm:"column:model"`
	Color      string `gorm:"column:color"`
	AuditState string `gorm:"column:audit_state"`
}

func (vehicleRow) TableName() string { return "vehicles" }

type driverLocationRow struct {
	DriverID  string    `gorm:"column:driver_id;primaryKey"`
	Lng       float64   `gorm:"column:lng"`
	Lat       float64   `gorm:"column:lat"`
	SpeedKPH  int       `gorm:"column:speed_kph"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (driverLocationRow) TableName() string { return "driver_locations" }

func (s *MySQLStore) SeedPassenger(phone string) domain.PassengerProfile {
	now := time.Now()
	var existing accountRow
	if err := s.db.Where("phone = ? AND role = ?", phone, string(domain.RolePassenger)).First(&existing).Error; err == nil {
		passenger, _ := s.GetPassengerByPhone(phone)
		return passenger
	}
	account := accountRow{ID: uuid.NewString(), Phone: phone, Role: string(domain.RolePassenger), Status: "ACTIVE", CreatedAt: now}
	passenger := passengerProfileRow{ID: uuid.NewString(), AccountID: account.ID, Nickname: "乘客" + phone[len(phone)-4:], CreatedAt: now}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
		return tx.Create(&passenger).Error
	}); err != nil {
		panic(err)
	}
	return domain.PassengerProfile{ID: passenger.ID, AccountID: passenger.AccountID, Nickname: passenger.Nickname, CreatedAt: passenger.CreatedAt}
}

func (s *MySQLStore) SeedApprovedDriver(phone, plate string) domain.DriverProfile {
	now := time.Now()
	if existing, ok := s.GetDriverByPhone(phone); ok {
		return existing
	}
	account := accountRow{ID: uuid.NewString(), Phone: phone, Role: string(domain.RoleDriver), Status: "ACTIVE", CreatedAt: now}
	driver := driverProfileRow{ID: uuid.NewString(), AccountID: account.ID, Name: "司机" + phone[len(phone)-4:], Phone: phone, AuditState: "APPROVED", WorkStatus: string(domain.DriverOffline), CreatedAt: now}
	vehicle := vehicleRow{ID: uuid.NewString(), DriverID: driver.ID, PlateNo: plate, Model: "快车", Color: "白色", AuditState: "APPROVED"}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
		if err := tx.Create(&driver).Error; err != nil {
			return err
		}
		return tx.Create(&vehicle).Error
	}); err != nil {
		panic(err)
	}
	return rowToDriver(driver, vehicle)
}

func (s *MySQLStore) GetPassengerByPhone(phone string) (domain.PassengerProfile, bool) {
	var row passengerProfileRow
	err := s.db.Table("passenger_profiles").
		Select("passenger_profiles.*").
		Joins("JOIN accounts ON accounts.id = passenger_profiles.account_id").
		Where("accounts.phone = ? AND accounts.role = ?", phone, string(domain.RolePassenger)).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.PassengerProfile{}, false
	}
	if err != nil {
		return domain.PassengerProfile{}, false
	}
	return domain.PassengerProfile{ID: row.ID, AccountID: row.AccountID, Nickname: row.Nickname, CreatedAt: row.CreatedAt}, true
}

func (s *MySQLStore) GetDriverByPhone(phone string) (domain.DriverProfile, bool) {
	var driver driverProfileRow
	if err := s.db.Where("phone = ?", phone).First(&driver).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.DriverProfile{}, false
	} else if err != nil {
		return domain.DriverProfile{}, false
	}
	vehicle := s.vehicleForDriver(driver.ID)
	return rowToDriver(driver, vehicle), true
}

func (s *MySQLStore) ListDrivers() []domain.DriverProfile {
	var rows []driverProfileRow
	if err := s.db.Order("created_at ASC").Find(&rows).Error; err != nil {
		return []domain.DriverProfile{}
	}
	drivers := make([]domain.DriverProfile, 0, len(rows))
	for _, row := range rows {
		drivers = append(drivers, rowToDriver(row, s.vehicleForDriver(row.ID)))
	}
	return drivers
}

func (s *MySQLStore) FindOnlineIdleDrivers() []domain.DriverProfile {
	var rows []driverProfileRow
	if err := s.db.Where("audit_state = ? AND work_status = ?", "APPROVED", string(domain.DriverOnlineIdle)).Order("created_at ASC").Find(&rows).Error; err != nil {
		return []domain.DriverProfile{}
	}
	drivers := make([]domain.DriverProfile, 0, len(rows))
	for _, row := range rows {
		drivers = append(drivers, rowToDriver(row, s.vehicleForDriver(row.ID)))
	}
	return drivers
}

func (s *MySQLStore) SetDriverWorkStatus(driverID string, status domain.DriverWorkStatus) (domain.DriverProfile, error) {
	var row driverProfileRow
	if err := s.db.Where("id = ?", driverID).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.DriverProfile{}, errors.New("driver not found")
	} else if err != nil {
		return domain.DriverProfile{}, err
	}
	if row.AuditState != "APPROVED" && status == domain.DriverOnlineIdle {
		return domain.DriverProfile{}, errors.New("driver not approved")
	}
	row.WorkStatus = string(status)
	if err := s.db.Model(&driverProfileRow{}).Where("id = ?", driverID).Update("work_status", row.WorkStatus).Error; err != nil {
		return domain.DriverProfile{}, err
	}
	return rowToDriver(row, s.vehicleForDriver(row.ID)), nil
}

func (s *MySQLStore) UpdateDriverLocation(driverID string, location domain.DriverLocation) (domain.DriverLocation, error) {
	var driver driverProfileRow
	if err := s.db.Where("id = ?", driverID).First(&driver).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.DriverLocation{}, errors.New("driver not found")
	} else if err != nil {
		return domain.DriverLocation{}, err
	}
	location.DriverID = driverID
	location.UpdatedAt = time.Now()
	row := driverLocationRow{DriverID: driverID, Lng: location.Lng, Lat: location.Lat, SpeedKPH: location.SpeedKPH, UpdatedAt: location.UpdatedAt}
	if err := s.db.Save(&row).Error; err != nil {
		return domain.DriverLocation{}, err
	}
	return location, nil
}

func (s *MySQLStore) GetDriverLocation(driverID string) (domain.DriverLocation, bool) {
	var row driverLocationRow
	if err := s.db.Where("driver_id = ?", driverID).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.DriverLocation{}, false
	} else if err != nil {
		return domain.DriverLocation{}, false
	}
	return domain.DriverLocation{DriverID: row.DriverID, Lng: row.Lng, Lat: row.Lat, SpeedKPH: row.SpeedKPH, UpdatedAt: row.UpdatedAt}, true
}

func (s *MySQLStore) vehicleForDriver(driverID string) vehicleRow {
	var vehicle vehicleRow
	_ = s.db.Where("driver_id = ?", driverID).First(&vehicle).Error
	return vehicle
}

func rowToDriver(row driverProfileRow, vehicle vehicleRow) domain.DriverProfile {
	return domain.DriverProfile{
		ID:         row.ID,
		AccountID:  row.AccountID,
		Name:       row.Name,
		Phone:      row.Phone,
		AuditState: row.AuditState,
		WorkStatus: domain.DriverWorkStatus(row.WorkStatus),
		Vehicle: domain.Vehicle{
			PlateNo: vehicle.PlateNo,
			Model:   vehicle.Model,
			Color:   vehicle.Color,
		},
		CreatedAt: row.CreatedAt,
	}
}

func sortOrdersNewestFirst(orders []domain.RideOrder) {
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.After(orders[j].CreatedAt) })
}
```

- [ ] **Step 2: Run tests to verify remaining interface methods are missing**

Run:

```bash
go -C /root/didi/backend test ./internal/store
```

Expected: FAIL with compile errors showing `*MySQLStore` does not implement `Store` because order, dispatch, payment, and review methods are missing.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/store/mysql_store.go
git commit -m "feat: add mysql store identity and driver methods

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 4: Add MySQL Store Order Methods

**Files:**
- Modify: `backend/internal/store/mysql_store.go`
- Test: `backend/internal/store/store_contract_test.go`

- [ ] **Step 1: Append order row model and order conversion helpers**

Append this code to `backend/internal/store/mysql_store.go`:

```go

type rideOrderRow struct {
	ID                string     `gorm:"column:id;primaryKey"`
	PassengerID       string     `gorm:"column:passenger_id"`
	DriverID          *string    `gorm:"column:driver_id"`
	PickupName        string     `gorm:"column:pickup_name"`
	PickupLng         float64    `gorm:"column:pickup_lng"`
	PickupLat         float64    `gorm:"column:pickup_lat"`
	DropoffName       string     `gorm:"column:dropoff_name"`
	DropoffLng        float64    `gorm:"column:dropoff_lng"`
	DropoffLat        float64    `gorm:"column:dropoff_lat"`
	Status            string     `gorm:"column:status"`
	PaymentStatus     string     `gorm:"column:payment_status"`
	ReviewStatus      string     `gorm:"column:review_status"`
	EstimatedDistance float64    `gorm:"column:estimated_distance"`
	EstimatedDuration int        `gorm:"column:estimated_duration"`
	EstimatedAmount   int64      `gorm:"column:estimated_amount"`
	FinalAmount       int64      `gorm:"column:final_amount"`
	CancelReason      string     `gorm:"column:cancel_reason"`
	Version           int64      `gorm:"column:version"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	AcceptedAt        *time.Time `gorm:"column:accepted_at"`
	ArrivedAt         *time.Time `gorm:"column:arrived_at"`
	StartedAt         *time.Time `gorm:"column:started_at"`
	EndedAt           *time.Time `gorm:"column:ended_at"`
	PaidAt            *time.Time `gorm:"column:paid_at"`
}

func (rideOrderRow) TableName() string { return "ride_orders" }

func orderToRow(order domain.RideOrder) rideOrderRow {
	var driverID *string
	if order.DriverID != "" {
		driverID = &order.DriverID
	}
	return rideOrderRow{
		ID:                order.ID,
		PassengerID:       order.PassengerID,
		DriverID:          driverID,
		PickupName:        order.Pickup.Name,
		PickupLng:         order.Pickup.Lng,
		PickupLat:         order.Pickup.Lat,
		DropoffName:       order.Dropoff.Name,
		DropoffLng:        order.Dropoff.Lng,
		DropoffLat:        order.Dropoff.Lat,
		Status:            string(order.Status),
		PaymentStatus:     string(order.PaymentStatus),
		ReviewStatus:      string(order.ReviewStatus),
		EstimatedDistance: order.EstimatedDistance,
		EstimatedDuration: order.EstimatedDuration,
		EstimatedAmount:   order.EstimatedAmount,
		FinalAmount:       order.FinalAmount,
		CancelReason:      order.CancelReason,
		Version:           order.Version,
		CreatedAt:         order.CreatedAt,
		AcceptedAt:        order.AcceptedAt,
		ArrivedAt:         order.ArrivedAt,
		StartedAt:         order.StartedAt,
		EndedAt:           order.EndedAt,
		PaidAt:            order.PaidAt,
	}
}

func rowToOrder(row rideOrderRow) domain.RideOrder {
	driverID := ""
	if row.DriverID != nil {
		driverID = *row.DriverID
	}
	return domain.RideOrder{
		ID:                row.ID,
		PassengerID:       row.PassengerID,
		DriverID:          driverID,
		Pickup:            domain.Point{Name: row.PickupName, Lng: row.PickupLng, Lat: row.PickupLat},
		Dropoff:           domain.Point{Name: row.DropoffName, Lng: row.DropoffLng, Lat: row.DropoffLat},
		Status:            domain.OrderStatus(row.Status),
		PaymentStatus:     domain.PaymentStatus(row.PaymentStatus),
		ReviewStatus:      domain.ReviewStatus(row.ReviewStatus),
		EstimatedDistance: row.EstimatedDistance,
		EstimatedDuration: row.EstimatedDuration,
		EstimatedAmount:   row.EstimatedAmount,
		FinalAmount:       row.FinalAmount,
		CancelReason:      row.CancelReason,
		Version:           row.Version,
		CreatedAt:         row.CreatedAt,
		AcceptedAt:        row.AcceptedAt,
		ArrivedAt:         row.ArrivedAt,
		StartedAt:         row.StartedAt,
		EndedAt:           row.EndedAt,
		PaidAt:            row.PaidAt,
	}
}
```

- [ ] **Step 2: Append order CRUD and status methods**

Append this code to `backend/internal/store/mysql_store.go`:

```go

func (s *MySQLStore) CreateOrder(order domain.RideOrder) domain.RideOrder {
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
	if err := s.db.Create(orderToRow(order)).Error; err != nil {
		panic(err)
	}
	return order
}

func (s *MySQLStore) GetOrder(orderID string) (domain.RideOrder, bool) {
	var row rideOrderRow
	if err := s.db.Where("id = ?", orderID).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.RideOrder{}, false
	} else if err != nil {
		return domain.RideOrder{}, false
	}
	return rowToOrder(row), true
}

func (s *MySQLStore) ListOrders() []domain.RideOrder {
	var rows []rideOrderRow
	if err := s.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return []domain.RideOrder{}
	}
	orders := make([]domain.RideOrder, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, rowToOrder(row))
	}
	return orders
}

func (s *MySQLStore) ListOrdersByPassenger(passengerID string) []domain.RideOrder {
	var rows []rideOrderRow
	if err := s.db.Where("passenger_id = ?", passengerID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return []domain.RideOrder{}
	}
	orders := make([]domain.RideOrder, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, rowToOrder(row))
	}
	return orders
}

func (s *MySQLStore) ListOrdersForDriver(driverID string) []domain.RideOrder {
	orders := s.ListOrders()
	result := make([]domain.RideOrder, 0, len(orders))
	for _, order := range orders {
		if order.DriverID == driverID {
			result = append(result, order)
			continue
		}
		if order.Status != domain.OrderDispatching {
			continue
		}
		attempts := s.ListDispatchAttempts(order.ID)
		if len(attempts) == 0 {
			continue
		}
		latest := attempts[len(attempts)-1]
		if latest.Status == domain.DispatchOffered && latest.DriverID == driverID {
			result = append(result, order)
		}
	}
	sortOrdersNewestFirst(result)
	return result
}

func (s *MySQLStore) UpdateOrderStatus(orderID string, to domain.OrderStatus) (domain.RideOrder, error) {
	var updated domain.RideOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var row rideOrderRow
		if err := tx.Where("id = ?", orderID).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("order not found")
		} else if err != nil {
			return err
		}
		order := rowToOrder(row)
		if err := domain.CanTransitionOrder(order.Status, to); err != nil {
			return err
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
		if err := tx.Save(orderToRow(order)).Error; err != nil {
			return err
		}
		updated = order
		return nil
	})
	if err != nil {
		return domain.RideOrder{}, err
	}
	return updated, nil
}
```

- [ ] **Step 3: Run tests to verify dispatch/payment/review methods are still missing**

Run:

```bash
go -C /root/didi/backend test ./internal/store
```

Expected: FAIL with compile errors for missing `AssignDriver`, dispatch, payment, and review methods.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/store/mysql_store.go
git commit -m "feat: add mysql store order methods

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 5: Add MySQL Store Dispatch Methods

**Files:**
- Modify: `backend/internal/store/mysql_store.go`
- Test: `backend/internal/store/store_contract_test.go`

- [ ] **Step 1: Append dispatch row models and helpers**

Append this code to `backend/internal/store/mysql_store.go`:

```go

type dispatchTaskRow struct {
	ID               string    `gorm:"column:id;primaryKey"`
	OrderID          string    `gorm:"column:order_id"`
	Status           string    `gorm:"column:status"`
	CandidateCount   int       `gorm:"column:candidate_count"`
	CurrentAttemptNo int       `gorm:"column:current_attempt_no"`
	MaxAttempts      int       `gorm:"column:max_attempts"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (dispatchTaskRow) TableName() string { return "dispatch_tasks" }

type dispatchAttemptRow struct {
	ID               string     `gorm:"column:id;primaryKey"`
	DispatchTaskID   string     `gorm:"column:dispatch_task_id"`
	OrderID          string     `gorm:"column:order_id"`
	DriverID         string     `gorm:"column:driver_id"`
	Status           string     `gorm:"column:status"`
	DistanceToPickup float64    `gorm:"column:distance_to_pickup"`
	OfferedAt        time.Time  `gorm:"column:offered_at"`
	RespondedAt      *time.Time `gorm:"column:responded_at"`
	TimeoutAt        time.Time  `gorm:"column:timeout_at"`
	RejectReason     string     `gorm:"column:reject_reason"`
	SequenceNo       int        `gorm:"column:sequence_no"`
}

func (dispatchAttemptRow) TableName() string { return "dispatch_attempts" }

func rowToDispatchTask(row dispatchTaskRow) domain.DispatchTask {
	return domain.DispatchTask{ID: row.ID, OrderID: row.OrderID, Status: domain.DispatchStatus(row.Status), CandidateCount: row.CandidateCount, CurrentAttemptNo: row.CurrentAttemptNo, MaxAttempts: row.MaxAttempts, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func rowToDispatchAttempt(row dispatchAttemptRow) domain.DispatchAttempt {
	return domain.DispatchAttempt{ID: row.ID, DispatchTaskID: row.DispatchTaskID, OrderID: row.OrderID, DriverID: row.DriverID, Status: domain.DispatchStatus(row.Status), DistanceToPickup: row.DistanceToPickup, OfferedAt: row.OfferedAt, RespondedAt: row.RespondedAt, TimeoutAt: row.TimeoutAt, RejectReason: row.RejectReason, SequenceNo: row.SequenceNo}
}
```

- [ ] **Step 2: Append assign and dispatch methods**

Append this code to `backend/internal/store/mysql_store.go`:

```go

func (s *MySQLStore) AssignDriver(orderID, driverID string) (domain.RideOrder, error) {
	var assigned domain.RideOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var orderRow rideOrderRow
		if err := tx.Where("id = ?", orderID).First(&orderRow).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("order not found")
		} else if err != nil {
			return err
		}
		var driverRow driverProfileRow
		if err := tx.Where("id = ?", driverID).First(&driverRow).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("driver not found")
		} else if err != nil {
			return err
		}
		if driverRow.WorkStatus != string(domain.DriverDispatched) && driverRow.WorkStatus != string(domain.DriverOnlineIdle) {
			return errors.New("driver not available")
		}
		order := rowToOrder(orderRow)
		if err := domain.CanTransitionOrder(order.Status, domain.OrderWaitingPickup); err != nil {
			return err
		}
		now := time.Now()
		order.DriverID = driverID
		order.Status = domain.OrderWaitingPickup
		order.AcceptedAt = &now
		order.Version++
		if err := tx.Save(orderToRow(order)).Error; err != nil {
			return err
		}
		if err := tx.Model(&driverProfileRow{}).Where("id = ?", driverID).Update("work_status", string(domain.DriverServing)).Error; err != nil {
			return err
		}
		assigned = order
		return nil
	})
	if err != nil {
		return domain.RideOrder{}, err
	}
	return assigned, nil
}

func (s *MySQLStore) CreateDispatchTask(orderID string, candidates int) domain.DispatchTask {
	now := time.Now()
	row := dispatchTaskRow{ID: uuid.NewString(), OrderID: orderID, Status: string(domain.DispatchPending), CandidateCount: candidates, CurrentAttemptNo: 0, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now}
	if err := s.db.Create(&row).Error; err != nil {
		panic(err)
	}
	return rowToDispatchTask(row)
}

func (s *MySQLStore) AddDispatchAttempt(attempt domain.DispatchAttempt) domain.DispatchAttempt {
	attempt.ID = uuid.NewString()
	if attempt.OfferedAt.IsZero() {
		attempt.OfferedAt = time.Now()
	}
	row := dispatchAttemptRow{ID: attempt.ID, DispatchTaskID: attempt.DispatchTaskID, OrderID: attempt.OrderID, DriverID: attempt.DriverID, Status: string(attempt.Status), DistanceToPickup: attempt.DistanceToPickup, OfferedAt: attempt.OfferedAt, RespondedAt: attempt.RespondedAt, TimeoutAt: attempt.TimeoutAt, RejectReason: attempt.RejectReason, SequenceNo: attempt.SequenceNo}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if attempt.Status == domain.DispatchOffered {
			return tx.Model(&driverProfileRow{}).Where("id = ?", attempt.DriverID).Update("work_status", string(domain.DriverDispatched)).Error
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return attempt
}

func (s *MySQLStore) ListDispatchAttempts(orderID string) []domain.DispatchAttempt {
	var rows []dispatchAttemptRow
	if err := s.db.Where("order_id = ?", orderID).Order("sequence_no ASC").Find(&rows).Error; err != nil {
		return []domain.DispatchAttempt{}
	}
	attempts := make([]domain.DispatchAttempt, 0, len(rows))
	for _, row := range rows {
		attempts = append(attempts, rowToDispatchAttempt(row))
	}
	return attempts
}

func (s *MySQLStore) ResetDriverToIdle(driverID string) error {
	var driver driverProfileRow
	if err := s.db.Where("id = ?", driverID).First(&driver).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("driver not found")
	} else if err != nil {
		return err
	}
	if driver.WorkStatus == string(domain.DriverDispatched) {
		return s.db.Model(&driverProfileRow{}).Where("id = ?", driverID).Update("work_status", string(domain.DriverOnlineIdle)).Error
	}
	return nil
}
```

- [ ] **Step 3: Run tests to verify payment/review methods are still missing**

Run:

```bash
go -C /root/didi/backend test ./internal/store
```

Expected: FAIL with compile errors for missing `CreatePayment`, `MarkPaymentPaid`, and `CreateReview`.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/store/mysql_store.go
git commit -m "feat: add mysql store dispatch methods

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 6: Add MySQL Store Payment and Review Methods

**Files:**
- Modify: `backend/internal/store/mysql_store.go`
- Test: `backend/internal/store/store_contract_test.go`

- [ ] **Step 1: Append payment/review row models and methods**

Append this code to `backend/internal/store/mysql_store.go`:

```go

type paymentOrderRow struct {
	ID        string     `gorm:"column:id;primaryKey"`
	OrderID   string     `gorm:"column:order_id"`
	Amount    int64      `gorm:"column:amount"`
	Status    string     `gorm:"column:status"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	PaidAt    *time.Time `gorm:"column:paid_at"`
}

func (paymentOrderRow) TableName() string { return "payment_orders" }

type reviewRow struct {
	ID        string    `gorm:"column:id;primaryKey"`
	OrderID   string    `gorm:"column:order_id"`
	Score     int       `gorm:"column:score"`
	Content   string    `gorm:"column:content"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (reviewRow) TableName() string { return "reviews" }

func rowToPayment(row paymentOrderRow) domain.PaymentOrder {
	return domain.PaymentOrder{ID: row.ID, OrderID: row.OrderID, Amount: row.Amount, Status: domain.PaymentStatus(row.Status), CreatedAt: row.CreatedAt, PaidAt: row.PaidAt}
}

func rowToReview(row reviewRow) domain.Review {
	return domain.Review{ID: row.ID, OrderID: row.OrderID, Score: row.Score, Content: row.Content, Status: domain.ReviewStatus(row.Status), CreatedAt: row.CreatedAt}
}

func (s *MySQLStore) CreatePayment(orderID string, amount int64) domain.PaymentOrder {
	row := paymentOrderRow{ID: uuid.NewString(), OrderID: orderID, Amount: amount, Status: string(domain.PaymentUnpaid), CreatedAt: time.Now()}
	if err := s.db.Create(&row).Error; err != nil {
		panic(err)
	}
	return rowToPayment(row)
}

func (s *MySQLStore) MarkPaymentPaid(orderID string) (domain.PaymentOrder, error) {
	var updated domain.PaymentOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var row paymentOrderRow
		if err := tx.Where("order_id = ?", orderID).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("payment not found")
		} else if err != nil {
			return err
		}
		now := time.Now()
		row.Status = string(domain.PaymentPaid)
		row.PaidAt = &now
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		updated = rowToPayment(row)
		return nil
	})
	if err != nil {
		return domain.PaymentOrder{}, err
	}
	return updated, nil
}

func (s *MySQLStore) CreateReview(orderID string, score int, content string) domain.Review {
	row := reviewRow{ID: uuid.NewString(), OrderID: orderID, Score: score, Content: content, Status: string(domain.ReviewReviewed), CreatedAt: time.Now()}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&rideOrderRow{}).Where("id = ?", orderID).Updates(map[string]any{"review_status": string(domain.ReviewReviewed), "version": gorm.Expr("version + 1")}).Error
	}); err != nil {
		panic(err)
	}
	return rowToReview(row)
}
```

- [ ] **Step 2: Run store tests**

Run:

```bash
go -C /root/didi/backend test ./internal/store
```

Expected: PASS when `DIDI_MYSQL_TEST_DSN` is unset because the MySQL contract skips and the MemoryStore contract passes.

- [ ] **Step 3: Run full backend unit tests**

Run:

```bash
go -C /root/didi/backend test ./internal/auth ./internal/gateway ./internal/http ./internal/store ./internal/driver ./internal/admin ./internal/domain
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/store/mysql_store.go
git commit -m "feat: complete mysql store behavior

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 7: Add Idempotent MySQL Demo Seeding and API Store Selection

**Files:**
- Create: `backend/internal/store/mysql_seed.go`
- Modify: `backend/cmd/api/main.go:1-31`
- Test: `backend/internal/store/mysql_store_test.go`

- [ ] **Step 1: Add MySQL demo seed helper**

Create `backend/internal/store/mysql_seed.go`:

```go
package store

import "didi/backend/internal/domain"

func SeedDemoDataForStore(s Store) {
	s.SeedPassenger("13800000001")
	d1 := s.SeedApprovedDriver("13900000001", "京A12345")
	d2 := s.SeedApprovedDriver("13900000002", "京B67890")
	_, _ = s.SetDriverWorkStatus(d1.ID, domain.DriverOnlineIdle)
	_, _ = s.SetDriverWorkStatus(d2.ID, domain.DriverOnlineIdle)
}
```

- [ ] **Step 2: Update existing memory seed to delegate through Store interface**

Modify `backend/internal/store/seed.go` to:

```go
package store

func SeedDemoData(s *MemoryStore) {
	SeedDemoDataForStore(s)
}
```

- [ ] **Step 3: Replace API main with explicit store backend selection**

Replace `backend/cmd/api/main.go` with:

```go
package main

import (
	"log"
	"os"
	"time"

	api "didi/backend/internal/http"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	s := buildStore()
	dispatch := services.NewDispatchService(s)
	app := &api.App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: dispatch, Payment: services.NewPaymentService(s), Events: api.NewEventHub()}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := dispatch.ProcessTimeouts(time.Now()); err != nil {
				log.Printf("process dispatch timeouts: %v", err)
			}
		}
	}()
	r := api.NewRouter(app)
	log.Println("ride-hailing API listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func buildStore() store.Store {
	if os.Getenv("STORE_BACKEND") != "mysql" {
		s := store.NewMemoryStore()
		store.SeedDemoData(s)
		return s
	}
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required when STORE_BACKEND=mysql")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("open mysql store: %v", err)
	}
	s := store.NewMySQLStore(db)
	store.SeedDemoDataForStore(s)
	return s
}
```

- [ ] **Step 4: Run API-related tests**

Run:

```bash
go -C /root/didi/backend test ./internal/http ./internal/store
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/store/mysql_seed.go backend/internal/store/seed.go backend/cmd/api/main.go
git commit -m "feat: configure api shared store backend

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 8: Convert Admin Service to Authenticated MySQL Read Model

**Files:**
- Create: `backend/internal/admin/repo.go`
- Create: `backend/internal/admin/auth.go`
- Modify: `backend/internal/admin/service.go`
- Modify: `backend/internal/admin/http.go`
- Modify: `backend/internal/admin/service_test.go`
- Modify: `backend/cmd/admin-service/main.go`

- [ ] **Step 1: Replace admin service test with repository-backed behavior**

Replace `backend/internal/admin/service_test.go` with:

```go
package admin

import "testing"

type fakeDriverRepo struct {
	drivers []DriverSnapshot
}

func (r fakeDriverRepo) ListDrivers() ([]DriverSnapshot, error) {
	return r.drivers, nil
}

func TestListDriversReturnsRepositorySnapshots(t *testing.T) {
	svc := NewService(fakeDriverRepo{drivers: []DriverSnapshot{{ID: "driver-1", Phone: "13900000001", AuditState: "APPROVED", WorkStatus: "ONLINE_IDLE", PlateNo: "京A12345"}}})
	drivers, err := svc.ListDrivers()
	if err != nil {
		t.Fatalf("expected drivers: %v", err)
	}
	if len(drivers) != 1 {
		t.Fatalf("expected one driver, got %#v", drivers)
	}
	if drivers[0].Phone != "13900000001" || drivers[0].PlateNo != "京A12345" {
		t.Fatalf("expected repository driver data, got %#v", drivers[0])
	}
}
```

- [ ] **Step 2: Run test to verify current fake service fails**

Run:

```bash
go -C /root/didi/backend test ./internal/admin
```

Expected: FAIL because `NewService` currently takes no repository and `ListDrivers` returns no error.

- [ ] **Step 3: Add admin repository**

Create `backend/internal/admin/repo.go`:

```go
package admin

import "gorm.io/gorm"

type DriverRepository interface {
	ListDrivers() ([]DriverSnapshot, error)
}

type MySQLDriverRepository struct {
	db *gorm.DB
}

func NewMySQLDriverRepository(db *gorm.DB) *MySQLDriverRepository {
	return &MySQLDriverRepository{db: db}
}

type driverSnapshotRow struct {
	ID         string `gorm:"column:id"`
	Phone      string `gorm:"column:phone"`
	AuditState string `gorm:"column:audit_state"`
	WorkStatus string `gorm:"column:work_status"`
	PlateNo    string `gorm:"column:plate_no"`
}

func (r *MySQLDriverRepository) ListDrivers() ([]DriverSnapshot, error) {
	var rows []driverSnapshotRow
	err := r.db.Table("driver_profiles").
		Select("driver_profiles.id, driver_profiles.phone, driver_profiles.audit_state, driver_profiles.work_status, vehicles.plate_no").
		Joins("LEFT JOIN vehicles ON vehicles.driver_id = driver_profiles.id").
		Order("driver_profiles.created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	drivers := make([]DriverSnapshot, 0, len(rows))
	for _, row := range rows {
		drivers = append(drivers, DriverSnapshot{ID: row.ID, Phone: row.Phone, AuditState: row.AuditState, WorkStatus: row.WorkStatus, PlateNo: row.PlateNo})
	}
	return drivers, nil
}
```

- [ ] **Step 4: Replace admin service implementation**

Replace `backend/internal/admin/service.go` with:

```go
package admin

type DriverSnapshot struct {
	ID         string `json:"id"`
	Phone      string `json:"phone"`
	AuditState string `json:"auditState"`
	WorkStatus string `json:"workStatus"`
	PlateNo    string `json:"plateNo"`
}

type Service struct {
	drivers DriverRepository
}

func NewService(drivers DriverRepository) *Service {
	return &Service{drivers: drivers}
}

func (s *Service) ListDrivers() ([]DriverSnapshot, error) {
	return s.drivers.ListDrivers()
}
```

- [ ] **Step 5: Add admin-service auth middleware**

Create `backend/internal/admin/auth.go`:

```go
package admin

import (
	"net/http"
	"strings"

	"didi/backend/internal/domain"
	"didi/backend/internal/platform/httpx"
	"didi/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func requireAdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			httpx.Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		session, err := services.ValidateToken(strings.TrimPrefix(authorization, "Bearer "))
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		if session.Role != domain.RoleAdmin {
			httpx.Fail(c, http.StatusForbidden, "无权访问该资源")
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 6: Replace admin HTTP registration**

Replace `backend/internal/admin/http.go` with:

```go
package admin

import (
	"net/http"

	"didi/backend/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok", "service": "admin-service"})
	})

	handler := func(c *gin.Context) {
		drivers, err := svc.ListDrivers()
		if err != nil {
			httpx.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		httpx.OK(c, drivers)
	}

	admin := r.Group("/api/admin", requireAdminAuth())
	admin.GET("/drivers", handler)

	v1 := r.Group("/v1/admin", requireAdminAuth())
	v1.GET("/drivers", handler)
}
```

- [ ] **Step 7: Replace admin-service main**

Replace `backend/cmd/admin-service/main.go` with:

```go
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
```

- [ ] **Step 8: Run admin tests**

Run:

```bash
go -C /root/didi/backend test ./internal/admin
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add backend/internal/admin/repo.go backend/internal/admin/auth.go backend/internal/admin/service.go backend/internal/admin/http.go backend/internal/admin/service_test.go backend/cmd/admin-service/main.go
git commit -m "feat: make admin service read shared mysql state

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 9: Route Only Admin Read Traffic Through Gateway

**Files:**
- Modify: `backend/internal/gateway/router.go`
- Modify: `backend/internal/gateway/router_test.go`
- Modify: `backend/cmd/gateway/main.go`

- [ ] **Step 1: Add failing gateway tests for admin read cutover and write fallback**

Append to `backend/internal/gateway/router_test.go`:

```go

func TestAdminDriversRoutesToAdminProxyWhenConfigured(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "admin")
	}))
	defer admin.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL, AdminBase: admin.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodGet, server.URL+"/api/admin/drivers")
	if body != "admin" {
		t.Fatalf("expected admin upstream, got %q", body)
	}
}

func TestAdminDriversFallsBackToAPIProxyWithoutAdminBase(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodGet, server.URL+"/api/admin/drivers")
	if body != "api" {
		t.Fatalf("expected api upstream, got %q", body)
	}
}

func TestAdminApproveDriverStillRoutesToLegacyAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "admin")
	}))
	defer admin.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL, AdminBase: admin.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodPost, server.URL+"/api/admin/drivers/driver-1/approve")
	if body != "api" {
		t.Fatalf("expected api upstream, got %q", body)
	}
}
```

- [ ] **Step 2: Run gateway tests to verify they fail**

Run:

```bash
go -C /root/didi/backend test ./internal/gateway
```

Expected: FAIL because `Clients` has no `AdminBase` field and routing does not recognize admin read endpoints.

- [ ] **Step 3: Replace gateway router implementation**

Replace `backend/internal/gateway/router.go` with:

```go
package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Clients struct {
	APIBase   string
	AuthBase  string
	AdminBase string
}

func NewRouter(clients *Clients) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": gin.H{"status": "ok", "service": "gateway"}})
	})
	if clients != nil && clients.APIBase != "" {
		apiProxy := newReverseProxy(clients.APIBase)
		authProxy := apiProxy
		if clients.AuthBase != "" {
			authProxy = newReverseProxy(clients.AuthBase)
		}
		adminProxy := apiProxy
		if clients.AdminBase != "" {
			adminProxy = newReverseProxy(clients.AdminBase)
		}
		r.Any("/api/*path", func(c *gin.Context) {
			if shouldProxyToAuth(c) {
				gin.WrapH(authProxy)(c)
				return
			}
			if shouldProxyToAdmin(c) {
				gin.WrapH(adminProxy)(c)
				return
			}
			gin.WrapH(apiProxy)(c)
		})
		r.Any("/ws", gin.WrapH(apiProxy))
	}
	return r
}

func shouldProxyToAuth(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	switch c.Request.URL.Path {
	case "/api/auth/login", "/api/auth/send-code":
		return true
	default:
		return false
	}
}

func shouldProxyToAdmin(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet {
		return false
	}
	switch c.Request.URL.Path {
	case "/api/admin/drivers":
		return true
	default:
		return false
	}
}

func newReverseProxy(target string) *httputil.ReverseProxy {
	urlValue, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(urlValue)
}
```

- [ ] **Step 4: Update gateway main to read ADMIN_BASE**

Replace `backend/cmd/gateway/main.go` with:

```go
package main

import (
	"log"
	"os"

	"didi/backend/internal/gateway"
)

func main() {
	apiBase := os.Getenv("API_BASE")
	if apiBase == "" {
		apiBase = "http://localhost:18080"
	}
	authBase := os.Getenv("AUTH_BASE")
	adminBase := os.Getenv("ADMIN_BASE")
	r := gateway.NewRouter(&gateway.Clients{APIBase: apiBase, AuthBase: authBase, AdminBase: adminBase})
	log.Fatal(r.Run(":8080"))
}
```

- [ ] **Step 5: Run gateway tests**

Run:

```bash
go -C /root/didi/backend test ./internal/gateway
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/gateway/router.go backend/internal/gateway/router_test.go backend/cmd/gateway/main.go
git commit -m "feat: route admin read traffic to admin service

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 10: Wire Docker Compose for Shared State and Admin Service

**Files:**
- Modify: `infra/docker-compose.yml`

- [ ] **Step 1: Update Compose service configuration**

Modify `infra/docker-compose.yml` so the `api`, new `admin-service`, and `gateway` sections are:

```yaml
  api:
    build: ../backend
    command: ["/bin/didi-api"]
    environment:
      STORE_BACKEND: mysql
      MYSQL_DSN: root:root@tcp(mysql:3306)/didi?parseTime=true&charset=utf8mb4&loc=Local
    expose:
      - "8080"
    depends_on:
      - mysql
      - redis
      - kafka
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  admin-service:
    build: ../backend
    command: ["/bin/didi-admin-service"]
    environment:
      ADMIN_MYSQL_DSN: root:root@tcp(mysql:3306)/didi?parseTime=true&charset=utf8mb4&loc=Local
    expose:
      - "18089"
    depends_on:
      mysql:
        condition: service_started
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://localhost:18089/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  gateway:
    build: ../backend
    command: ["/bin/didi-gateway"]
    environment:
      API_BASE: http://api:8080
      AUTH_BASE: http://auth-service:18081
      ADMIN_BASE: http://admin-service:18089
    ports:
      - "8080:8080"
    depends_on:
      api:
        condition: service_healthy
      auth-service:
        condition: service_healthy
      admin-service:
        condition: service_healthy
```

Keep the existing `mysql`, `redis`, `kafka`, `auth-service`, and `frontend` sections unchanged except for dependency ordering required by YAML validity.

- [ ] **Step 2: Validate Compose config**

Run:

```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml config
```

Expected: command exits 0 and includes `admin-service`, `STORE_BACKEND: mysql`, and `ADMIN_BASE: http://admin-service:18089`.

- [ ] **Step 3: Commit**

```bash
git add infra/docker-compose.yml
git commit -m "chore: wire compose shared mysql admin slice

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 11: Add Compose Integration Proof for Admin Shared State

**Files:**
- Modify: `backend/integration/order_dispatch_flow_test.go`

- [ ] **Step 1: Add helper for authenticated requests and admin read test**

Append to `backend/integration/order_dispatch_flow_test.go`:

```go

func TestAdminDriversReadSharedStateViaGateway(t *testing.T) {
	token := loginViaGateway(t, "13700000001", "ADMIN")
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/admin/drivers", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected admin drivers success: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Data []struct {
			ID         string `json:"id"`
			Phone      string `json:"phone"`
			AuditState string `json:"auditState"`
			WorkStatus string `json:"workStatus"`
			PlateNo    string `json:"plateNo"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if len(body.Data) < 2 {
		t.Fatalf("expected seeded shared drivers, got %#v", body.Data)
	}
	found := false
	for _, driver := range body.Data {
		if driver.Phone == "13900000001" && driver.AuditState == "APPROVED" && driver.PlateNo == "京A12345" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected admin-service to read api-seeded driver from MySQL, got %#v", body.Data)
	}
}

func TestPassengerCannotReadAdminDriversViaGateway(t *testing.T) {
	token := loginViaGateway(t, "13800000001", "PASSENGER")
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/admin/drivers", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected forbidden response: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func loginViaGateway(t *testing.T, phone, role string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"phone": phone, "code": "123456", "role": role})
	if err != nil {
		t.Fatalf("marshal login payload: %v", err)
	}
	res, err := http.Post("http://localhost:8080/api/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("expected login success: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body.Data.AccessToken == "" {
		t.Fatal("expected access token")
	}
	return body.Data.AccessToken
}
```

- [ ] **Step 2: Run integration tests before Compose rebuild**

Run:

```bash
go -C /root/didi/backend test ./integration
```

Expected: likely FAIL if Compose is not running with rebuilt services. This confirms integration depends on the container environment.

- [ ] **Step 3: Commit**

```bash
git add backend/integration/order_dispatch_flow_test.go
git commit -m "test: prove admin read uses shared state

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 12: Validate Full Phase 4 Migration

**Files:**
- No source edits expected.

- [ ] **Step 1: Run focused unit tests**

Run:

```bash
go -C /root/didi/backend test ./internal/auth ./internal/gateway ./internal/http ./internal/store ./internal/driver ./internal/admin ./internal/domain
```

Expected: PASS.

- [ ] **Step 2: Rebuild and start Compose**

Run:

```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml up --build -d
```

Expected: images build successfully and services start in detached mode.

- [ ] **Step 3: Check service health**

Run:

```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml ps
```

Expected: `api`, `auth-service`, `admin-service`, `gateway`, and `frontend` are running; healthchecks for `api`, `auth-service`, and `admin-service` are healthy.

- [ ] **Step 4: Run integration tests**

Run:

```bash
go -C /root/didi/backend test ./integration
```

Expected: PASS, including auth send-code/login, protected legacy passenger endpoint, admin shared-state read, and passenger forbidden admin read.

- [ ] **Step 5: Inspect logs if integration fails**

Run:

```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml logs gateway auth-service admin-service api --tail 100
```

Expected when healthy: no startup fatal errors, no MySQL connection errors, and no admin-service auth bypass.

- [ ] **Step 6: Commit validation-only fixes if needed**

If validation reveals deterministic code or config fixes, commit only those changed files:

```bash
git add <fixed-files>
git commit -m "fix: stabilize shared state migration validation

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Self-Review Notes

- Spec coverage: the plan covers MySQL-backed legacy API state, explicit startup failure for MySQL config errors, admin-service read-only MySQL snapshots, gateway admin read cutover, auth preservation, Compose wiring, and integration proof. It intentionally excludes driver/order write service cutover, Kafka consistency, dual writes, admin approval migration, and websocket migration.
- Placeholder scan: no placeholder phrases or unspecified test instructions remain.
- Type consistency: `NewMySQLStore`, `SeedDemoDataForStore`, `DriverRepository`, `NewMySQLDriverRepository`, `AdminBase`, `STORE_BACKEND`, `MYSQL_DSN`, and `ADMIN_MYSQL_DSN` names are used consistently across tasks.

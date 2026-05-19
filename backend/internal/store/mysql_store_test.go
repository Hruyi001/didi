package store

import (
	"os"
	"testing"
	"time"

	"didi/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLStoreContract(t *testing.T) {
	db := openMySQLTestDB(t)
	RunStoreContract(t, func(t *testing.T) Store {
		t.Helper()
		resetMySQLStoreTables(t, db)
		return NewMySQLStore(db)
	})
}

func TestMySQLSeedPassengerUsesExistingAccountAndCreatesMissingProfile(t *testing.T) {
	db := openMySQLTestDB(t)
	resetMySQLStoreTables(t, db)
	now := time.Now()
	account := accountRow{ID: uuid.NewString(), Phone: "13800000011", Role: string(domain.RolePassenger), Status: "ACTIVE", CreatedAt: now}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("seed existing account: %v", err)
	}

	passenger := NewMySQLStore(db).SeedPassenger("13800000011")

	if passenger.ID == "" || passenger.AccountID != account.ID {
		t.Fatalf("expected profile for existing account, got %#v", passenger)
	}
	var profileCount int64
	if err := db.Model(&passengerProfileRow{}).Where("account_id = ?", account.ID).Count(&profileCount).Error; err != nil {
		t.Fatalf("count passenger profiles: %v", err)
	}
	if profileCount != 1 {
		t.Fatalf("expected one passenger profile, got %d", profileCount)
	}
}

func TestMySQLSeedPassengerShortPhoneDoesNotPanic(t *testing.T) {
	db := openMySQLTestDB(t)
	resetMySQLStoreTables(t, db)

	passenger := NewMySQLStore(db).SeedPassenger("123")

	if passenger.Nickname != "乘客123" {
		t.Fatalf("expected full short phone in nickname, got %q", passenger.Nickname)
	}
}

func TestMySQLSeedApprovedDriverShortPhoneDoesNotPanic(t *testing.T) {
	db := openMySQLTestDB(t)
	resetMySQLStoreTables(t, db)

	driver := NewMySQLStore(db).SeedApprovedDriver("12", "京D00012")

	if driver.Name != "司机12" {
		t.Fatalf("expected full short phone in name, got %q", driver.Name)
	}
}

func TestMySQLSeedApprovedDriverRepairsExistingDriverMissingVehicle(t *testing.T) {
	db := openMySQLTestDB(t)
	resetMySQLStoreTables(t, db)
	now := time.Now()
	account := accountRow{ID: uuid.NewString(), Phone: "13900000001", Role: string(domain.RoleDriver), Status: "ACTIVE", CreatedAt: now}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("seed existing account: %v", err)
	}
	existingDriver := driverProfileRow{ID: uuid.NewString(), AccountID: account.ID, Name: "司机0001", Phone: account.Phone, AuditState: "APPROVED", WorkStatus: string(domain.DriverOffline), CreatedAt: now}
	if err := db.Create(&existingDriver).Error; err != nil {
		t.Fatalf("seed existing driver: %v", err)
	}

	driver := NewMySQLStore(db).SeedApprovedDriver("13900000001", "京A12345")

	if driver.ID != existingDriver.ID || driver.Vehicle.PlateNo != "京A12345" {
		t.Fatalf("expected existing driver repaired with vehicle plate, got %#v", driver)
	}
	var vehicle vehicleRow
	if err := db.Where("driver_id = ?", existingDriver.ID).First(&vehicle).Error; err != nil {
		t.Fatalf("expected repaired vehicle row: %v", err)
	}
	if vehicle.PlateNo != "京A12345" {
		t.Fatalf("expected repaired vehicle plate 京A12345, got %q", vehicle.PlateNo)
	}
}

func TestMySQLSeedApprovedDriverRepairsExistingDriverEmptyVehiclePlate(t *testing.T) {
	db := openMySQLTestDB(t)
	resetMySQLStoreTables(t, db)
	now := time.Now()
	account := accountRow{ID: uuid.NewString(), Phone: "13900000001", Role: string(domain.RoleDriver), Status: "ACTIVE", CreatedAt: now}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("seed existing account: %v", err)
	}
	existingDriver := driverProfileRow{ID: uuid.NewString(), AccountID: account.ID, Name: "司机0001", Phone: account.Phone, AuditState: "APPROVED", WorkStatus: string(domain.DriverOffline), CreatedAt: now}
	if err := db.Create(&existingDriver).Error; err != nil {
		t.Fatalf("seed existing driver: %v", err)
	}
	existingVehicle := vehicleRow{ID: uuid.NewString(), DriverID: existingDriver.ID, PlateNo: "", Model: "", Color: "", AuditState: "APPROVED"}
	if err := db.Create(&existingVehicle).Error; err != nil {
		t.Fatalf("seed existing empty vehicle: %v", err)
	}

	driver := NewMySQLStore(db).SeedApprovedDriver("13900000001", "京A12345")

	if driver.Vehicle.PlateNo != "京A12345" {
		t.Fatalf("expected existing empty vehicle plate repaired, got %#v", driver.Vehicle)
	}
	var vehicle vehicleRow
	if err := db.Where("id = ?", existingVehicle.ID).First(&vehicle).Error; err != nil {
		t.Fatalf("expected repaired vehicle row: %v", err)
	}
	if vehicle.PlateNo != "京A12345" {
		t.Fatalf("expected repaired vehicle plate 京A12345, got %q", vehicle.PlateNo)
	}
}

func openMySQLTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DIDI_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set DIDI_MYSQL_TEST_DSN to run MySQL store tests")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	return db
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

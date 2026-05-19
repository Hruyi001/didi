package store

import (
	"errors"
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

func TestSeedDemoDataForStoreReturnsWorkStatusErrors(t *testing.T) {
	s := &seedFailingStore{MemoryStore: NewMemoryStore()}

	err := SeedDemoDataForStore(s)

	assertStoreError(t, err, "set demo driver 1 online: seed status failed")
}

func assertStoreError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %q, got nil", want)
	}
	if err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func assertStorePanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic %q, got nil", want)
		}
		if got := recoveredErrorMessage(recovered); got != want {
			t.Fatalf("expected panic %q, got %q", want, got)
		}
	}()
	fn()
}

func recoveredErrorMessage(recovered any) string {
	if err, ok := recovered.(error); ok {
		return err.Error()
	}
	if message, ok := recovered.(string); ok {
		return message
	}
	return ""
}

type seedFailingStore struct {
	*MemoryStore
}

func (s *seedFailingStore) SetDriverWorkStatus(driverID string, status domain.DriverWorkStatus) (domain.DriverProfile, error) {
	return domain.DriverProfile{}, errors.New("seed status failed")
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
		offer, err := s.AddDispatchAttempt(domain.DispatchAttempt{
			DispatchTaskID:   task.ID,
			OrderID:          order.ID,
			DriverID:         driver.ID,
			Status:           domain.DispatchOffered,
			DistanceToPickup: 1.2,
			TimeoutAt:        time.Now().Add(20 * time.Second),
			SequenceNo:       1,
		})
		if err != nil {
			t.Fatalf("expected offered attempt: %v", err)
		}
		if offer.ID == "" || offer.Status != domain.DispatchOffered || offer.DispatchTaskID != task.ID {
			t.Fatalf("expected offered attempt with dispatch task, got %#v", offer)
		}
		attempts := s.ListDispatchAttempts(order.ID)
		if len(attempts) != 1 || attempts[0].DriverID != driver.ID || attempts[0].DispatchTaskID != task.ID {
			t.Fatalf("expected listed attempt with dispatch task, got %#v", attempts)
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
		duplicatePayment := s.CreatePayment(order.ID, ended.FinalAmount)
		if duplicatePayment.ID != payment.ID || duplicatePayment.Amount != payment.Amount || duplicatePayment.Status != payment.Status {
			t.Fatalf("expected duplicate payment creation to return existing payment, got first=%#v duplicate=%#v", payment, duplicatePayment)
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
		if !ok || storedOrder.ReviewStatus != domain.ReviewReviewed || storedOrder.Version != completed.Version+1 {
			t.Fatalf("expected reviewed order with one version increment, ok=%v order=%#v completed=%#v", ok, storedOrder, completed)
		}

		updatedReview := s.CreateReview(order.ID, 4, "还不错")
		if updatedReview.ID == "" || updatedReview.OrderID != order.ID || updatedReview.Score != 4 || updatedReview.Content != "还不错" {
			t.Fatalf("expected replacement review for order, got %#v", updatedReview)
		}
		storedOrder, ok = s.GetOrder(order.ID)
		if !ok || storedOrder.ReviewStatus != domain.ReviewReviewed || storedOrder.Version != completed.Version+2 {
			t.Fatalf("expected duplicate review call to increment version once, ok=%v order=%#v completed=%#v", ok, storedOrder, completed)
		}
	})

	t.Run("business errors match memory store semantics", func(t *testing.T) {
		s := newStore(t)
		_, err := s.SetDriverWorkStatus("missing-driver", domain.DriverOnlineIdle)
		assertStoreError(t, err, "driver not found")

		_, err = s.UpdateOrderStatus("missing-order", domain.OrderDispatching)
		assertStoreError(t, err, "order not found")

		_, err = s.AssignDriver("missing-order", "missing-driver")
		assertStoreError(t, err, "order not found")

		_, err = s.AddDispatchAttempt(domain.DispatchAttempt{DriverID: "missing-driver", Status: domain.DispatchOffered})
		assertStoreError(t, err, "cannot offer dispatch attempt to unclaimable driver missing-driver")

		_, err = s.MarkPaymentPaid("missing-order")
		assertStoreError(t, err, "payment not found")

		assertStorePanic(t, "order not found", func() {
			s.CreateReview("missing-order", 5, "很好")
		})
	})
}

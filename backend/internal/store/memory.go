package store

import (
	"errors"
	"fmt"
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
	driverLocations  map[string]domain.DriverLocation
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
		driverLocations:  map[string]domain.DriverLocation{},
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

func (s *MemoryStore) GetPassengerByPhone(phone string) (domain.PassengerProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, passenger := range s.passengers {
		account, ok := s.accounts[passenger.AccountID]
		if ok && account.Phone == phone {
			return passenger, true
		}
	}
	return domain.PassengerProfile{}, false
}

func (s *MemoryStore) GetDriverByPhone(phone string) (domain.DriverProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, driver := range s.drivers {
		if driver.Phone == phone {
			return driver, true
		}
	}
	return domain.DriverProfile{}, false
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

func (s *MemoryStore) UpdateDriverLocation(driverID string, location domain.DriverLocation) (domain.DriverLocation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.drivers[driverID]; !ok {
		return domain.DriverLocation{}, errors.New("driver not found")
	}
	location.DriverID = driverID
	location.UpdatedAt = time.Now()
	s.driverLocations[driverID] = location
	return location, nil
}

func (s *MemoryStore) GetDriverLocation(driverID string) (domain.DriverLocation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	location, ok := s.driverLocations[driverID]
	return location, ok
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

func (s *MemoryStore) ListOrdersByPassenger(passengerID string) []domain.RideOrder {
	orders := s.ListOrders()
	result := make([]domain.RideOrder, 0, len(orders))
	for _, order := range orders {
		if order.PassengerID == passengerID {
			result = append(result, order)
		}
	}
	return result
}

func (s *MemoryStore) ListOrdersForDriver(driverID string) []domain.RideOrder {
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
	return result
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

func (s *MemoryStore) AddDispatchAttempt(attempt domain.DispatchAttempt) (domain.DispatchAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	attempt.ID = uuid.NewString()
	if attempt.OfferedAt.IsZero() {
		attempt.OfferedAt = time.Now()
	}
	if attempt.Status == domain.DispatchOffered {
		driver, ok := s.drivers[attempt.DriverID]
		if !ok || driver.WorkStatus != domain.DriverOnlineIdle {
			return domain.DispatchAttempt{}, fmt.Errorf("cannot offer dispatch attempt to unclaimable driver %s", attempt.DriverID)
		}
		driver.WorkStatus = domain.DriverDispatched
		s.drivers[driver.ID] = driver
	}
	s.dispatchAttempts[attempt.OrderID] = append(s.dispatchAttempts[attempt.OrderID], attempt)
	return attempt, nil
}

func (s *MemoryStore) ResetDriverToIdle(driverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	driver, ok := s.drivers[driverID]
	if !ok {
		return errors.New("driver not found")
	}
	if driver.WorkStatus == domain.DriverDispatched {
		driver.WorkStatus = domain.DriverOnlineIdle
		s.drivers[driverID] = driver
	}
	return nil
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
	if payment, ok := s.payments[orderID]; ok {
		return payment
	}
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
	order, ok := s.orders[orderID]
	if !ok {
		panic("order not found")
	}
	review := domain.Review{ID: uuid.NewString(), OrderID: orderID, Score: score, Content: content, Status: domain.ReviewReviewed, CreatedAt: time.Now()}
	s.reviews[orderID] = review
	order.ReviewStatus = domain.ReviewReviewed
	order.Version++
	s.orders[orderID] = order
	return review
}

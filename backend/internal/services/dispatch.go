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
	attempt, err := s.store.AddDispatchAttempt(domain.DispatchAttempt{DispatchTaskID: task.ID, OrderID: orderID, DriverID: driver.ID, Status: domain.DispatchOffered, DistanceToPickup: 1.2, TimeoutAt: time.Now().Add(20 * time.Second), SequenceNo: 1})
	if err != nil {
		return domain.DispatchAttempt{}, err
	}
	return attempt, nil
}

func (s *DispatchService) Accept(orderID, driverID string) (domain.RideOrder, error) {
	if err := s.ensureCurrentOffer(orderID, driverID); err != nil {
		return domain.RideOrder{}, err
	}
	order, err := s.store.AssignDriver(orderID, driverID)
	if err != nil {
		return domain.RideOrder{}, err
	}
	attempts := s.store.ListDispatchAttempts(orderID)
	latest := attempts[len(attempts)-1]
	if _, err := s.store.AddDispatchAttempt(domain.DispatchAttempt{DispatchTaskID: latest.DispatchTaskID, OrderID: orderID, DriverID: driverID, Status: domain.DispatchAccepted, OfferedAt: time.Now(), TimeoutAt: time.Now(), SequenceNo: len(attempts) + 1}); err != nil {
		return domain.RideOrder{}, err
	}
	return order, nil
}

func (s *DispatchService) Reject(orderID, driverID string, reason string) (domain.DispatchAttempt, error) {
	if err := s.ensureCurrentOffer(orderID, driverID); err != nil {
		return domain.DispatchAttempt{}, err
	}
	return s.advance(orderID, driverID, domain.DispatchRejected, reason, time.Now())
}

func (s *DispatchService) ProcessTimeouts(now time.Time) error {
	for _, order := range s.store.ListOrders() {
		if order.Status != domain.OrderDispatching {
			continue
		}
		attempts := s.store.ListDispatchAttempts(order.ID)
		if len(attempts) == 0 {
			continue
		}
		latest := attempts[len(attempts)-1]
		if latest.Status != domain.DispatchOffered || latest.TimeoutAt.After(now) {
			continue
		}
		if _, err := s.advance(order.ID, latest.DriverID, domain.DispatchTimeout, "司机响应超时", now); err != nil && err.Error() != "附近暂无可用司机" {
			return err
		}
	}
	return nil
}

func (s *DispatchService) ensureCurrentOffer(orderID, driverID string) error {
	attempts := s.store.ListDispatchAttempts(orderID)
	if len(attempts) == 0 {
		return fmt.Errorf("订单当前不可接单")
	}
	latest := attempts[len(attempts)-1]
	if latest.Status != domain.DispatchOffered || latest.DriverID != driverID {
		return fmt.Errorf("当前订单未派给该司机")
	}
	return nil
}

func (s *DispatchService) advance(orderID, driverID string, result domain.DispatchStatus, reason string, now time.Time) (domain.DispatchAttempt, error) {
	attempts := s.store.ListDispatchAttempts(orderID)
	latest := attempts[len(attempts)-1]
	finishedAttempt, err := s.store.AddDispatchAttempt(domain.DispatchAttempt{DispatchTaskID: latest.DispatchTaskID, OrderID: orderID, DriverID: driverID, Status: result, RejectReason: reason, OfferedAt: now, TimeoutAt: now, SequenceNo: len(attempts) + 1})
	if err != nil {
		return domain.DispatchAttempt{}, err
	}
	if err := s.store.ResetDriverToIdle(driverID); err != nil {
		return domain.DispatchAttempt{}, err
	}

	offeredDrivers := map[string]struct{}{}
	for _, attempt := range attempts {
		if attempt.Status == domain.DispatchOffered || attempt.Status == domain.DispatchAccepted || attempt.Status == domain.DispatchRejected || attempt.Status == domain.DispatchTimeout {
			offeredDrivers[attempt.DriverID] = struct{}{}
		}
	}
	offeredDrivers[driverID] = struct{}{}

	drivers := s.store.FindOnlineIdleDrivers()
	for _, driver := range drivers {
		if _, used := offeredDrivers[driver.ID]; used {
			continue
		}
		nextAttempt, err := s.store.AddDispatchAttempt(domain.DispatchAttempt{
			DispatchTaskID:   latest.DispatchTaskID,
			OrderID:          orderID,
			DriverID:         driver.ID,
			Status:           domain.DispatchOffered,
			DistanceToPickup: 1.2,
			TimeoutAt:        now.Add(20 * time.Second),
			SequenceNo:       len(attempts) + 2,
		})
		if err != nil {
			continue
		}
		return nextAttempt, nil
	}

	_, err = s.store.UpdateOrderStatus(orderID, domain.OrderDispatchFailed)
	if err != nil {
		return domain.DispatchAttempt{}, err
	}
	return finishedAttempt, fmt.Errorf("附近暂无可用司机")
}

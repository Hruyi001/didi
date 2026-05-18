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

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

type DriverLocation struct {
	DriverID  string    `json:"driverId"`
	Lng       float64   `json:"lng"`
	Lat       float64   `json:"lat"`
	SpeedKPH  int       `json:"speedKph"`
	UpdatedAt time.Time `json:"updatedAt"`
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

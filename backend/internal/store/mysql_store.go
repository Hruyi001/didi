package store

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"didi/backend/internal/domain"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ID            string    `gorm:"column:id;primaryKey"`
	AccountID     string    `gorm:"column:account_id"`
	Name          string    `gorm:"column:name"`
	Phone         string    `gorm:"column:phone"`
	AuditState    string    `gorm:"column:audit_state"`
	WorkStatus    string    `gorm:"column:work_status"`
	AcceptedCount int       `gorm:"column:accepted_count"`
	RejectedCount int       `gorm:"column:rejected_count"`
	TimeoutCount  int       `gorm:"column:timeout_count"`
	CreatedAt     time.Time `gorm:"column:created_at"`
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
	passenger, err := s.seedPassenger(phone)
	if err == nil {
		return passenger
	}
	if isDuplicateError(err) {
		passenger, err = s.seedPassenger(phone)
		if err == nil {
			return passenger
		}
	}
	panic(err)
}

func (s *MySQLStore) seedPassenger(phone string) (domain.PassengerProfile, error) {
	var passenger passengerProfileRow
	err := s.db.Transaction(func(tx *gorm.DB) error {
		account, err := lockAccountByPhone(tx, phone)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			now := time.Now()
			account = accountRow{ID: uuid.NewString(), Phone: phone, Role: string(domain.RolePassenger), Status: "ACTIVE", CreatedAt: now}
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
		} else if account.Role != string(domain.RolePassenger) {
			return seedRoleConflictError(phone, account.Role, "passenger")
		}

		if err := tx.Where("account_id = ?", account.ID).First(&passenger).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		now := time.Now()
		passenger = passengerProfileRow{ID: uuid.NewString(), AccountID: account.ID, Nickname: "乘客" + safePhoneSuffix(phone), CreatedAt: now}
		return tx.Create(&passenger).Error
	})
	if err != nil {
		return domain.PassengerProfile{}, err
	}
	return domain.PassengerProfile{ID: passenger.ID, AccountID: passenger.AccountID, Nickname: passenger.Nickname, CreatedAt: passenger.CreatedAt}, nil
}

func (s *MySQLStore) SeedApprovedDriver(phone, plate string) domain.DriverProfile {
	driver, err := s.seedApprovedDriver(phone, plate)
	if err == nil {
		return driver
	}
	if isDuplicateError(err) {
		driver, err = s.seedApprovedDriver(phone, plate)
		if err == nil {
			return driver
		}
	}
	panic(err)
}

func (s *MySQLStore) seedApprovedDriver(phone, plate string) (domain.DriverProfile, error) {
	var driver driverProfileRow
	var vehicle vehicleRow
	err := s.db.Transaction(func(tx *gorm.DB) error {
		account, err := lockAccountByPhone(tx, phone)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			now := time.Now()
			account = accountRow{ID: uuid.NewString(), Phone: phone, Role: string(domain.RoleDriver), Status: "ACTIVE", CreatedAt: now}
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
		} else if account.Role != string(domain.RoleDriver) {
			return seedRoleConflictError(phone, account.Role, "driver")
		}

		if err := tx.Where("account_id = ?", account.ID).First(&driver).Error; err == nil {
			vehicle, err = ensureVehicleForDriverTx(tx, driver.ID, plate)
			return err
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		now := time.Now()
		driver = driverProfileRow{ID: uuid.NewString(), AccountID: account.ID, Name: "司机" + safePhoneSuffix(phone), Phone: phone, AuditState: "APPROVED", WorkStatus: string(domain.DriverOffline), AcceptedCount: 0, RejectedCount: 0, TimeoutCount: 0, CreatedAt: now}
		vehicle = vehicleRow{ID: uuid.NewString(), DriverID: driver.ID, PlateNo: plate, Model: "快车", Color: "白色", AuditState: "APPROVED"}
		if err := tx.Create(&driver).Error; err != nil {
			return err
		}
		return tx.Create(&vehicle).Error
	})
	if err != nil {
		return domain.DriverProfile{}, err
	}
	return rowToDriver(driver, vehicle), nil
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
	return vehicleForDriverTx(s.db, driverID)
}

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
	CancelReason      *string    `gorm:"column:cancel_reason"`
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
		CancelReason:      stringPtrOrNil(order.CancelReason),
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
	cancelReason := ""
	if row.CancelReason != nil {
		cancelReason = *row.CancelReason
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
		CancelReason:      cancelReason,
		Version:           row.Version,
		CreatedAt:         row.CreatedAt,
		AcceptedAt:        row.AcceptedAt,
		ArrivedAt:         row.ArrivedAt,
		StartedAt:         row.StartedAt,
		EndedAt:           row.EndedAt,
		PaidAt:            row.PaidAt,
	}
}

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
		latest, ok := s.latestDispatchAttemptForOrder(order.ID)
		if ok && latest.Status == string(domain.DispatchOffered) && latest.DriverID == driverID {
			result = append(result, order)
		}
	}
	sortOrdersNewestFirst(result)
	return result
}

type latestDispatchAttemptRow struct {
	DriverID string `gorm:"column:driver_id"`
	Status   string `gorm:"column:status"`
}

func (s *MySQLStore) latestDispatchAttemptForOrder(orderID string) (latestDispatchAttemptRow, bool) {
	var row latestDispatchAttemptRow
	err := s.db.Table("dispatch_attempts").
		Select("driver_id, status").
		Where("order_id = ?", orderID).
		Order("sequence_no DESC, offered_at DESC, id DESC").
		Limit(1).
		First(&row).Error
	if err != nil {
		return latestDispatchAttemptRow{}, false
	}
	return row, true
}

func (s *MySQLStore) UpdateOrderStatus(orderID string, to domain.OrderStatus) (domain.RideOrder, error) {
	var updated domain.RideOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var row rideOrderRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
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
		updates := map[string]any{
			"status":  order.Status,
			"version": order.Version,
		}
		switch to {
		case domain.OrderDriverArrived:
			order.ArrivedAt = &now
			updates["arrived_at"] = order.ArrivedAt
		case domain.OrderInProgress:
			order.StartedAt = &now
			updates["started_at"] = order.StartedAt
		case domain.OrderWaitingPayment:
			order.EndedAt = &now
			order.FinalAmount = order.EstimatedAmount
			updates["ended_at"] = order.EndedAt
			updates["final_amount"] = order.FinalAmount
		case domain.OrderCompleted:
			order.PaidAt = &now
			updates["paid_at"] = order.PaidAt
		}
		if err := tx.Model(&rideOrderRow{}).Where("id = ?", orderID).Updates(updates).Error; err != nil {
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
	return domain.DispatchTask{
		ID:               row.ID,
		OrderID:          row.OrderID,
		Status:           domain.DispatchStatus(row.Status),
		CandidateCount:   row.CandidateCount,
		CurrentAttemptNo: row.CurrentAttemptNo,
		MaxAttempts:      row.MaxAttempts,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func rowToDispatchAttempt(row dispatchAttemptRow) domain.DispatchAttempt {
	return domain.DispatchAttempt{
		ID:               row.ID,
		DispatchTaskID:   row.DispatchTaskID,
		OrderID:          row.OrderID,
		DriverID:         row.DriverID,
		Status:           domain.DispatchStatus(row.Status),
		DistanceToPickup: row.DistanceToPickup,
		OfferedAt:        row.OfferedAt,
		RespondedAt:      row.RespondedAt,
		TimeoutAt:        row.TimeoutAt,
		RejectReason:     row.RejectReason,
		SequenceNo:       row.SequenceNo,
	}
}

func (s *MySQLStore) AssignDriver(orderID, driverID string) (domain.RideOrder, error) {
	var assigned domain.RideOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var orderRow rideOrderRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&orderRow).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("order not found")
		} else if err != nil {
			return err
		}

		var latest dispatchAttemptRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_id = ?", orderID).Order("sequence_no DESC, offered_at DESC, id DESC").Limit(1).First(&latest).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("当前订单未派给该司机")
		} else if err != nil {
			return err
		}
		if latest.Status != string(domain.DispatchOffered) || latest.DriverID != driverID {
			return errors.New("当前订单未派给该司机")
		}

		var driverRow driverProfileRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", driverID).First(&driverRow).Error; errors.Is(err, gorm.ErrRecordNotFound) {
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

		if err := tx.Model(&rideOrderRow{}).Where("id = ?", orderID).Updates(map[string]any{
			"driver_id":   driverID,
			"status":      string(order.Status),
			"accepted_at": order.AcceptedAt,
			"version":     order.Version,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&driverProfileRow{}).Where("id = ?", driverID).Updates(map[string]any{
			"work_status":    string(domain.DriverServing),
			"accepted_count": gorm.Expr("accepted_count + ?", 1),
		}).Error; err != nil {
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

func (s *MySQLStore) AddDispatchAttempt(attempt domain.DispatchAttempt) (domain.DispatchAttempt, error) {
	attempt.ID = uuid.NewString()
	if attempt.OfferedAt.IsZero() {
		attempt.OfferedAt = time.Now()
	}
	row := dispatchAttemptRow{ID: attempt.ID, DispatchTaskID: attempt.DispatchTaskID, OrderID: attempt.OrderID, DriverID: attempt.DriverID, Status: string(attempt.Status), DistanceToPickup: attempt.DistanceToPickup, OfferedAt: attempt.OfferedAt, RespondedAt: attempt.RespondedAt, TimeoutAt: attempt.TimeoutAt, RejectReason: attempt.RejectReason, SequenceNo: attempt.SequenceNo}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if attempt.Status == domain.DispatchOffered {
			result := tx.Model(&driverProfileRow{}).
				Where("id = ? AND work_status = ?", attempt.DriverID, string(domain.DriverOnlineIdle)).
				Update("work_status", string(domain.DriverDispatched))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("cannot offer dispatch attempt to unclaimable driver %s", attempt.DriverID)
			}
		}
		return tx.Create(&row).Error
	}); err != nil {
		return domain.DispatchAttempt{}, err
	}
	return attempt, nil
}

func (s *MySQLStore) ListDispatchAttempts(orderID string) []domain.DispatchAttempt {
	var rows []dispatchAttemptRow
	if err := s.db.Where("order_id = ?", orderID).Order("sequence_no ASC, offered_at ASC, id ASC").Find(&rows).Error; err != nil {
		return []domain.DispatchAttempt{}
	}
	attempts := make([]domain.DispatchAttempt, 0, len(rows))
	for _, row := range rows {
		attempts = append(attempts, rowToDispatchAttempt(row))
	}
	return attempts
}

func (s *MySQLStore) ResetDriverToIdle(driverID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var driver driverProfileRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", driverID).First(&driver).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("driver not found")
		} else if err != nil {
			return err
		}
		if driver.WorkStatus == string(domain.DriverDispatched) {
			return tx.Model(&driverProfileRow{}).Where("id = ?", driverID).Update("work_status", string(domain.DriverOnlineIdle)).Error
		}
		return nil
	})
}

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
	return domain.PaymentOrder{
		ID:        row.ID,
		OrderID:   row.OrderID,
		Amount:    row.Amount,
		Status:    domain.PaymentStatus(row.Status),
		CreatedAt: row.CreatedAt,
		PaidAt:    row.PaidAt,
	}
}

func rowToReview(row reviewRow) domain.Review {
	return domain.Review{
		ID:        row.ID,
		OrderID:   row.OrderID,
		Score:     row.Score,
		Content:   row.Content,
		Status:    domain.ReviewStatus(row.Status),
		CreatedAt: row.CreatedAt,
	}
}

func (s *MySQLStore) CreatePayment(orderID string, amount int64) domain.PaymentOrder {
	row := paymentOrderRow{ID: uuid.NewString(), OrderID: orderID, Amount: amount, Status: string(domain.PaymentUnpaid), CreatedAt: time.Now()}
	if err := s.db.Create(&row).Error; err != nil {
		var existing paymentOrderRow
		if findErr := s.db.Where("order_id = ?", orderID).First(&existing).Error; findErr == nil {
			return rowToPayment(existing)
		}
		panic(err)
	}
	return rowToPayment(row)
}

func (s *MySQLStore) MarkPaymentPaid(orderID string) (domain.PaymentOrder, error) {
	var updated domain.PaymentOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var row paymentOrderRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_id = ?", orderID).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("payment not found")
		} else if err != nil {
			return err
		}
		now := time.Now()
		row.Status = string(domain.PaymentPaid)
		row.PaidAt = &now
		if err := tx.Model(&paymentOrderRow{}).Where("id = ?", row.ID).Updates(map[string]any{"status": row.Status, "paid_at": row.PaidAt}).Error; err != nil {
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
		var order rideOrderRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&order).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("order not found")
		} else if err != nil {
			return err
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "order_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"id":         row.ID,
				"score":      row.Score,
				"content":    row.Content,
				"status":     row.Status,
				"created_at": row.CreatedAt,
			}),
		}).Create(&row).Error; err != nil {
			return err
		}

		result := tx.Model(&rideOrderRow{}).Where("id = ?", orderID).Updates(map[string]any{"review_status": string(domain.ReviewReviewed), "version": gorm.Expr("version + ?", 1)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("order not found")
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return rowToReview(row)
}

func lockAccountByPhone(tx *gorm.DB, phone string) (accountRow, error) {
	var account accountRow
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("phone = ?", phone).First(&account).Error
	return account, err
}

func seedRoleConflictError(phone, existingRole, requestedRole string) error {
	return fmt.Errorf("phone %s belongs to role %s; cannot seed %s", phone, existingRole, requestedRole)
}

func vehicleForDriverTx(tx *gorm.DB, driverID string) vehicleRow {
	var vehicle vehicleRow
	_ = tx.Where("driver_id = ?", driverID).First(&vehicle).Error
	return vehicle
}

func ensureVehicleForDriverTx(tx *gorm.DB, driverID, plate string) (vehicleRow, error) {
	var vehicle vehicleRow
	if err := tx.Where("driver_id = ?", driverID).First(&vehicle).Error; err == nil {
		updates := map[string]any{}
		if vehicle.PlateNo == "" {
			updates["plate_no"] = plate
			vehicle.PlateNo = plate
		}
		if vehicle.Model == "" {
			updates["model"] = "快车"
			vehicle.Model = "快车"
		}
		if vehicle.Color == "" {
			updates["color"] = "白色"
			vehicle.Color = "白色"
		}
		if vehicle.AuditState == "" {
			updates["audit_state"] = "APPROVED"
			vehicle.AuditState = "APPROVED"
		}
		if len(updates) > 0 {
			return vehicle, tx.Model(&vehicleRow{}).Where("id = ?", vehicle.ID).Updates(updates).Error
		}
		return vehicle, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return vehicleRow{}, err
	}
	vehicle = vehicleRow{ID: uuid.NewString(), DriverID: driverID, PlateNo: plate, Model: "快车", Color: "白色", AuditState: "APPROVED"}
	if err := tx.Create(&vehicle).Error; err != nil {
		if isDuplicateError(err) {
			var existing vehicleRow
			if readErr := tx.Where("driver_id = ?", driverID).First(&existing).Error; readErr != nil {
				return vehicleRow{}, readErr
			}
			return existing, nil
		}
		return vehicleRow{}, err
	}
	return vehicle, nil
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func safePhoneSuffix(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	return phone[len(phone)-4:]
}

func isDuplicateError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
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
		Stats: domain.DriverStats{
			Accepted: row.AcceptedCount,
			Rejected: row.RejectedCount,
			Timeouts: row.TimeoutCount,
		},
		CreatedAt: row.CreatedAt,
	}
}

func sortOrdersNewestFirst(orders []domain.RideOrder) {
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.After(orders[j].CreatedAt) })
}

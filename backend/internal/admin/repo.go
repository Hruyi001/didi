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
	ID         string `gorm:"column:driver_id"`
	Phone      string `gorm:"column:phone"`
	AuditState string `gorm:"column:audit_state"`
	WorkStatus string `gorm:"column:work_status"`
	PlateNo    string `gorm:"column:plate_no"`
}

func (r *MySQLDriverRepository) ListDrivers() ([]DriverSnapshot, error) {
	var rows []driverSnapshotRow
	err := r.db.Table("driver_profiles").
		Select("driver_profiles.id AS driver_id, driver_profiles.phone, driver_profiles.audit_state, driver_profiles.work_status, vehicles.plate_no AS plate_no").
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

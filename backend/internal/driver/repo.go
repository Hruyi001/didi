package driver

type MemoryRepo struct {
	drivers map[string]Driver
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{drivers: map[string]Driver{}}
}

func (r *MemoryRepo) SeedApproved(id, phone string) Driver {
	driver := Driver{ID: id, Phone: phone, AuditState: "APPROVED", WorkStatus: "OFFLINE"}
	r.drivers[id] = driver
	return driver
}

func (r *MemoryRepo) Get(driverID string) (Driver, bool) {
	driver, ok := r.drivers[driverID]
	return driver, ok
}

func (r *MemoryRepo) Save(driver Driver) {
	r.drivers[driver.ID] = driver
}

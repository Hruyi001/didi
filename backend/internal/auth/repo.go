package auth

import "didi/backend/internal/contracts"

type MemoryRepo struct {
	passengers map[string]struct{}
	drivers    map[string]struct{}
	admins     map[string]struct{}
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		passengers: map[string]struct{}{"13800000001": {}},
		drivers: map[string]struct{}{
			"13900000001": {},
			"13900000002": {},
		},
		admins: map[string]struct{}{"13700000001": {}},
	}
}

func (r *MemoryRepo) IdentityExists(phone string, role contracts.AccountRole) bool {
	switch role {
	case contracts.RolePassenger:
		_, ok := r.passengers[phone]
		return ok
	case contracts.RoleDriver:
		_, ok := r.drivers[phone]
		return ok
	case contracts.RoleAdmin:
		_, ok := r.admins[phone]
		return ok
	default:
		return false
	}
}

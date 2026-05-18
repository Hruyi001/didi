package user

import "github.com/google/uuid"

type MemoryRepo struct {
	profiles map[string]PassengerProfile
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{profiles: map[string]PassengerProfile{}}
}

func (r *MemoryRepo) EnsurePassenger(accountID, phone string) PassengerProfile {
	if profile, ok := r.profiles[accountID]; ok {
		return profile
	}
	profile := PassengerProfile{
		ID:        uuid.NewString(),
		AccountID: accountID,
		Phone:     phone,
		Nickname:  "乘客" + phone[len(phone)-4:],
	}
	r.profiles[accountID] = profile
	return profile
}

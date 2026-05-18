package dispatch

type MemoryRepo struct {
	attempts []Attempt
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{attempts: []Attempt{}}
}

func (r *MemoryRepo) Save(attempt Attempt) {
	r.attempts = append(r.attempts, attempt)
}

package location

import "sort"

type DriverPoint struct {
	DriverID  string  `json:"driverId"`
	Lng       float64 `json:"lng"`
	Lat       float64 `json:"lat"`
	DistanceM int     `json:"distanceM"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) Nearby(lng, lat float64, limit int) []DriverPoint {
	points := s.repo.All()
	sort.Slice(points, func(i, j int) bool { return points[i].DistanceM < points[j].DistanceM })
	if len(points) > limit {
		return points[:limit]
	}
	return points
}

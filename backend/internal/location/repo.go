package location

type MemoryRepo struct {
	points map[string]DriverPoint
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{points: map[string]DriverPoint{}}
}

func (r *MemoryRepo) Upsert(driverID string, lng, lat float64) {
	r.points[driverID] = DriverPoint{DriverID: driverID, Lng: lng, Lat: lat, DistanceM: distanceFor(driverID)}
}

func (r *MemoryRepo) All() []DriverPoint {
	points := make([]DriverPoint, 0, len(r.points))
	for _, point := range r.points {
		points = append(points, point)
	}
	return points
}

func distanceFor(driverID string) int {
	if driverID == "d1" {
		return 800
	}
	return 1200
}

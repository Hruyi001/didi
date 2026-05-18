package driver

import "testing"

func TestApprovedDriverCanGoOnline(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo)
	driver := repo.SeedApproved("driver-1", "13900000001")
	updated, err := svc.SetOnline(driver.ID)
	if err != nil {
		t.Fatalf("expected driver online: %v", err)
	}
	if updated.WorkStatus != "ONLINE_IDLE" {
		t.Fatalf("expected ONLINE_IDLE, got %s", updated.WorkStatus)
	}
}

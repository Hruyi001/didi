package user

import "testing"

func TestCreatePassengerProfile(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo)
	profile := svc.EnsurePassenger("acct-1", "13800000001")
	if profile.AccountID != "acct-1" {
		t.Fatalf("expected account binding, got %q", profile.AccountID)
	}
}

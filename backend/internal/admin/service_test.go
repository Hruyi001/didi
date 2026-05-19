package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type fakeDriverRepo struct {
	drivers []DriverSnapshot
	err     error
}

func (r fakeDriverRepo) ListDrivers() ([]DriverSnapshot, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.drivers, nil
}

func TestListDriversReturnsRepositorySnapshots(t *testing.T) {
	svc := NewService(fakeDriverRepo{drivers: []DriverSnapshot{{ID: "driver-1", Phone: "13900000001", AuditState: "APPROVED", WorkStatus: "ONLINE_IDLE", PlateNo: "京A12345"}}})
	drivers, err := svc.ListDrivers()
	if err != nil {
		t.Fatalf("expected drivers: %v", err)
	}
	if len(drivers) != 1 {
		t.Fatalf("expected one driver, got %#v", drivers)
	}
	if drivers[0].Phone != "13900000001" || drivers[0].PlateNo != "京A12345" {
		t.Fatalf("expected repository driver data, got %#v", drivers[0])
	}
}

func TestAdminDriversRequiresLogin(t *testing.T) {
	res := requestAdminDrivers(t, fakeDriverRepo{}, "")
	assertErrorResponse(t, res, http.StatusUnauthorized, "请先登录")
}

func TestAdminDriversRejectsInvalidTokenWithLegacyMessage(t *testing.T) {
	res := requestAdminDrivers(t, fakeDriverRepo{}, "Bearer invalid-token")
	assertErrorResponse(t, res, http.StatusUnauthorized, "token error")
}

func TestAdminDriversRequiresAdminRole(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", "admin-test-secret")
	token, _, _, err := services.IssueTokens("13800000001", domain.RolePassenger)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	res := requestAdminDrivers(t, fakeDriverRepo{}, "Bearer "+token)
	assertErrorResponse(t, res, http.StatusForbidden, "无权访问该资源")
}

func TestAdminDriversAllowsAdminToken(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", "admin-test-secret")
	token, _, _, err := services.IssueTokens("13800000002", domain.RoleAdmin)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	res := requestAdminDrivers(t, fakeDriverRepo{drivers: []DriverSnapshot{{ID: "driver-2", Phone: "13900000002", AuditState: "APPROVED", WorkStatus: "ONLINE_IDLE", PlateNo: "京B12345"}}}, "Bearer "+token)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body %s", http.StatusOK, res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "driver-2") {
		t.Fatalf("expected admin driver data, got %s", res.Body.String())
	}
}

func TestAdminDriversHidesRepositoryErrors(t *testing.T) {
	t.Setenv("AUTH_TOKEN_SECRET", "admin-test-secret")
	token, _, _, err := services.IssueTokens("13800000003", domain.RoleAdmin)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	repoErr := errors.New("mysql password leaked in raw error")

	res := requestAdminDrivers(t, fakeDriverRepo{err: repoErr}, "Bearer "+token)

	assertErrorResponse(t, res, http.StatusInternalServerError, "查询司机列表失败")
	if strings.Contains(res.Body.String(), repoErr.Error()) {
		t.Fatalf("expected response to hide repo error, got %s", res.Body.String())
	}
}

func TestMain(m *testing.M) {
	os.Unsetenv("AUTH_TOKEN_SECRET")
	os.Exit(m.Run())
}

func requestAdminDrivers(t *testing.T, repo fakeDriverRepo, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterHTTP(r, NewService(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/drivers", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

func assertErrorResponse(t *testing.T, res *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if res.Code != status {
		t.Fatalf("expected status %d, got %d body %s", status, res.Code, res.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error != message {
		t.Fatalf("expected error %q, got %q", message, body.Error)
	}
}

package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"aiops/internal/model"
)

func TestAdminHandler_Login_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	// Create a user with bcrypt password
	adminRepo.CreateUser(&model.User{
		TenantID: 1, Username: "admin", PasswordHash: hashPassword("admin123"),
		Role: "admin", Status: "active",
	})

	c, w := setupGinTestWithBody("POST", "/api/v1/auth/login", `{"username":"admin","password":"admin123"}`)
	h.Login(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil {
		t.Fatal("response should contain token")
	}
	if resp["username"] != "admin" {
		t.Errorf("username = %v, want admin", resp["username"])
	}
}

func TestAdminHandler_Login_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	adminRepo.CreateUser(&model.User{
		TenantID: 1, Username: "admin", PasswordHash: hashPassword("admin123"),
		Role: "admin", Status: "active",
	})

	c, w := setupGinTestWithBody("POST", "/api/v1/auth/login", `{"username":"admin","password":"wrong"}`)
	h.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAdminHandler_Login_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/auth/login", `{"username":"nonexistent","password":"pass"}`)
	h.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAdminHandler_Login_DisabledUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	adminRepo.CreateUser(&model.User{
		TenantID: 1, Username: "disabled", PasswordHash: hashPassword("pass"),
		Role: "viewer", Status: "disabled",
	})

	c, w := setupGinTestWithBody("POST", "/api/v1/auth/login", `{"username":"disabled","password":"pass"}`)
	h.Login(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestAdminHandler_Login_AccountLockout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	adminRepo.CreateUser(&model.User{
		TenantID: 1, Username: "locktest", PasswordHash: hashPassword("correct"),
		Role: "viewer", Status: "active",
	})

	// Fail 5 times
	for i := 0; i < 5; i++ {
		c, _ := setupGinTestWithBody("POST", "/api/v1/auth/login", `{"username":"locktest","password":"wrong"}`)
		h.Login(c)
	}

	// 6th attempt should be locked out
	c, w := setupGinTestWithBody("POST", "/api/v1/auth/login", `{"username":"locktest","password":"correct"}`)
	h.Login(c)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429 (locked out)", w.Code)
	}
}

func TestAdminHandler_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/users", `{"username":"newuser","password":"NewPass123!","role":"viewer"}`)
	c.Set("user_id", int64(1))
	c.Set("username", "admin")
	h.CreateUser(c)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201, body: %s", w.Code, w.Body.String())
	}
}

func TestAdminHandler_CreateUser_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/users", `{"username":"badrole","password":"Pass123!","role":"hacker"}`)
	c.Set("user_id", int64(1))
	c.Set("username", "admin")
	h.CreateUser(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAdminHandler_CreateUser_WeakPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/users", `{"username":"weak","password":"123","role":"viewer"}`)
	c.Set("user_id", int64(1))
	c.Set("username", "admin")
	h.CreateUser(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (weak password)", w.Code)
	}
}

func TestAdminHandler_ListUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	adminRepo.CreateUser(&model.User{TenantID: 1, Username: "u1", PasswordHash: "h", Role: "viewer", Status: "active"})
	adminRepo.CreateUser(&model.User{TenantID: 1, Username: "u2", PasswordHash: "h", Role: "admin", Status: "active"})

	c, w := setupGinTest()
	c.Request = setupRequest("GET", "/api/v1/users?limit=10&offset=0")
	h.ListUsers(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAdminHandler_UpdateUser_RoleValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	user := &model.User{TenantID: 1, Username: "target", PasswordHash: "h", Role: "viewer", Status: "active"}
	adminRepo.CreateUser(user)

	// Try to set invalid role
	c, w := setupGinTestWithBody("PUT", "/api/v1/users/1", `{"role":"superadmin"}`)
	c.Set("user_id", int64(999)) // different user
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.UpdateUser(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAdminHandler_UpdateUser_SelfRoleModification(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	user := &model.User{TenantID: 1, Username: "self", PasswordHash: "h", Role: "viewer", Status: "active"}
	adminRepo.CreateUser(user)

	// Try to change own role
	c, w := setupGinTestWithBody("PUT", "/api/v1/users/1", `{"role":"admin"}`)
	c.Set("user_id", user.ID)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.UpdateUser(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (self-role modification)", w.Code)
	}
}

func TestAdminHandler_CreateTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/tenants", `{"code":"new-tenant","name":"New Tenant"}`)
	h.CreateTenant(c)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201, body: %s", w.Code, w.Body.String())
	}
}

func TestAdminHandler_DeleteTenant_DefaultBlocked(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.DeleteTenant(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (cannot delete default tenant)", w.Code)
	}
}

func TestAdminHandler_GenerateAndValidateToken(t *testing.T) {
	token := generateToken(1, 1, "admin")

	// Create handler for validation
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	userID, tenantID, role, err := h.validateToken(token)
	if err != nil {
		t.Fatalf("validateToken: %v", err)
	}
	if userID != 1 {
		t.Errorf("userID = %d, want 1", userID)
	}
	if tenantID != 1 {
		t.Errorf("tenantID = %d, want 1", tenantID)
	}
	if role != "admin" {
		t.Errorf("role = %s, want admin", role)
	}
}

func TestAdminHandler_ValidateToken_Invalid(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	_, _, _, err := h.validateToken("invalid-token")
	if err == nil {
		t.Fatal("invalid token should return error")
	}
}

func TestAdminHandler_ValidateToken_Expired(t *testing.T) {
	// This test verifies the token format parsing
	db := setupTestDB(t)
	defer db.Close()
	adminRepo := createTestAdminRepo(db)
	tenantRepo := createTestTenantRepo(db)
	h := NewAdminHandler(adminRepo, tenantRepo)

	// A malformed token should fail
	_, _, _, err := h.validateToken("aaaa:bbbb")
	if err == nil {
		t.Fatal("malformed token should return error")
	}
}

// setupRequest creates an HTTP request and sets it on the context.
func setupRequest(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	return req
}

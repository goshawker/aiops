package repo

import (
	"testing"

	"aiops/internal/model"
)

func TestAdminRepo_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	user := &model.User{
		TenantID:     1,
		Username:     "testuser",
		PasswordHash: "hashed_password",
		DisplayName:  "Test User",
		Email:        "test@example.com",
		Role:         "viewer",
		Status:       "active",
	}

	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("CreateUser should set user ID")
	}
}

func TestAdminRepo_CreateUser_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	user := &model.User{TenantID: 1, Username: "dup", PasswordHash: "hash", Role: "viewer", Status: "active"}
	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	user2 := &model.User{TenantID: 1, Username: "dup", PasswordHash: "hash", Role: "viewer", Status: "active"}
	if err := repo.CreateUser(user2); err == nil {
		t.Fatal("duplicate username should fail")
	}
}

func TestAdminRepo_GetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	user := &model.User{TenantID: 1, Username: "lookup", PasswordHash: "hash", Role: "admin", Status: "active"}
	repo.CreateUser(user)

	found, err := repo.GetUserByUsername("lookup")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if found == nil {
		t.Fatal("user not found")
	}
	if found.Username != "lookup" {
		t.Errorf("username = %s, want lookup", found.Username)
	}
	if found.Role != "admin" {
		t.Errorf("role = %s, want admin", found.Role)
	}
}

func TestAdminRepo_GetUserByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	found, err := repo.GetUserByUsername("nonexistent")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if found != nil {
		t.Fatal("should return nil for nonexistent user")
	}
}

func TestAdminRepo_GetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	user := &model.User{TenantID: 1, Username: "byid", PasswordHash: "hash", Role: "viewer", Status: "active"}
	repo.CreateUser(user)

	found, err := repo.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if found == nil || found.Username != "byid" {
		t.Fatal("user not found by ID")
	}
}

func TestAdminRepo_ListUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	for i := 0; i < 5; i++ {
		repo.CreateUser(&model.User{TenantID: 1, Username: "user" + string(rune('a'+i)), PasswordHash: "h", Role: "viewer", Status: "active"})
	}

	users, total, err := repo.ListUsers(3, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(users) != 3 {
		t.Errorf("len = %d, want 3", len(users))
	}
}

func TestAdminRepo_UpdateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	user := &model.User{TenantID: 1, Username: "update", PasswordHash: "hash", Role: "viewer", Status: "active"}
	repo.CreateUser(user)

	user.DisplayName = "Updated Name"
	user.Role = "operator"
	if err := repo.UpdateUser(user); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	found, _ := repo.GetUserByID(user.ID)
	if found.DisplayName != "Updated Name" {
		t.Errorf("display_name = %s, want Updated Name", found.DisplayName)
	}
	if found.Role != "operator" {
		t.Errorf("role = %s, want operator", found.Role)
	}
}

func TestAdminRepo_DeleteUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	user := &model.User{TenantID: 1, Username: "delete", PasswordHash: "hash", Role: "viewer", Status: "active"}
	repo.CreateUser(user)

	if err := repo.DeleteUser(user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	found, _ := repo.GetUserByID(user.ID)
	if found != nil {
		t.Fatal("user should be deleted")
	}
}

func TestAdminRepo_CountUsersByTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	repo.CreateUser(&model.User{TenantID: 1, Username: "u1", PasswordHash: "h", Role: "viewer", Status: "active"})
	repo.CreateUser(&model.User{TenantID: 1, Username: "u2", PasswordHash: "h", Role: "viewer", Status: "active"})

	var count int
	if err := repo.CountUsersByTenant(1, &count); err != nil {
		t.Fatalf("CountUsersByTenant: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestAdminRepo_InsertAuditLog(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	log := &model.AuditLog{
		UserID:   1,
		Username: "admin",
		Action:   "login",
		Resource: "auth",
		IP:       "127.0.0.1",
	}

	if err := repo.InsertAuditLog(log); err != nil {
		t.Fatalf("InsertAuditLog: %v", err)
	}
	if log.ID == 0 {
		t.Fatal("InsertAuditLog should set ID")
	}

	// Insert a second log to verify hash chain
	log2 := &model.AuditLog{
		UserID:   1,
		Username: "admin",
		Action:   "create",
		Resource: "user",
		IP:       "127.0.0.1",
	}
	repo.InsertAuditLog(log2)
	if log2.ID <= log.ID {
		t.Fatal("second log should have higher ID")
	}
}

func TestAdminRepo_ListAuditLogs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	repo.InsertAuditLog(&model.AuditLog{UserID: 1, Username: "admin", Action: "login", Resource: "auth"})
	repo.InsertAuditLog(&model.AuditLog{UserID: 1, Username: "admin", Action: "create", Resource: "user"})
	repo.InsertAuditLog(&model.AuditLog{UserID: 1, Username: "admin", Action: "logout", Resource: "auth"})

	logs, total, err := repo.ListAuditLogs(10, 0, 0, "")
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(logs) != 3 {
		t.Errorf("len = %d, want 3", len(logs))
	}
}

func TestAdminRepo_ListAuditLogs_FilterByAction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAdminRepo(db)

	repo.InsertAuditLog(&model.AuditLog{UserID: 1, Username: "admin", Action: "login", Resource: "auth"})
	repo.InsertAuditLog(&model.AuditLog{UserID: 1, Username: "admin", Action: "create", Resource: "user"})

	logs, total, err := repo.ListAuditLogs(10, 0, 0, "login")
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(logs) != 1 || logs[0].Action != "login" {
		t.Error("should filter by action")
	}
}

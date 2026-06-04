package repo

import (
	"testing"

	"aiops/internal/model"
)

func TestTenantRepo_CreateTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTenantRepo(db)

	tenant := &model.Tenant{
		Code:   "test-tenant",
		Name:   "Test Tenant",
		Plan:   "pro",
		Status: "active",
	}

	if err := repo.CreateTenant(tenant); err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	if tenant.ID == 0 {
		t.Fatal("CreateTenant should set ID")
	}
}

func TestTenantRepo_CreateTenant_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTenantRepo(db)

	t1 := &model.Tenant{Code: "dup", Name: "Dup 1"}
	repo.CreateTenant(t1)

	t2 := &model.Tenant{Code: "dup", Name: "Dup 2"}
	if err := repo.CreateTenant(t2); err == nil {
		t.Fatal("duplicate code should fail")
	}
}

func TestTenantRepo_GetTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTenantRepo(db)

	tenant := &model.Tenant{Code: "get-test", Name: "Get Test"}
	repo.CreateTenant(tenant)

	found, err := repo.GetTenant(tenant.ID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if found == nil {
		t.Fatal("tenant not found")
	}
	if found.Code != "get-test" {
		t.Errorf("code = %s, want get-test", found.Code)
	}
}

func TestTenantRepo_GetTenant_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTenantRepo(db)

	found, err := repo.GetTenant(999)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if found != nil {
		t.Fatal("should return nil for nonexistent tenant")
	}
}

func TestTenantRepo_ListTenants(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTenantRepo(db)

	repo.CreateTenant(&model.Tenant{Code: "t1", Name: "T1"})
	repo.CreateTenant(&model.Tenant{Code: "t2", Name: "T2"})
	repo.CreateTenant(&model.Tenant{Code: "t3", Name: "T3"})

	tenants, total, err := repo.ListTenants(10, 0)
	if err != nil {
		t.Fatalf("ListTenants: %v", err)
	}
	// 3 created + 1 default = 4
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(tenants) != 4 {
		t.Errorf("len = %d, want 4", len(tenants))
	}
}

func TestTenantRepo_UpdateTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTenantRepo(db)

	tenant := &model.Tenant{Code: "upd", Name: "Old Name"}
	repo.CreateTenant(tenant)

	tenant.Name = "New Name"
	tenant.Plan = "enterprise"
	if err := repo.UpdateTenant(tenant); err != nil {
		t.Fatalf("UpdateTenant: %v", err)
	}

	found, _ := repo.GetTenant(tenant.ID)
	if found.Name != "New Name" {
		t.Errorf("name = %s, want New Name", found.Name)
	}
	if found.Plan != "enterprise" {
		t.Errorf("plan = %s, want enterprise", found.Plan)
	}
}

func TestTenantRepo_DeleteTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTenantRepo(db)

	tenant := &model.Tenant{Code: "del", Name: "Delete Me"}
	repo.CreateTenant(tenant)

	if err := repo.DeleteTenant(tenant.ID); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}

	found, _ := repo.GetTenant(tenant.ID)
	if found != nil {
		t.Fatal("tenant should be deleted")
	}
}

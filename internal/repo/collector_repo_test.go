package repo

import (
	"testing"

	"aiops/internal/model"
)

func TestCollectorRepo_RegisterCollector(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	c := &model.Collector{
		TenantID: 1,
		Name:     "web-01",
		Hostname: "web-01.example.com",
		IP:       "192.168.1.10",
		Version:  "1.0.0",
	}

	if err := repo.RegisterCollector(c); err != nil {
		t.Fatalf("RegisterCollector: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("RegisterCollector should set ID")
	}
}

func TestCollectorRepo_GetCollector(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	c := &model.Collector{TenantID: 1, Name: "get-col", Hostname: "host", IP: "1.2.3.4"}
	repo.RegisterCollector(c)

	found, err := repo.GetCollector(c.ID)
	if err != nil {
		t.Fatalf("GetCollector: %v", err)
	}
	if found == nil {
		t.Fatal("collector not found")
	}
	if found.Name != "get-col" {
		t.Errorf("name = %s, want get-col", found.Name)
	}
}

func TestCollectorRepo_GetCollector_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	found, err := repo.GetCollector(999)
	if err != nil {
		t.Fatalf("GetCollector: %v", err)
	}
	if found != nil {
		t.Fatal("should return nil")
	}
}

func TestCollectorRepo_ListCollectors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	repo.RegisterCollector(&model.Collector{TenantID: 1, Name: "c1", IP: "10.0.0.1"})
	repo.RegisterCollector(&model.Collector{TenantID: 1, Name: "c2", IP: "10.0.0.2"})
	repo.RegisterCollector(&model.Collector{TenantID: 1, Name: "c3", IP: "10.0.0.3"})

	// Set c1 and c3 to online via heartbeat
	c1, _ := repo.GetCollector(1)
	repo.UpdateHeartbeat(c1.ID, &model.CollectorHeartbeat{})
	c3, _ := repo.GetCollector(3)
	repo.UpdateHeartbeat(c3.ID, &model.CollectorHeartbeat{})

	// List all
	all, total, err := repo.ListCollectors(100, 0, "")
	if err != nil {
		t.Fatalf("ListCollectors: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(all) != 3 {
		t.Errorf("len = %d, want 3", len(all))
	}

	// List online only
	online, totalOnline, _ := repo.ListCollectors(100, 0, "online")
	if totalOnline != 2 {
		t.Errorf("online total = %d, want 2", totalOnline)
	}
	if len(online) != 2 {
		t.Errorf("online len = %d, want 2", len(online))
	}
}

func TestCollectorRepo_UpdateHeartbeat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	c := &model.Collector{TenantID: 1, Name: "hb", Status: "offline"}
	repo.RegisterCollector(c)

	hb := &model.CollectorHeartbeat{
		CPU: 50.0, Memory: 60.0, Uptime: 3600,
		Collected: 100, Errors: 0,
	}
	if err := repo.UpdateHeartbeat(c.ID, hb); err != nil {
		t.Fatalf("UpdateHeartbeat: %v", err)
	}

	found, _ := repo.GetCollector(c.ID)
	if found.Status != "online" {
		t.Errorf("status = %s, want online", found.Status)
	}
}

func TestCollectorRepo_DeleteCollector(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	c := &model.Collector{TenantID: 1, Name: "del-col"}
	repo.RegisterCollector(c)

	if err := repo.DeleteCollector(c.ID); err != nil {
		t.Fatalf("DeleteCollector: %v", err)
	}

	found, _ := repo.GetCollector(c.ID)
	if found != nil {
		t.Fatal("collector should be deleted")
	}
}

func TestCollectorRepo_SaveConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	c := &model.Collector{TenantID: 1, Name: "cfg-col"}
	repo.RegisterCollector(c)

	cfg := &model.CollectorConfig{
		CollectorID: c.ID,
		Content:     `{"interval":"30s"}`,
		Version:     1,
	}
	if err := repo.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
}

func TestCollectorRepo_GetLatestConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewCollectorRepo(db)

	c := &model.Collector{TenantID: 1, Name: "cfg-col2"}
	repo.RegisterCollector(c)

	repo.SaveConfig(&model.CollectorConfig{CollectorID: c.ID, Content: `{"v":1}`, Version: 1})
	repo.SaveConfig(&model.CollectorConfig{CollectorID: c.ID, Content: `{"v":2}`, Version: 2})

	cfg, err := repo.GetLatestConfig(c.ID, "")
	if err != nil {
		t.Fatalf("GetLatestConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("config should not be nil")
	}
}

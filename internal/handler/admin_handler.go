package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"aiops/internal/model"
	"aiops/internal/repo"
)

// loginAttempt tracks failed login attempts for account lockout.
type loginAttempt struct {
	count    int
	lastFail time.Time
	locked   bool
}

// AdminHandler handles user management, tenant management, and audit logs.
type AdminHandler struct {
	repo       *repo.AdminRepo
	tenantRepo *repo.TenantRepo

	// Login lockout
	mu          sync.Mutex
	loginLocks  map[string]*loginAttempt
	maxAttempts int
	lockoutTime time.Duration
}

func NewAdminHandler(repo *repo.AdminRepo, tenantRepo *repo.TenantRepo) *AdminHandler {
	return &AdminHandler{
		repo:        repo,
		tenantRepo:  tenantRepo,
		loginLocks:  make(map[string]*loginAttempt),
		maxAttempts: 5,
		lockoutTime: 15 * time.Minute,
	}
}

// isLockedOut checks if an account is currently locked.
func (h *AdminHandler) isLockedOut(username string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	attempt, ok := h.loginLocks[username]
	if !ok {
		return false
	}

	// Auto-unlock after lockout period
	if attempt.locked && time.Since(attempt.lastFail) > h.lockoutTime {
		delete(h.loginLocks, username)
		return false
	}

	return attempt.locked
}

// recordFailedLogin increments the failed attempt counter.
func (h *AdminHandler) recordFailedLogin(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	attempt, ok := h.loginLocks[username]
	if !ok {
		attempt = &loginAttempt{}
		h.loginLocks[username] = attempt
	}

	attempt.count++
	attempt.lastFail = time.Now()

	if attempt.count >= h.maxAttempts {
		attempt.locked = true
	}
}

// clearLoginAttempts resets the failed attempt counter on successful login.
func (h *AdminHandler) clearLoginAttempts(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.loginLocks, username)
}

// --- Auth Middleware ---

// AuthMiddleware validates the session and injects user info.
// Priority 1: Validate HMAC token from Authorization header (direct client access)
// Priority 2: Trust X-User-ID header only from internal Gateway calls (X-Forwarded-For present)
func (h *AdminHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow unauthenticated access for login endpoint
		if c.Request.URL.Path == "/api/v1/auth/login" {
			c.Next()
			return
		}

		// Priority 1: Validate HMAC token directly
		if authHeader := c.GetHeader("Authorization"); authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr := authHeader[7:]
			userID, tenantID, role, err := validateToken(tokenStr)
			if err == nil && userID > 0 {
				c.Set("user_id", userID)
				c.Set("tenant_id", tenantID)
				c.Set("role", role)
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token 无效或已过期"})
			c.Abort()
			return
		}

		// Priority 2: Trust X-User-ID only from internal Gateway (X-Forwarded-For present)
		// This is the path for requests that came through the Gateway proxy
		if c.GetHeader("X-Forwarded-For") != "" || c.GetHeader("X-Real-IP") != "" {
			userIDStr := c.GetHeader("X-User-ID")
			if userIDStr == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
				c.Abort()
				return
			}

			userID, err := strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效用户"})
				c.Abort()
				return
			}

			user, err := h.repo.GetUserByID(userID)
			if err != nil || user == nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}

			if user.Status != "active" {
				c.JSON(http.StatusForbidden, gin.H{"error": "账号已禁用"})
				c.Abort()
				return
			}

			c.Set("user_id", user.ID)
			c.Set("username", user.Username)
			c.Set("tenant_id", user.TenantID)
			c.Set("role", user.Role)
			c.Next()
			return
		}

		// No valid auth source found
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		c.Abort()
	}
}

// RequirePermission checks if the user has a specific permission.
// Supports both action-based and path+method-based access control.
func RequirePermission(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)

		// Check action-based permission
		if !model.HasPermission(roleStr, action) {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}

		// Path + method based RBAC
		path := c.FullPath()
		method := c.Request.Method
		if !checkPathPermission(roleStr, path, method) {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足: 无权访问此资源"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkPathPermission enforces path+method based access control.
func checkPathPermission(role, path, method string) bool {
	// Admin has full access
	if role == model.RoleAdmin {
		return true
	}

	// Operator: read/write on most resources, no user/tenant management
	if role == model.RoleOperator {
		// Block user/tenant management for operators
		if contains(path, "/users") || contains(path, "/tenants") {
			return method == "GET" // read-only
		}
		return true
	}

	// Viewer: read-only on all resources
	if role == model.RoleViewer {
		return method == "GET"
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Auth Handlers ---

func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// Check account lockout
	if h.isLockedOut(req.Username) {
		h.auditLog(c, 0, req.Username, "login_blocked", "auth", "", "账号已锁定")
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": fmt.Sprintf("账号已锁定，请 %d 分钟后重试", int(h.lockoutTime.Minutes())),
		})
		return
	}

	user, err := h.repo.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		h.recordFailedLogin(req.Username)
		h.auditLog(c, 0, req.Username, "login_failed", "auth", "", "用户不存在")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// Check password hash
	if !verifyPassword(req.Password, user.PasswordHash) {
		h.recordFailedLogin(req.Username)
		h.auditLog(c, user.ID, user.Username, "login_failed", "auth", "", "密码错误")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if user.Status != "active" {
		h.auditLog(c, user.ID, user.Username, "login_failed", "auth", "", "账号已禁用")
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已禁用"})
		return
	}

	// Successful login — clear failed attempts
	h.clearLoginAttempts(req.Username)

	// Update last login
	h.repo.UpdateLastLogin(user.ID)

	// Audit log
	h.auditLog(c, user.ID, user.Username, "login", "auth", "", "登录成功")

	// Generate token: hex(payload):hex(hmac-signature)
	// payload = "user_id:tenant_id:role:timestamp"
	token := generateToken(user.ID, user.TenantID, user.Role)

	c.JSON(http.StatusOK, gin.H{
		"token":       token,
		"user_id":     user.ID,
		"username":    user.Username,
		"display_name": user.DisplayName,
		"role":        user.Role,
		"tenant_id":   user.TenantID,
	})
}

func (h *AdminHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.repo.GetUserByID(userID.(int64))
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

// --- User Management ---

func (h *AdminHandler) ListUsers(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)

	users, total, err := h.repo.ListUsers(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Role        string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// Default role
	if req.Role == "" {
		req.Role = model.RoleViewer
	}
	if req.Role != model.RoleAdmin && req.Role != model.RoleOperator && req.Role != model.RoleViewer {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效角色"})
		return
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: hashPassword(req.Password),
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Role:         req.Role,
		Status:       "active",
		TenantID:     1, // Default tenant
	}

	if err := h.repo.CreateUser(user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	username, _ := c.Get("username")
	h.auditLog(c, user.ID, username.(string), "create", "user", strconv.FormatInt(user.ID, 10), "创建用户 "+user.Username)

	c.JSON(http.StatusCreated, user)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}

	user, err := h.repo.GetUserByID(id)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Role        string `json:"role"`
		Status      string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := h.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	username, _ := c.Get("username")
	h.auditLog(c, user.ID, username.(string), "update", "user", strconv.FormatInt(user.ID, 10), "更新用户 "+user.Username)

	c.JSON(http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}

	// Cannot delete self
	currentUserID, _ := c.Get("user_id")
	if currentUserID.(int64) == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
		return
	}

	if err := h.repo.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	username, _ := c.Get("username")
	h.auditLog(c, id, username.(string), "delete", "user", strconv.FormatInt(id, 10), "删除用户")

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- Audit Logs ---

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)
	userID := parseInt64Default(c.Query("user_id"), 0)
	action := c.Query("action")

	logs, total, err := h.repo.ListAuditLogs(limit, offset, userID, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}

// --- Helpers ---

func (h *AdminHandler) auditLog(c *gin.Context, userID int64, username, action, resource, resourceID, detail string) {
	ip := c.ClientIP()
	h.repo.InsertAuditLog(&model.AuditLog{
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     detail,
		IP:         ip,
	})
}

func hashPassword(s string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), 12)
	if err != nil {
		// Fallback to SHA-256 only if bcrypt fails (should not happen)
		h := sha256.Sum256([]byte(s))
		return hex.EncodeToString(h[:])
	}
	return string(hash)
}

func verifyPassword(password, hash string) bool {
	// Try bcrypt first
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err == nil {
		return true
	}
	// Fallback: check if it's a legacy SHA-256 hash
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:]) == hash
}

// --- Tenant Management ---

func (h *AdminHandler) ListTenants(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)

	tenants, total, err := h.tenantRepo.ListTenants(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tenants, "total": total})
}

func (h *AdminHandler) CreateTenant(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Plan     string `json:"plan"`
		MaxHosts int    `json:"max_hosts"`
		MaxUsers int    `json:"max_users"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Plan == "" {
		req.Plan = "free"
	}
	// Apply plan defaults if not explicitly set
	if limits, ok := model.PlanLimits[req.Plan]; ok && req.MaxHosts == 0 {
		req.MaxHosts = limits.MaxHosts
	}
	if limits, ok := model.PlanLimits[req.Plan]; ok && req.MaxUsers == 0 {
		req.MaxUsers = limits.MaxUsers
	}

	tenant := &model.Tenant{
		Name:     req.Name,
		Code:     req.Code,
		Plan:     req.Plan,
		MaxHosts: req.MaxHosts,
		MaxUsers: req.MaxUsers,
	}

	if err := h.tenantRepo.CreateTenant(tenant); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "租户编码已存在"})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

func (h *AdminHandler) UpdateTenant(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	tenant, err := h.tenantRepo.GetTenant(id)
	if err != nil || tenant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "租户不存在"})
		return
	}

	var req struct {
		Name     string `json:"name"`
		Code     string `json:"code"`
		Status   string `json:"status"`
		Plan     string `json:"plan"`
		MaxHosts int    `json:"max_hosts"`
		MaxUsers int    `json:"max_users"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Code != "" {
		tenant.Code = req.Code
	}
	if req.Status != "" {
		tenant.Status = req.Status
	}
	if req.Plan != "" {
		tenant.Plan = req.Plan
	}
	if req.MaxHosts > 0 {
		tenant.MaxHosts = req.MaxHosts
	}
	if req.MaxUsers > 0 {
		tenant.MaxUsers = req.MaxUsers
	}

	if err := h.tenantRepo.UpdateTenant(tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tenant)
}

func (h *AdminHandler) DeleteTenant(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除默认租户"})
		return
	}
	if err := h.tenantRepo.DeleteTenant(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// jwtSecret must match the gateway's JWT_SECRET env var.
var jwtSecret = func() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		panic("FATAL: JWT_SECRET environment variable is required (minimum 32 bytes)")
	}
	if len(s) < 32 {
		panic("FATAL: JWT_SECRET must be at least 32 bytes")
	}
	return s
}()

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// generateToken creates a HMAC-SHA256 signed token for the gateway to validate.
// Format: hex(payload):hex(signature) where payload = "user_id:tenant_id:role:timestamp"
func generateToken(userID, tenantID int64, role string) string {
	payload := fmt.Sprintf("%d:%d:%s:%s", userID, tenantID, role, time.Now().UTC().Format(time.RFC3339))
	payloadHex := hex.EncodeToString([]byte(payload))

	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return payloadHex + ":" + signature
}

// validateToken parses and validates a HMAC-SHA256 token.
// Returns user_id, tenant_id, role on success.
func validateToken(token string) (int64, int64, string, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return 0, 0, "", fmt.Errorf("invalid token format")
	}

	payload, err := hex.DecodeString(parts[0])
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid token encoding")
	}
	signature := parts[1]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return 0, 0, "", fmt.Errorf("invalid token signature")
	}

	// Parse payload: user_id:tenant_id:role:timestamp
	payloadStr := string(payload)
	fields := strings.SplitN(payloadStr, ":", 4)
	if len(fields) != 4 {
		return 0, 0, "", fmt.Errorf("invalid token payload")
	}

	// Check token expiry (24 hours)
	ts, err := time.Parse(time.RFC3339, fields[3])
	if err == nil && time.Since(ts) > 24*time.Hour {
		return 0, 0, "", fmt.Errorf("token expired")
	}

	userID, _ := strconv.ParseInt(fields[0], 10, 64)
	tenantID, _ := strconv.ParseInt(fields[1], 10, 64)
	role := fields[2]

	return userID, tenantID, role, nil
}

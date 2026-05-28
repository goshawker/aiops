package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"aiops/internal/model"
	"aiops/internal/repo"
)

// AdminHandler handles user management and audit logs.
type AdminHandler struct {
	repo *repo.AdminRepo
}

func NewAdminHandler(repo *repo.AdminRepo) *AdminHandler {
	return &AdminHandler{repo: repo}
}

// --- Auth Middleware ---

// AuthMiddleware validates the session and injects user info.
func (h *AdminHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple token-based auth (Phase 1: header-based)
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr == "" {
			// Allow unauthenticated access for login endpoint
			if c.Request.URL.Path == "/api/v1/auth/login" {
				c.Next()
				return
			}
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
		c.Set("role", user.Role)
		c.Next()
	}
}

// RequirePermission checks if the user has a specific permission.
func RequirePermission(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)
		if !model.HasPermission(roleStr, action) {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
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

	user, err := h.repo.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// Check password hash
	hash := sha256Hash(req.Password)
	if hash != user.PasswordHash {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已禁用"})
		return
	}

	// Update last login
	h.repo.UpdateLastLogin(user.ID)

	// Audit log
	h.auditLog(c, user.ID, user.Username, "login", "auth", "", "登录成功")

	c.JSON(http.StatusOK, gin.H{
		"user_id":       user.ID,
		"username":      user.Username,
		"display_name":  user.DisplayName,
		"role":          user.Role,
		"tenant_id":     user.TenantID,
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
		PasswordHash: sha256Hash(req.Password),
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

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

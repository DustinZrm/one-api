package api

import (
	"encoding/json"
	"fmt"
	"io"
	"one-mcp/internal/core"
	"one-mcp/internal/model"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// JWT Secret Key (In production, this should be an env var)
var jwtSecret = []byte("one-mcp-secret-key-change-me")

type Handler struct {
	db      *gorm.DB
	gateway *core.Gateway
}

func NewHandler(db *gorm.DB, gateway *core.Gateway) *Handler {
	return &Handler{
		db:      db,
		gateway: gateway,
	}
}

// Admin APIs

func (h *Handler) Login(c *gin.Context) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	var admin model.Admin
	if err := h.db.Where("username = ?", creds.Username).First(&admin).Error; err != nil {
		c.JSON(401, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(creds.Password)); err != nil {
		c.JSON(401, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": admin.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(200, gin.H{"token": tokenString})
}

func (h *Handler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("username", claims["username"])
		}

		c.Next()
	}
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	username, _ := c.Get("username")
	
	var admin model.Admin
	if err := h.db.Where("username = ?", username).First(&admin).Error; err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(400, gin.H{"error": "Incorrect old password"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to hash password"})
		return
	}

	admin.Password = string(hashedPassword)
	h.db.Save(&admin)

	c.JSON(200, gin.H{"status": "ok", "message": "Password changed successfully"})
}

func (h *Handler) ListServers(c *gin.Context) {
	var servers []model.UpstreamServer
	h.db.Find(&servers)
	c.JSON(200, servers)
}

func (h *Handler) CreateServer(c *gin.Context) {
	var server model.UpstreamServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if server.TransportType == "stdio" {
		var args []string
		if server.Args != "" {
			if err := json.Unmarshal([]byte(server.Args), &args); err != nil {
				c.JSON(400, gin.H{"error": "Invalid args format"})
				return
			}
		}
		if err := core.ValidateCommand(server.Command, args); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}

	fmt.Printf("[Debug] Creating Server: Name=%s Type=%s URL=%s Cmd=%s\n", server.Name, server.TransportType, server.URL, server.Command)

	// Check if exists (including soft-deleted)
	var existing model.UpstreamServer
	if err := h.db.Unscoped().Where("name = ?", server.Name).First(&existing).Error; err == nil {
		if existing.DeletedAt.Valid {
			// Hard delete old record to allow re-creation
			h.db.Unscoped().Delete(&existing)
		} else {
			c.JSON(400, gin.H{"error": "Server name already exists"})
			return
		}
	}

	h.db.Create(&server)
	h.gateway.ReloadUpstreams()
	c.JSON(200, server)
}

func (h *Handler) UpdateServer(c *gin.Context) {
	id := c.Param("id")
	var server model.UpstreamServer
	if err := h.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if server.TransportType == "stdio" {
		var args []string
		if server.Args != "" {
			if err := json.Unmarshal([]byte(server.Args), &args); err != nil {
				c.JSON(400, gin.H{"error": "Invalid args format"})
				return
			}
		}
		if err := core.ValidateCommand(server.Command, args); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}

	fmt.Printf("[Debug] Updating Server %s: Name=%s Type=%s URL=%s Cmd=%s\n", id, server.Name, server.TransportType, server.URL, server.Command)

	h.db.Save(&server)
	h.gateway.ReloadUpstreams()
	c.JSON(200, server)
}

func (h *Handler) DeleteServer(c *gin.Context) {
	id := c.Param("id")
	h.db.Unscoped().Where("id = ?", id).Delete(&model.UpstreamServer{})
	h.gateway.ReloadUpstreams()
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) ListKeys(c *gin.Context) {
	var keys []model.ApiKey
	h.db.Find(&keys)
	c.JSON(200, keys)
}

func (h *Handler) CreateKey(c *gin.Context) {
	var key model.ApiKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if key.Key == "" {
		key.Key = "sk-" + uuid.New().String()
	}
	h.db.Create(&key)
	c.JSON(200, key)
}

func (h *Handler) UpdateKey(c *gin.Context) {
	id := c.Param("id")
	var key model.ApiKey
	if err := h.db.First(&key, "id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	
	// We only bind specific fields to allow partial updates
	var updateData struct {
		Description    string `json:"description"`
		AllowedServers string `json:"allowed_servers"`
		AllowedTools   string `json:"allowed_tools"`
	}
	
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	key.Description = updateData.Description
	key.AllowedServers = updateData.AllowedServers
	key.AllowedTools = updateData.AllowedTools
	
	h.db.Save(&key)
	c.JSON(200, key)
}

func (h *Handler) DeleteKey(c *gin.Context) {
	id := c.Param("id")
	h.db.Where("id = ?", id).Delete(&model.ApiKey{})
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) ListAllTools(c *gin.Context) {
	tools, err := h.gateway.GetAllTools()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, tools)
}

// MCP SSE Endpoints

type Session struct {
	MsgChan        chan []byte
	AllowedServers []string
	AllowedTools   []string
}

var sessions sync.Map // map[string]*Session

func (h *Handler) HandleSSE(c *gin.Context) {
	// Auth
	token := c.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	
	var apiKey model.ApiKey
	if err := h.db.Where("key = ?", token).First(&apiKey).Error; err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	
	// Parse permissions
	var allowedServers []string
	if apiKey.AllowedServers != "" {
		json.Unmarshal([]byte(apiKey.AllowedServers), &allowedServers)
	}

	var allowedTools []string
	if apiKey.AllowedTools != "" {
		json.Unmarshal([]byte(apiKey.AllowedTools), &allowedTools)
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	
	origin := c.Request.Header.Get("Origin")
	if origin != "" {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
	} else {
		c.Header("Access-Control-Allow-Origin", "*")
	}

	sessionID := uuid.New().String()
	msgChan := make(chan []byte, 10)
	
	session := &Session{
		MsgChan:        msgChan,
		AllowedServers: allowedServers,
		AllowedTools:   allowedTools,
	}
	sessions.Store(sessionID, session)
	
	defer func() {
		sessions.Delete(sessionID)
		close(msgChan)
	}()

	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s/mcp/messages?sessionId=%s", scheme, host, sessionID)
	
	c.SSEvent("endpoint", endpoint)
	c.Writer.Flush()

	notify := c.Writer.CloseNotify()
	for {
		select {
		case msg := <-msgChan:
			c.SSEvent("message", string(msg))
			c.Writer.Flush()
		case <-notify:
			return
		}
	}
}

func (h *Handler) HandleMessage(c *gin.Context) {
	sessionID := c.Query("sessionId")
	val, ok := sessions.Load(sessionID)
	if !ok {
		c.JSON(404, gin.H{"error": "Session not found"})
		return
	}
	session := val.(*Session)

	body, _ := io.ReadAll(c.Request.Body)
	
	resp, err := h.gateway.HandleMessage(body, session.AllowedServers, session.AllowedTools)
	
	if err != nil {
		// Log error but maybe don't return 500 if it's just JSON-RPC error
		// Ideally we should return JSON-RPC error response via SSE?
		// But for now, just return HTTP error if internal failure
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if resp != nil {
		respBytes, _ := json.Marshal(resp)
		select {
		case session.MsgChan <- respBytes:
		default:
		}
	}

	c.Status(202) // Accepted
}

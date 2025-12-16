package model

import (
	"time"
	"gorm.io/gorm"
)

type Admin struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Password string `gorm:"not null" json:"-"` // Hashed password
}

type UpstreamServer struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	Name      string `gorm:"uniqueIndex;not null" json:"name"` // Unique identifier, used as prefix
	
	// Transport Configuration
	TransportType string `gorm:"default:'sse'" json:"transport_type"` // "sse" or "stdio"
	
	// SSE Configuration
	URL       string `json:"url"`              // SSE Endpoint URL
	AuthToken string `json:"auth_token"`       // Optional auth token for upstream

	// Stdio Configuration
	Command string `json:"command"`          // Executable command
	Args    string `json:"args"`             // JSON array of arguments
	Env     string `json:"env"`              // JSON object of environment variables
	
	// HTTP/REST Configuration
	// If TransportType == "http", this JSON string contains the tool definition and mapping
	// Structure:
	// {
	//   "name": "my_tool",
	//   "description": "...",
	//   "method": "GET", // or POST
	//   "headers": {"k":"v"},
	//   "parameters": [ { "name": "q", "type": "string", "description": "...", "required": true, "default": "..." } ]
	// }
	ToolConfig string `json:"tool_config"`

	Enabled   bool   `gorm:"default:true" json:"enabled"`
}

type ApiKey struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Key         string `gorm:"uniqueIndex;not null" json:"key"`
	Description string `json:"description"`
	
	// Permissions: List of allowed UpstreamServer IDs
	// Stored as JSON string, e.g. "[1, 2, 3]"
	// If empty or "*", allows all.
	// DEPRECATED: Use AllowedTools instead for finer granularity, or keep for backward compatibility
	AllowedServers string `json:"allowed_servers"` 

	// AllowedTools: List of allowed tool names (prefixed), e.g. ["github__get_issue", "filesystem__read_file"]
	// Stored as JSON string
	// If empty, falls back to AllowedServers check.
	// If ["*"], allows all tools.
	AllowedTools string `json:"allowed_tools"`
}

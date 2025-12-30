package main

import (
	"log"
	"os"
	"path/filepath"
	"one-mcp/internal/api"
	"one-mcp/internal/core"
	"one-mcp/internal/model"

	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	// Determine data directory
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	dataDir = filepath.Clean(dataDir)

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "one-mcp.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// Auto Migrate
	db.AutoMigrate(&model.UpstreamServer{}, &model.ApiKey{}, &model.Admin{})

	// Initialize Default Admin if not exists
	var adminCount int64
	db.Model(&model.Admin{}).Count(&adminCount)
	if adminCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		db.Create(&model.Admin{
			Username: "admin",
			Password: string(hashedPassword),
		})
		log.Println("Initialized default admin user: admin / admin")
	}

	// Init Gateway
	gateway := core.NewGateway(db)
	gateway.ReloadUpstreams()

	// Init Handler
	handler := api.NewHandler(db, gateway)

	r := gin.Default()
	
	// CORS
	config := cors.DefaultConfig()
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		config.AllowOrigins = strings.Split(origins, ",")
		config.AllowAllOrigins = false
	} else {
		config.AllowAllOrigins = true
	}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Routes
	
	// Public Login API
	r.POST("/api/login", handler.Login)

	// Protected Admin APIs
	apiGroup := r.Group("/api/v1")
	apiGroup.Use(handler.AdminAuthMiddleware())
	{
		apiGroup.GET("/servers", handler.ListServers)
		apiGroup.POST("/servers", handler.CreateServer)
		apiGroup.PUT("/servers/:id", handler.UpdateServer)
		apiGroup.DELETE("/servers/:id", handler.DeleteServer)

		apiGroup.GET("/keys", handler.ListKeys)
		apiGroup.POST("/keys", handler.CreateKey)
		apiGroup.PUT("/keys/:id", handler.UpdateKey)
		apiGroup.DELETE("/keys/:id", handler.DeleteKey)
		
		apiGroup.GET("/tools", handler.ListAllTools)
		
		apiGroup.POST("/change-password", handler.ChangePassword)
	}

	mcpGroup := r.Group("/mcp")
	{
		mcpGroup.GET("/sse", handler.HandleSSE)
		mcpGroup.POST("/messages", handler.HandleMessage)
	}

	// Serve Frontend (SPA)
	// Serve static files from ../web/dist
	r.Use(static.Serve("/", static.LocalFile("../web/dist", true)))
	
	// Fallback for SPA: if not found (and not api), serve index.html
	r.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api") && !strings.HasPrefix(c.Request.URL.Path, "/mcp") {
			c.File("../web/dist/index.html")
		}
	})

	r.Run(":8080")
}

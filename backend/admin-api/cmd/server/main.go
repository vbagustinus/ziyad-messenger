package main

import (
	"log"
	"os"

	auditHandler "admin-service/internal/audit"
	"admin-service/internal/auth"
	"admin-service/internal/channels"
	"admin-service/internal/cluster"
	"admin-service/internal/db"
	"admin-service/internal/departments"
	"admin-service/internal/devices"
	"admin-service/internal/middleware"
	"admin-service/internal/monitoring"
	"admin-service/internal/roles"
	"admin-service/internal/system"
	"admin-service/internal/users"
	"admin-service/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	dbPath := os.Getenv("ADMIN_DB_PATH")
	if dbPath == "" {
		dbPath = "data/admin.db"
	}
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal(err)
	}
	if err := db.Init(dbPath); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := auth.EnsureSuperAdmin("admin", "admin"); err != nil {
		log.Printf("EnsureSuperAdmin: %v", err)
	}

	hub := websocket.NewHub()
	go hub.Run()

	r := gin.Default()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		// Allow any origin in LAN for ease of testing
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.POST("/admin/login", auth.LoginHandler)

	api := r.Group("/admin")
	api.Use(middleware.JWTAuth(), middleware.RequireAdmin())
	{
		api.GET("/me", auth.MeHandler)
		api.POST("/admins", middleware.Audit("admin.create", "admins"), auth.CreateAdminHandler)

		api.GET("/users", middleware.Audit("users.list", "users"), users.List)
		api.POST("/users", middleware.Audit("user.create", "users"), users.Create)
		api.PUT("/users/:id", middleware.Audit("user.update", "users"), users.Update)
		api.DELETE("/users/:id", middleware.Audit("user.delete", "users"), users.Delete)
		api.POST("/users/:id/reset-password", middleware.Audit("user.reset_password", "users"), users.ResetPassword)

		api.GET("/departments", middleware.Audit("departments.list", "departments"), departments.List)
		api.POST("/departments", middleware.Audit("department.create", "departments"), departments.Create)
		api.DELETE("/departments/:id", middleware.Audit("department.delete", "departments"), departments.Delete)

		api.GET("/roles", middleware.Audit("roles.list", "roles"), roles.List)
		api.POST("/roles", middleware.Audit("role.create", "roles"), roles.Create)
		api.PUT("/roles/:id", middleware.Audit("role.update", "roles"), roles.Update)
		api.DELETE("/roles/:id", middleware.Audit("role.delete", "roles"), roles.Delete)

		api.GET("/devices", middleware.Audit("devices.list", "devices"), devices.List)
		api.DELETE("/devices/:id", middleware.Audit("device.delete", "devices"), devices.Delete)

		api.GET("/channels", middleware.Audit("channels.list", "channels"), channels.List)
		api.POST("/channels", middleware.Audit("channel.create", "channels"), channels.Create)
		api.DELETE("/channels/:id", middleware.Audit("channel.delete", "channels"), channels.Delete)

		// Membership management
		api.GET("/channels/:id/members", channels.ListMembers)
		api.POST("/channels/:id/members", channels.AddMember)
		api.DELETE("/channels/:id/members/:user_id", channels.RemoveMember)

		api.GET("/monitoring/network", monitoring.Network)
		api.GET("/monitoring/users", monitoring.Users)
		api.GET("/monitoring/messages", monitoring.Messages)
		api.GET("/monitoring/files", monitoring.Files)
		api.GET("/monitoring/system", monitoring.System)

		api.GET("/audit", auditHandler.ListHandler)

		api.GET("/system/health", system.Health)
		api.GET("/cluster/status", cluster.Status)

		api.GET("/ws", websocket.HandleWS(hub))
	}

	// Public endpoints (no auth required) for client apps
	r.GET("/public/channels", channels.List)
	r.GET("/public/users", users.List)
	r.GET("/health", system.Health)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	log.Printf("Admin service listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

// Router holds all handlers
type Router struct {
	cfg         *config.Config
	authHandler *Auth
	// Add more handlers here as needed
	// roomHandler *Room
	// recordingHandler *Recording
	// reportHandler *Report
}

// NewRouter creates a new router with all handlers
func NewRouter(cfg *config.Config, authHandler *Auth) *Router {
	return &Router{
		cfg:         cfg,
		authHandler: authHandler,
	}
}

// Setup configures all application routes
func (rt *Router) Setup(e *echo.Echo) {
	// Health check endpoint
	e.GET("/health", rt.healthCheck)

	// API v1 group
	v1 := e.Group("/v1")

	// Setup route groups
	rt.setupAuthRoutes(v1)
	// rt.setupRoomRoutes(v1)
	// rt.setupRecordingRoutes(v1)
	// rt.setupReportRoutes(v1)
}

// setupAuthRoutes configures authentication routes
func (rt *Router) setupAuthRoutes(g *echo.Group) {
	authGroup := g.Group("/auth")

	if rt.authHandler != nil {
		// Use Echo handlers directly
		authGroup.GET("/google/login", rt.authHandler.GoogleLogin)
		authGroup.GET("/google/callback", rt.authHandler.GoogleCallback)
		authGroup.POST("/refresh", rt.authHandler.RefreshToken)
		authGroup.POST("/logout", rt.authHandler.Logout)
		authGroup.GET("/me", rt.authHandler.Me)
	} else {
		// Placeholder routes when handler is not initialized
		authGroup.GET("/google/login", rt.notImplemented)
		authGroup.GET("/google/callback", rt.notImplemented)
		authGroup.POST("/refresh", rt.notImplemented)
		authGroup.POST("/logout", rt.notImplemented)
		authGroup.GET("/me", rt.notImplemented)
	}
}

// notImplemented returns 501 Not Implemented response
func (rt *Router) notImplemented(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]interface{}{
		"error":   "This endpoint is not yet implemented",
		"path":    c.Request().URL.Path,
		"method":  c.Request().Method,
		"message": "Please initialize the required handler in main.go",
	})
}

// healthCheck returns health status
func (rt *Router) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"environment": "production",
	})
}

// // setupRoomRoutes configures room management routes
// func (rt *Router) setupRoomRoutes(g *echo.Group) {
// 	roomGroup := g.Group("/rooms")

// 	// TODO: Add room routes
// 	roomGroup.GET("", rt.notImplemented)
// 	roomGroup.POST("", rt.notImplemented)
// 	roomGroup.GET("/:id", rt.notImplemented)
// 	roomGroup.PUT("/:id", rt.notImplemented)
// 	roomGroup.DELETE("/:id", rt.notImplemented)
// }

// // setupRecordingRoutes configures recording routes
// func (rt *Router) setupRecordingRoutes(g *echo.Group) {
// 	recordingGroup := g.Group("/recordings")

// 	// TODO: Add recording routes
// 	recordingGroup.GET("", rt.notImplemented)
// 	recordingGroup.GET("/:id", rt.notImplemented)
// 	recordingGroup.POST("/:id/upload", rt.notImplemented)
// }

// // setupReportRoutes configures report routes
// func (rt *Router) setupReportRoutes(g *echo.Group) {
// 	reportGroup := g.Group("/reports")

// 	// TODO: Add report routes
// 	reportGroup.GET("", rt.notImplemented)
// 	reportGroup.GET("/:id", rt.notImplemented)
// 	reportGroup.POST("", rt.notImplemented)
// }

// // Health check handler
// func (rt *Router) healthCheck(c echo.Context) error {
// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"status":      "healthy",
// 		"time":        time.Now().Format(time.RFC3339),
// 		"environment": rt.cfg.Server.Environment,
// 		"version":     "1.0.0",
// 	})
// }

// // Welcome handler
// func (rt *Router) welcome(c echo.Context) error {
// 	return c.JSON(http.StatusOK, map[string]string{
// 		"message": "Welcome to Meeting Assistant API",
// 		"version": "1.0.0",
// 		"docs":    "/api/v1/docs",
// 	})
// }

// // Not implemented handler
// func (rt *Router) notImplemented(c echo.Context) error {
// 	return c.JSON(http.StatusNotImplemented, map[string]string{
// 		"message": "This endpoint is not yet implemented",
// 		"path":    c.Request().URL.Path,
// 		"method":  c.Request().Method,
// 	})
// }

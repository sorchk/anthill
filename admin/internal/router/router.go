package router

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/config"
	"anthill/admin/internal/handler"
	"anthill/admin/internal/middleware"
	"anthill/admin/internal/server"
)

type connManagerAdapter struct {
	cm *server.ConnManager
}

func (a *connManagerAdapter) GetConnection(nodeID string) (*handler.NodeConnectionInfo, bool) {
	conn, ok := a.cm.GetConnection(nodeID)
	if !ok {
		return nil, false
	}
	return &handler.NodeConnectionInfo{
		NodeID:        conn.NodeID,
		Conn:          conn.Conn,
		Protocol:      conn.Protocol,
		LastHeartbeat: conn.LastHeartbeat,
		Mode:          conn.Mode,
	}, true
}

func (a *connManagerAdapter) ConnectActiveNode(nodeID, addr string, port int, protocol string, cert *tls.Certificate, caCert *x509.Certificate) error {
	return a.cm.ConnectActiveNode(nodeID, addr, port, protocol, cert, caCert)
}

func Setup(db *gorm.DB, cfg *config.Config, bootstrapHandler *handler.BootstrapHandler, certHandler *handler.CertHandler, connMgr *server.ConnManager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authHandler := handler.NewAuthHandler(db)
	nodePluginHandler := handler.NewNodePluginHandler(db)
	serviceHandler := handler.NewServicePluginHandler(db)

	nodeHandler := handler.NewNodeHandler(db, &connManagerAdapter{cm: connMgr}, certHandler.GetCACert(), certHandler.GetCACertX509())
	pluginDir := "./data/plugins"
	pluginHandler := handler.NewPluginHandler(db, pluginDir)
	auditHandler := handler.NewAuditLogHandler(db)
	userHandler := handler.NewUserHandler(db)
	statsHandler := handler.NewStatsHandler(db)
	nodeGroupHandler := handler.NewNodeGroupHandler(db)
	deploymentHandler := handler.NewDeploymentHandler(db)
	deployHandler := handler.NewDeployHandler(db)
	tunnelHandler := handler.NewTunnelHandler(db)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/check", authHandler.Check)
			auth.POST("/init", authHandler.Init)
		}

		sessionHandler := handler.NewSessionHandler(db, []byte(cfg.JWTKey))

		protected := api.Group("")
		protected.Use(middleware.AuthRequired())
		protected.Use(middleware.AuditLogger())
		{
			protected.GET("/me", authHandler.Me)

			sessions := protected.Group("/sessions")
			{
				sessions.GET("", sessionHandler.List)
				sessions.DELETE("/:id", sessionHandler.Revoke)
				sessions.DELETE("", sessionHandler.RevokeAll)
			}

		users := protected.Group("/users")
		users.Use(middleware.AdminOnly())
		{
			users.GET("", userHandler.List)
			users.GET("/:id", userHandler.Get)
			users.POST("", userHandler.Create)
			users.PUT("/:id", userHandler.Update)
			users.DELETE("/:id", userHandler.Delete)
		users.POST("/:id/password", userHandler.ChangePassword)
			}

			stats := protected.Group("/stats")
			{
				stats.GET("", statsHandler.Dashboard)
				stats.GET("/health", statsHandler.Health)
			}

			nodeGroups := protected.Group("/node-groups")
			nodeGroups.Use(middleware.AdminOnly())
			{
				nodeGroups.GET("", nodeGroupHandler.List)
				nodeGroups.POST("", nodeGroupHandler.Create)
				nodeGroups.DELETE("/:id", nodeGroupHandler.Delete)
			}

			nodes := protected.Group("/nodes")
			{
				nodes.GET("", nodeHandler.List)
				nodes.GET("/:id", nodeHandler.Get)
				nodes.POST("", middleware.AdminOnly(), nodeHandler.Create)
				nodes.PUT("/:id", middleware.AdminOnly(), nodeHandler.Update)
				nodes.DELETE("/:id", middleware.AdminOnly(), nodeHandler.Delete)
				nodes.POST("/:id/connect", middleware.AdminOnly(), nodeHandler.Connect)
				nodes.POST("/:id/bootstrap", bootstrapHandler.Bootstrap)
				nodes.POST("/:id/cert/renew", middleware.AdminOnly(), certHandler.RenewCert)
				nodes.POST("/:id/cert/revoke", middleware.AdminOnly(), certHandler.RevokeCert)
				nodes.POST("/:id/reset-token", middleware.AdminOnly(), nodeHandler.ResetToken)

				nodes.GET("/:id/plugins", nodePluginHandler.ListForNode)
				nodes.POST("/:id/plugins/install", nodePluginHandler.Install)
				nodes.DELETE("/:id/plugins/:plugin", nodePluginHandler.Uninstall)
				nodes.POST("/:id/plugins/:plugin/enable", nodePluginHandler.Enable)
				nodes.POST("/:id/plugins/:plugin/disable", nodePluginHandler.Disable)

				nodes.POST("/:id/services/:plugin/start", serviceHandler.Start)
				nodes.POST("/:id/services/:plugin/stop", serviceHandler.Stop)
				nodes.POST("/:id/services/:plugin/restart", serviceHandler.Restart)
				nodes.POST("/:id/services/:plugin/pause", serviceHandler.Pause)
				nodes.GET("/:id/services/:plugin/status", serviceHandler.Status)
				nodes.PUT("/:id/services/:plugin/config", serviceHandler.Config)
			}

			certs := protected.Group("/certs")
			certs.Use(middleware.AdminOnly())
			{
				certs.GET("/crl", certHandler.GetCRL)
			}

			plugins := protected.Group("/plugins")
			{
				plugins.GET("", pluginHandler.List)
				plugins.POST("", middleware.AdminOnly(), pluginHandler.Upload)
				plugins.DELETE("/:id", middleware.AdminOnly(), pluginHandler.Delete)
				plugins.GET("/:id/download", pluginHandler.Download)
			}

			logs := protected.Group("/logs")
			{
				logs.GET("", func(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{}) })
			}

			audit := protected.Group("/audit")
			{
				audit.GET("", auditHandler.List)
				audit.GET("/:id", auditHandler.Get)
			}

			deployments := protected.Group("/deployments")
			{
				deployments.GET("", deploymentHandler.List)
				deployments.POST("", deploymentHandler.Create)
				deployments.GET("/:id", deploymentHandler.Get)
				deployments.POST("/:id/cancel", deploymentHandler.Cancel)
			}

			deploy := protected.Group("/deploy")
			{
				deploy.GET("", deployHandler.ListTasks)
				deploy.POST("", deployHandler.CreateTask)
				deploy.GET("/:id", deployHandler.GetTask)
			}

			tunnels := protected.Group("/tunnels")
			{
				tunnels.GET("", tunnelHandler.List)
				tunnels.GET("/:id", tunnelHandler.Get)
				tunnels.POST("", middleware.AdminOnly(), tunnelHandler.Create)
				tunnels.PUT("/:id", middleware.AdminOnly(), tunnelHandler.Update)
				tunnels.DELETE("/:id", middleware.AdminOnly(), tunnelHandler.Delete)
				tunnels.POST("/:id/toggle", middleware.AdminOnly(), tunnelHandler.Toggle)
				tunnels.GET("/:id/stats", tunnelHandler.Stats)
			}
		}
	}

	return r
}
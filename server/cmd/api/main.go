package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"kredly/internal/config"
	"kredly/internal/database" // Tambahan import database
	"kredly/internal/groq"
	"kredly/internal/handlers"
	"kredly/internal/middleware" // Tambahan import middleware
	"kredly/internal/models"     // Tambahan import models
	"kredly/internal/service"
	"kredly/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Configurations
	cfg := config.LoadConfig()

	// ==========================================
	// INISIALISASI DATABASE
	// ==========================================
	database.ConnectDB(cfg.DatabaseURL)
	if err := models.SetupIndexes(database.DB); err != nil {
		log.Fatalf("Failed to setup database indexes: %v", err)
	}
	// ==========================================

	// 2. Initialize Groq API Client
	groqClient := groq.NewClient(cfg.GroqAPIKeys, cfg.GroqBaseURL)

	// 3. Initialize CAT system
	sessionStore := store.NewSessionStore(database.DB)
	catService := service.NewCATService(sessionStore, groqClient)
	sessionHandler := handlers.NewSessionHandler(catService)

	// Start background session expiration job
	// Runs every hour; marks in-progress sessions as expired when ExpiresAt < now.
	jobCtx, jobCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer jobCancel()
	go catService.RunExpirationJob(jobCtx)

	// 4. Initialize Auth system
	authService := service.NewAuthService()
	emailService := service.NewEmailService(cfg)
	authHandler := handlers.NewAuthHandler(authService, emailService, cfg)

	// 5. Initialize Onboarding system
	onboardingHandler := handlers.NewOnboardingHandler(groqClient)

	// 6. Initialize Profile system
	profileHandler := handlers.NewProfileHandler(groqClient)

	// 7. Initialize HTTP Handlers
	cvHandler := handlers.NewCVHandler(groqClient)

	// 8. Initialize Blockchain Handler
	blockchainHandler := handlers.NewBlockchainHandler(database.DB)

	// 9. Initialize Job Handler
	jobHandler := handlers.NewJobHandler(database.DB)

	// 10. Initialize Activity Handler
	activityHandler := handlers.NewActivityHandler(database.DB)

	// 11. Initialize Token Handler
	tokenHandler := handlers.NewTokenHandler()

	// 12. Initialize Chat Handler
	chatHandler := handlers.NewChatHandler(groqClient)

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 6. Initialize Gin router
	r := gin.Default()

	// 7. Setup CORS Middleware
	r.Use(corsMiddleware())

	// 8. Define Routes
	// Detect jika deploy di Vercel, gunakan base path "/", jika tidak gunakan "/api"
	basePath := "/api"
	// if os.Getenv("VERCEL") == "1" || os.Getenv("VERCEL_ENV") != "" {
	// 	basePath = "/"
	// 	log.Println("Running on Vercel, using base path: /")
	// } else {
	// 	log.Printf("Running locally, using base path: %s", basePath)
	// }

	api := r.Group(basePath)
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":      "ok",
				"environment": cfg.Environment,
				"groq_loaded": len(cfg.GroqAPIKeys) > 0,
			})
		})

		api.POST("/parse-cv", middleware.AuthMiddleware(cfg, authService), cvHandler.HandleParseCV)
		api.POST("/profile/custom-assessment", middleware.AuthMiddleware(cfg, authService), cvHandler.HandleCreateCustomAssessment)

		// Blockchain verification and issuance
		api.POST("/blockchain/verify-by-hash", blockchainHandler.HandleVerifyByHashOnly) // Verify with hash only (search DB first)
		api.POST("/blockchain/verify-by-certificate-id", blockchainHandler.HandleVerifyByCertificateId) // Verify with certificate ID only
		api.GET("/blockchain/verify", blockchainHandler.HandleCheckCertificate)
		api.POST("/blockchain/issue", middleware.AuthMiddleware(cfg, authService), blockchainHandler.HandleIssueCertificate)
		api.GET("/certificates/user", middleware.AuthMiddleware(cfg, authService), blockchainHandler.HandleGetUserCertificates)
		api.GET("/certificates/metadata/:sessionId", blockchainHandler.HandleGetCertificateMetadata)
		api.GET("/certificates/metadata/cert/:certificateId", blockchainHandler.HandleGetCertificateMetadataByCertID)

		// CAT Session endpoints
		api.POST("/sessions", middleware.AuthMiddleware(cfg, authService), sessionHandler.HandleCreateSession)
		api.GET("/sessions/:id", sessionHandler.HandleGetSession)
		api.GET("/sessions/:id/next-item", sessionHandler.HandleNextItem)
		api.POST("/sessions/:id/answer", sessionHandler.HandleSubmitAnswer)
		api.GET("/sessions/:id/result", sessionHandler.HandleGetResult)
		api.POST("/sessions/:id/abandon", sessionHandler.HandleAbandonSession)

		// Auth endpoints (Better Auth compatible format)
		auth := api.Group("/auth")
		{
			// 1. Social Sign In (Better Auth format: /sign-in/:provider)
			auth.GET("/sign-in/google", authHandler.HandleGoogleLogin)

			// 2. OAuth Callback (Better Auth format: /callback/:provider)
			auth.GET("/callback/google", authHandler.HandleGoogleCallback)

			// 3. Get Session / Get Me (Better Auth uses /get-session) - Protected with auto-refresh
			auth.GET("/get-session", middleware.AuthMiddleware(cfg, authService), authHandler.HandleMe)

			// 4. Logout (Better Auth uses /sign-out)
			auth.POST("/sign-out", authHandler.HandleLogout)

			// 5. Minta OTP ke Email (Better Auth format: /sign-in/email-otp)
			// Frontend mengirim: { "email": "user@example.com", "type": "sign-in" }
			auth.POST("/sign-in/email-otp", authHandler.HandleSendEmailOTP)

			// 6. Verifikasi OTP dan Login (Better Auth format: /verify/email-otp)
			// Frontend mengirim: { "email": "user@example.com", "otp": "123456", "type": "sign-in" }
			auth.POST("/verify/email-otp", authHandler.HandleVerifyEmailOTP)
		}

		// Onboarding endpoints - Protected
		onboarding := api.Group("/onboarding")
		onboarding.Use(middleware.AuthMiddleware(cfg, authService))
		{
			onboarding.POST("/complete", onboardingHandler.HandleCompleteOnboarding)
		}

		// Username check (public, used by onboarding step 1)
		api.GET("/check-username", profileHandler.HandleCheckUsername)

		// Profile endpoints - Protected/Public
		api.GET("/profile", middleware.AuthMiddleware(cfg, authService), profileHandler.HandleGetProfile)
		api.GET("/profile/public/:username", profileHandler.HandleGetPublicProfileByUsername)

		// User account management - Protected
		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware(cfg, authService))
		{
			user.GET("/me/token-balance", tokenHandler.HandleGetTokenBalance)
			user.POST("/me/topup", tokenHandler.HandleSimulateTopup)
			user.PUT("/update-profile", profileHandler.HandleUpdateProfile)
			user.POST("/upload-cv", profileHandler.HandleUploadCV)
			user.POST("/upload-profile-photo", profileHandler.HandleUploadProfilePhoto)
			user.DELETE("/delete-account", profileHandler.HandleDeleteAccount)
			user.GET("/public-profile-settings", profileHandler.HandleGetPublicProfileSettings)
			user.PUT("/public-profile-settings", profileHandler.HandleUpdatePublicProfileSettings)
		}

		// Job endpoints - Protected
		jobs := api.Group("/jobs")
		jobs.Use(middleware.AuthMiddleware(cfg, authService))
		{
			jobs.POST("/fetch", jobHandler.FetchAndStoreJobs)
			jobs.GET("", jobHandler.GetUserJobs)
		}

		// Activity endpoints - Protected
		api.GET("/activities", middleware.AuthMiddleware(cfg, authService), activityHandler.GetUserActivities)

		// Chat endpoint - Public for now (add auth if needed)
		api.POST("/chat", chatHandler.HandleChat)
	}

	// 9. Start Server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server is starting on %s in %s mode...\n", addr, cfg.Environment)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// corsMiddleware handles CORS configuration for development and production
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Gunakan origin spesifik, bukan wildcard, karena menggunakan credentials
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "http://localhost:3000" // default untuk development
		}

		// Allow specific origin (bukan wildcard *)
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

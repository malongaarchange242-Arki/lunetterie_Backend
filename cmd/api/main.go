package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	authHandlers "github.com/lunetterie/backend/internal/auth/handlers"
	authMiddleware "github.com/lunetterie/backend/internal/auth/middleware"
	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	authServices "github.com/lunetterie/backend/internal/auth/services"
	inventoryHandlers "github.com/lunetterie/backend/internal/inventory/handlers"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/workflows"
)

func findFrontendDir() string {
	candidates := []string{
		os.Getenv("FRONTEND_DIR"),
		"../Frontend",
		"Frontend",
		"/Frontend",
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return "../Frontend"
}

func main() {
	// Charger les variables d'environnement
	_ = godotenv.Load()

	// Récupérer la config
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/lunetterie?sslmode=disable"
	}

	aiServiceURL := os.Getenv("AI_SERVICE_URL")
	if aiServiceURL == "" {
		aiServiceURL = "http://localhost:8000"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connexion à la base de données
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("❌ Erreur connexion BD: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ BD injoignable: %v", err)
	}

	log.Println("✅ Connecté à PostgreSQL")

	// Initialiser les repositories
	glassRepo := repositories.NewGlassRepository(db)
	locationRepo := repositories.NewLocationRepository(db)
	analysisRepo := repositories.NewAnalysisRepository(db)
	movementRepo := repositories.NewMovementRepository(db)
	transferRepo := repositories.NewTransferRepository(db)
	userRepo := authRepositories.NewUserRepository(db)
	webauthnRepo := authRepositories.NewWebAuthnRepository(db)

	// Initialiser les services
	allocationSvc := services.NewAllocationService(db)
	movementSvc := services.NewMovementService(movementRepo)
	barcodeSvc := services.NewBarcodeService()
	analysisSvc := services.NewAnalysisService(analysisRepo)
	storageGeneratorSvc := services.NewStorageGeneratorService(db)
	authSvc := authServices.NewAuthService(os.Getenv("JWT_SECRET"))
	webauthnSvc := authServices.NewWebAuthnService(webauthnRepo, userRepo)
	transferSvc := services.NewTransferService(transferRepo, glassRepo, movementRepo, allocationSvc)

	// Initialiser les workflows
	receptionWorkflow := workflows.NewReceptionWorkflow(
		allocationSvc,
		movementSvc,
		barcodeSvc,
		analysisSvc,
		glassRepo,
		locationRepo,
		analysisRepo,
		aiServiceURL,
	)

	// Initialiser les handlers
	receptionHandler := inventoryHandlers.NewReceptionHandler(receptionWorkflow)
	storageGeneratorHandler := inventoryHandlers.NewStorageGeneratorHandler(storageGeneratorSvc)
	transferHandler := inventoryHandlers.NewTransferHandler(transferSvc, glassRepo)
	authHandler := authHandlers.NewAuthHandler(userRepo, authSvc, webauthnSvc)
	webauthnHandler := authHandlers.NewWebAuthnHandler(webauthnSvc, authSvc)

	// Créer le router Gin
	router := gin.Default()

	// Forcer le domaine localhost:8080 pour WebAuthn
	router.Use(func(c *gin.Context) {
		// Rediriger 127.0.0.1 vers localhost (important pour WebAuthn)
		if c.Request.Host == "127.0.0.1:"+port {
			c.Redirect(http.StatusMovedPermanently, "http://localhost:"+port+c.Request.URL.Path)
			c.Abort()
			return
		}
		c.Next()
	})

	// CORS pour tests local
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Fichiers statiques du frontend admin
	frontendDir := findFrontendDir()
	router.StaticFile("/admin.html", filepath.Join(frontendDir, "admin.html"))
	router.StaticFile("/admin.css", filepath.Join(frontendDir, "admin.css"))
	router.StaticFile("/admin.js", filepath.Join(frontendDir, "admin.js"))
	router.StaticFile("/logo.jpeg", filepath.Join(frontendDir, "logo.jpeg"))
	router.StaticFile("/login.css", filepath.Join(frontendDir, "login.css"))
	router.StaticFile("/login.html", filepath.Join(frontendDir, "index.html"))
	router.StaticFile("/login.js", filepath.Join(frontendDir, "login.js"))
	router.StaticFile("/scan.html", filepath.Join(frontendDir, "scan.html"))
	router.StaticFile("/scan.css", filepath.Join(frontendDir, "scan.css"))
	router.StaticFile("/scan.js", filepath.Join(frontendDir, "scan.js"))

	// Servi via c.Data (pas c.File/StaticFile, qui passent par http.ServeFile) : net/http
	// redirige spécialement toute URL se terminant par "/index.html" vers "./", ce qui
	// reboucle à l'infini avec la redirection "/" -> "/index.html" ci-dessous.
	indexHTMLPath := filepath.Join(frontendDir, "index.html")
	router.GET("/index.html", func(c *gin.Context) {
		content, err := os.ReadFile(indexHTMLPath)
		if err != nil {
			c.String(http.StatusNotFound, "index.html introuvable")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})

	// Fichier de test HTML (page d'inscription biométrique)
	router.StaticFile("/test", "./test.html")

	// Redirection racine vers la page de connexion
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/index.html")
	})

	// Route health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "lunetterie-backend",
			"host":    c.Request.Host,
		})
	})

	// Routes API v1
	v1 := router.Group("/api/v1")
	{
		// Routes d'authentification
		auth := v1.Group("/auth")
		{
			// Routes WebAuthn (Windows Hello)
			webauthn := auth.Group("/webauthn")
			{
				webauthn.POST("/register-challenge", webauthnHandler.RegisterChallenge)
				webauthn.POST("/register-verify", webauthnHandler.RegisterVerify)
				webauthn.POST("/login-challenge", webauthnHandler.LoginChallenge)
				webauthn.POST("/login-verify", webauthnHandler.LoginVerify)
				webauthn.POST("/discoverable-login-challenge", webauthnHandler.DiscoverableLoginChallenge)
				webauthn.POST("/discoverable-login-verify", webauthnHandler.DiscoverableLoginVerify)
				webauthn.POST("/enroll-challenge", webauthnHandler.EnrollChallenge)
				webauthn.POST("/enroll-verify", webauthnHandler.EnrollVerify)
				webauthn.DELETE("/credentials/:userId", webauthnHandler.RemoveCredentials)
			}

			// Routes classiques (empreinte digitale hashée)
			auth.POST("/register", authHandler.RegisterUser)
			auth.POST("/register-fingerprint", authHandler.RegisterFingerprintUser)
			auth.POST("/login-fingerprint", authHandler.LoginWithFingerprint)
			auth.GET("/users", authHandler.ListUsers)
			auth.POST("/users", authHandler.CreateUser)
		}

		// Routes inventaire
		inventory := v1.Group("/inventory")
		inventory.Use(authMiddleware.RequireAuth(authSvc))
		{
			inventory.POST("/reception", receptionHandler.HandleReception)
			storage := inventory.Group("/storage")
			{
				storage.POST("/generate", storageGeneratorHandler.GenerateLocations)
				storage.POST("/find-free", storageGeneratorHandler.FindFreeLocation)
			}
			transfers := inventory.Group("/transfers")
			{
				transfers.POST("", transferHandler.CreateTransfer)
				transfers.GET("", transferHandler.ListTransfers)
				transfers.GET("/:id", transferHandler.GetTransfer)
				transfers.POST("/:id/items", transferHandler.AddItem)
				transfers.POST("/:id/dispatch", transferHandler.Dispatch)
				transfers.POST("/:id/receive", transferHandler.ReceiveItem)
			}
		}
	}

	// Démarrer le serveur
	log.Printf("🚀 Serveur démarré sur http://localhost:%s", port)
	log.Printf("📄 Page d'accueil: http://localhost:%s/index.html", port)
	log.Printf("🔐 Page login: http://localhost:%s/login.html", port)
	log.Printf("📋 Page admin: http://localhost:%s/admin.html", port)
	log.Printf("🧪 Page test biométrique: http://localhost:%s/test", port)
	log.Printf("❤️  Health check: http://localhost:%s/health", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Erreur démarrage serveur: %v", err)
	}
}

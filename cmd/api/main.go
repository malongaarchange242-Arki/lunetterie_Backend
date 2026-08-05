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
	shapeCorrectionRepo := repositories.NewShapeCorrectionRepository(db)
	movementRepo := repositories.NewMovementRepository(db)
	transferRepo := repositories.NewTransferRepository(db)
	userRepo := authRepositories.NewUserRepository(db)
	stationRepo := authRepositories.NewStationRepository(db)
	webauthnRepo := authRepositories.NewWebAuthnRepository(db)

	// Initialiser les services
	allocationSvc := services.NewAllocationService(db)
	movementSvc := services.NewMovementService(movementRepo)
	barcodeSvc := services.NewBarcodeService(db)
	analysisSvc := services.NewAnalysisService(analysisRepo)
	storageGeneratorSvc := services.NewStorageGeneratorService(db)
	authSvc := authServices.NewAuthService(os.Getenv("JWT_SECRET"))
	webauthnSvc := authServices.NewWebAuthnService(webauthnRepo, userRepo)
	transferSvc := services.NewTransferService(transferRepo, glassRepo, movementRepo, allocationSvc, stationRepo)
	deliveryRepo := repositories.NewDeliveryRepository(db)
	deliverySvc := services.NewDeliveryService(deliveryRepo, glassRepo, movementSvc)
	saleRepo := repositories.NewSaleRepository(db)
	reserveRepo := repositories.NewReserveRepository(db)
	saleSvc := services.NewSaleService(saleRepo, glassRepo, movementRepo, allocationSvc, stationRepo)
	reserveSvc := services.NewReserveService(reserveRepo, glassRepo, movementRepo, allocationSvc)
	displaySvc := services.NewDisplayService(glassRepo, movementRepo, allocationSvc, stationRepo, transferRepo)
	storageSvc := services.NewStorageService(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY"), "glasses-photos")
	aiSvc := services.NewAIService(aiServiceURL)
	similaritySvc := services.NewSimilarityService(glassRepo)

	// Initialiser les workflows
	receptionWorkflow := workflows.NewReceptionWorkflow(
		allocationSvc,
		movementSvc,
		barcodeSvc,
		analysisSvc,
		storageSvc,
		similaritySvc,
		glassRepo,
		locationRepo,
		analysisRepo,
		shapeCorrectionRepo,
	)

	// Initialiser les handlers
	receptionHandler := inventoryHandlers.NewReceptionHandler(receptionWorkflow)
	receptionCommandRepo := repositories.NewReceptionCommandRepository(db)
	receptionCommandHandler := inventoryHandlers.NewReceptionCommandHandler(receptionCommandRepo)
	supplierOrderRepo := repositories.NewSupplierOrderRepository(db)
	supplierOrderHandler := inventoryHandlers.NewSupplierOrderHandler(supplierOrderRepo)
	storageGeneratorHandler := inventoryHandlers.NewStorageGeneratorHandler(storageGeneratorSvc)
	transferHandler := inventoryHandlers.NewTransferHandler(transferSvc, glassRepo)
	// Delivery handler
	deliveryHandler := inventoryHandlers.NewDeliveryHandler(deliverySvc, glassRepo)
	saleHandler := inventoryHandlers.NewSaleHandler(saleSvc)
	reserveHandler := inventoryHandlers.NewReserveHandler(reserveSvc)
	presentoirHandler := inventoryHandlers.NewPresentoirHandler(locationRepo)
	movementHandler := inventoryHandlers.NewMovementHandler(movementRepo)
	glassHandler := inventoryHandlers.NewGlassHandler(glassRepo, displaySvc, similaritySvc)
	analyzeHandler := inventoryHandlers.NewAnalyzeHandler(aiSvc)
	chatHandler := inventoryHandlers.NewChatHandler(aiSvc)
	authHandler := authHandlers.NewAuthHandler(userRepo, stationRepo, authSvc, webauthnSvc)
	webauthnHandler := authHandlers.NewWebAuthnHandler(webauthnSvc, authSvc)

	// Créer le router Gin
	router := gin.Default()

	// Middleware de sécurité HTTP + CORS
	router.Use(func(c *gin.Context) {
		// En-têtes de sécurité
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' data: https://fonts.gstatic.com; "+
				"connect-src 'self' https://lunetterie-frontend.onrender.com; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		origin := c.GetHeader("Origin")
		allowedOrigins := []string{
			"https://lunetterie-frontend.onrender.com",
			"https://www.lunetterie-frontend.onrender.com",
			"http://localhost:3000",
			"http://localhost:8080",
		}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				break
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Fichiers statiques du frontend
	frontendDir := findFrontendDir()
	router.StaticFile("/logo.jpeg", filepath.Join(frontendDir, "logo.jpeg"))
	router.StaticFile("/login.css", filepath.Join(frontendDir, "login.css"))
	router.StaticFile("/login.html", filepath.Join(frontendDir, "index.html"))
	router.StaticFile("/login.js", filepath.Join(frontendDir, "login.js"))
	router.StaticFile("/scan.html", filepath.Join(frontendDir, "scan.html"))
	router.StaticFile("/scan.css", filepath.Join(frontendDir, "scan.css"))
	router.StaticFile("/scan.js", filepath.Join(frontendDir, "scan.js"))
	router.StaticFile("/direction.html", filepath.Join(frontendDir, "direction.html"))
	router.StaticFile("/direction.css", filepath.Join(frontendDir, "direction.css"))
	router.StaticFile("/direction.js", filepath.Join(frontendDir, "direction.js"))
	router.StaticFile("/reception.html", filepath.Join(frontendDir, "reception.html"))
	router.StaticFile("/reception.js", filepath.Join(frontendDir, "reception.js"))
	router.StaticFile("/historique.html", filepath.Join(frontendDir, "historique.html"))
	router.StaticFile("/historique.css", filepath.Join(frontendDir, "historique.css"))
	router.StaticFile("/historique.js", filepath.Join(frontendDir, "historique.js"))

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

	// Page admin protégée
	adminGroup := router.Group("/")
	adminGroup.Use(authMiddleware.RequireAuth(authSvc), authMiddleware.RequireRoles(1, 2, 8, 12))
	{
		adminGroup.GET("/admin.html", func(c *gin.Context) {
			c.File(filepath.Join(frontendDir, "admin.html"))
		})
		adminGroup.GET("/admin.css", func(c *gin.Context) {
			c.File(filepath.Join(frontendDir, "admin.css"))
		})
		adminGroup.GET("/admin.js", func(c *gin.Context) {
			c.File(filepath.Join(frontendDir, "admin.js"))
		})
	}

	// Rediriger /admin vers la page admin.
	router.GET("/admin", func(c *gin.Context) {
		if c.Query("debug") != "" {
			c.Redirect(http.StatusTemporaryRedirect, "/index.html")
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, "/admin.html")
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
			// Création de compte avec role_id/station_id choisis librement par l'appelant :
			// réservée aux rôles admin.
			auth.POST("/register-fingerprint", authMiddleware.RequireAuth(authSvc), authMiddleware.RequireRoles(1, 2, 8, 12), authHandler.RegisterFingerprintUser)
			auth.POST("/login-fingerprint", authHandler.LoginWithFingerprint)
			auth.POST("/login", authHandler.LoginWithPassword)
			auth.POST("/set-password", authHandler.SetInitialPassword)
			// Public : utilisée par la page de connexion avant authentification (étape email).
			auth.POST("/check-email", authHandler.CheckEmail)

			auth.Use(authMiddleware.RequireAuth(authSvc))
			{
				auth.GET("/me", authHandler.GetMe)
				auth.GET("/users", authMiddleware.RequireRoles(1, 2, 8, 12), authHandler.ListUsers)
				auth.GET("/stations", authMiddleware.RequireRoles(1, 2, 8, 12), authHandler.ListStations)
				auth.POST("/users", authMiddleware.RequireRoles(1, 2, 8, 12), authHandler.CreateUser)
			}
		}

		inventory := v1.Group("/inventory")
		inventory.Use(authMiddleware.RequireAuth(authSvc))
		{
			inventory.POST("/reception", receptionHandler.HandleReception)
			// Créer/lister les sessions de réception est réservé à la direction/admin ;
			// consulter un code précis et l'incrémenter reste ouvert à tout compte
			// authentifié, car c'est ce que fait le poste de scan (rôle MAGASINIER en
			// pratique) pendant la réception physique des montures.
			receptionCommands := inventory.Group("/reception-commands")
			{
				receptionCommands.POST("", authMiddleware.RequireRoles(1, 2, 8, 12), receptionCommandHandler.Create)
				receptionCommands.GET("", authMiddleware.RequireRoles(1, 2, 8, 12), receptionCommandHandler.List)
				receptionCommands.GET("/:code", receptionCommandHandler.GetByCode)
				receptionCommands.POST("/:code/increment", receptionCommandHandler.Increment)
			}

			supplierOrders := inventory.Group("/supplier-orders")
			supplierOrders.Use(authMiddleware.RequireRoles(1, 2, 8, 12))
			{
				supplierOrders.POST("", supplierOrderHandler.Create)
				supplierOrders.GET("", supplierOrderHandler.List)
				supplierOrders.DELETE("/:id", supplierOrderHandler.Delete)
			}
			inventory.POST("/analyze", analyzeHandler.HandleAnalyze)
			inventory.POST("/analyze-branche", analyzeHandler.HandleAnalyzeBranche)
			inventory.GET("/glasses", glassHandler.ListGlasses)
			inventory.GET("/glasses/:barcode", glassHandler.GetGlassByBarcode)
			inventory.GET("/glasses/:barcode/similar", glassHandler.GetSimilarGlasses)
			inventory.GET("/stock-summary", glassHandler.GetStockSummary)
			storage := inventory.Group("/storage")
			{
				storage.POST("/generate", storageGeneratorHandler.GenerateLocations)
				storage.POST("/find-free", storageGeneratorHandler.FindFreeLocation)
				storage.GET("/next-free", storageGeneratorHandler.PreviewFreeLocation)
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
			deliveries := inventory.Group("/deliveries")
			{
				// POST /api/v1/inventory/deliveries
				// body: { station_id, barcodes: [...] }
				deliveries.POST("", deliveryHandler.CreateDelivery)
			}

			sales := inventory.Group("/sales")
			{
				sales.POST("", saleHandler.CreateSale)
			}
			reserves := inventory.Group("/reserves")
			{
				reserves.POST("", reserveHandler.CreateReserve)
			}
			presentoir := inventory.Group("/presentoir")
			{
				presentoir.GET("/empty-slots", presentoirHandler.EmptySlotsToday)
			}
			inventory.GET("/movements", movementHandler.ListMovements)
		}

		// Assistant IA de direction (résumés/questions sur l'activité)
		ai := v1.Group("/ai")
		ai.Use(authMiddleware.RequireAuth(authSvc))
		{
			ai.POST("/chat", chatHandler.HandleChat)
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

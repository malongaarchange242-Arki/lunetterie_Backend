package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		"../Frontend_React/dist",
		"Frontend_React/dist",
		"../Frontend",
		"Frontend",
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

// startExpiredReservesSweeper rend au présentoir les montures mises de côté depuis trop
// longtemps. Un passage par heure suffit pour un délai qui se compte en jours ; le premier a
// lieu au démarrage, sinon un service redémarré chaque jour repousserait sans cesse la
// première échéance.
//
// Le balayage est idempotent : une monture déjà repassée EN_PRESENTOIR ne ressort plus de la
// requête. Deux instances de l'API peuvent donc tourner en parallèle — au pire elles se
// disputent le même emplacement, et la perdante réessaie à l'heure suivante.
func startExpiredReservesSweeper(reserveSvc *services.ReserveService) {
	sweep := func() {
		released, err := reserveSvc.ReleaseExpiredReserves()
		if err != nil {
			log.Printf("⚠️  Balayage des réservations expirées impossible: %v", err)
			return
		}
		if released > 0 {
			log.Printf("♻️  %d monture(s) réservée(s) depuis plus de %d jours remise(s) au présentoir", released, services.ReserveExpiryDays)
		}
	}

	go func() {
		sweep()
		for range time.Tick(time.Hour) {
			sweep()
		}
	}()
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

	if _, err := db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS city VARCHAR(100)`); err != nil {
		log.Printf("⚠️ Impossible d'ajouter la colonne users.city: %v", err)
	}

	// Initialiser les repositories
	glassRepo := repositories.NewGlassRepository(db)
	locationRepo := repositories.NewLocationRepository(db)
	analysisRepo := repositories.NewAnalysisRepository(db)
	countryRepo := repositories.NewCountryRepository(db)
	cityRepo := repositories.NewCityRepository(db)
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
	displaySvc := services.NewDisplayService(glassRepo, movementRepo, allocationSvc, stationRepo, transferRepo, userRepo)
	storageSvc := services.NewStorageService(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY"), "glasses-photos")
	aiSvc := services.NewAIService(aiServiceURL)
	similaritySvc := services.NewSimilarityService(glassRepo)

	startExpiredReservesSweeper(reserveSvc)

	// Initialiser les dépôts nécessaires
	receptionCommandRepo := repositories.NewReceptionCommandRepository(db)

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
		receptionCommandRepo,
	)

	// Initialiser les handlers
	receptionHandler := inventoryHandlers.NewReceptionHandler(receptionWorkflow)
	receptionCommandHandler := inventoryHandlers.NewReceptionCommandHandler(receptionCommandRepo)
	supplierOrderRepo := repositories.NewSupplierOrderRepository(db)
	supplierOrderHandler := inventoryHandlers.NewSupplierOrderHandler(supplierOrderRepo)
	expeditionHandler := inventoryHandlers.NewExpeditionHandler(supplierOrderRepo)
	demandBasketRepo := repositories.NewDemandBasketRepository(db)
	demandBasketHandler := inventoryHandlers.NewDemandBasketHandler(demandBasketRepo)
	sendListRepo := repositories.NewSendListRepository(db)
	// L'expédition d'une liste et la réception de son carton s'appuient toutes deux sur le
	// service de transfert : la première crée le transit, la seconde le clôt monture par monture.
	sendListDispatchSvc := services.NewSendListDispatchService(sendListRepo, glassRepo, stationRepo, transferSvc)
	sendListHandler := inventoryHandlers.NewSendListHandler(sendListRepo, sendListDispatchSvc)
	sendBoxHandler := inventoryHandlers.NewSendBoxHandler(sendListRepo, stationRepo, transferSvc)
	proformaRepo := repositories.NewProformaRepository(db)
	proformaHandler := inventoryHandlers.NewProformaHandler(proformaRepo, glassRepo, displaySvc, saleSvc)
	claimRepo := repositories.NewClaimRepository(db)
	claimHandler := inventoryHandlers.NewClaimHandler(claimRepo)
	savFollowupRepo := repositories.NewSavFollowupRepository(db)
	savFollowupHandler := inventoryHandlers.NewSavFollowupHandler(savFollowupRepo)
	countryHandler := inventoryHandlers.NewCountryHandler(countryRepo)
	cityHandler := inventoryHandlers.NewCityHandler(cityRepo)
	storageGeneratorHandler := inventoryHandlers.NewStorageGeneratorHandler(storageGeneratorSvc)
	transferHandler := inventoryHandlers.NewTransferHandler(transferSvc, glassRepo)
	// Delivery handler
	deliveryHandler := inventoryHandlers.NewDeliveryHandler(deliverySvc, glassRepo)
	saleHandler := inventoryHandlers.NewSaleHandler(saleSvc)
	reserveHandler := inventoryHandlers.NewReserveHandler(reserveSvc)
	presentoirHandler := inventoryHandlers.NewPresentoirHandler(locationRepo, displaySvc)
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
			"https://api-lunetterie.universearch.com",
			"https://www.api-lunetterie.universearch.com",
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:5501",
			"http://127.0.0.1:5500",
			"http://localhost:8443",
			"http://127.0.0.1:8080",
		}
		isAllowedOrigin := false
		for _, allowed := range allowedOrigins {
			if origin == allowed || strings.HasSuffix(origin, ".onrender.com") {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				isAllowedOrigin = true
				break
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			if isAllowedOrigin {
				c.Status(http.StatusNoContent)
			} else {
				c.AbortWithStatus(http.StatusForbidden)
			}
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
	// Additional React pages supported when using Frontend_React/dist.
	router.StaticFile("/vendeuse.html", filepath.Join(frontendDir, "vendeuse.html"))
	router.StaticFile("/magasin.html", filepath.Join(frontendDir, "magasin.html"))
	router.StaticFile("/caisse.html", filepath.Join(frontendDir, "caisse.html"))
	router.StaticFile("/responsable.html", filepath.Join(frontendDir, "responsable.html"))
	router.StaticFile("/sav.html", filepath.Join(frontendDir, "sav.html"))
	router.StaticFile("/labo.html", filepath.Join(frontendDir, "labo.html"))

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
			// Rate limiting par IP (pas global : chaque IP a son propre compteur) sur les
			// routes qui acceptent un secret devinable (mot de passe, jeton d'activation) —
			// freine le brute-force sans jamais impacter les autres utilisateurs de l'API.
			loginLimiter := authMiddleware.RateLimitByIP(1, 5)
			auth.POST("/login-fingerprint", loginLimiter, authHandler.LoginWithFingerprint)
			auth.POST("/login", loginLimiter, authHandler.LoginWithPassword)
			// Public : utilisée par la page de connexion avant authentification (étape email).
			auth.POST("/check-user", loginLimiter, authHandler.CheckUser)
			auth.POST("/set-password", loginLimiter, authHandler.SetInitialPassword)

			auth.Use(authMiddleware.RequireAuth(authSvc))
			{
				auth.POST("/logout", authHandler.Logout)
				auth.GET("/me", authHandler.GetMe)
				auth.GET("/users", authMiddleware.RequireRoles(1, 2, 8, 12), authHandler.ListUsers)
				auth.GET("/stations", authHandler.ListStations)
				auth.POST("/users", authMiddleware.RequireRoles(1, 2, 8, 12), authHandler.CreateUser)
			}
		}

		inventory := v1.Group("/inventory")
		inventory.Use(authMiddleware.RequireAuth(authSvc))
		{
			inventory.GET("/countries", countryHandler.List)
			inventory.GET("/cities", cityHandler.ListByCountry)
			inventory.GET("/expeditions", expeditionHandler.List)
			inventory.POST("/expeditions", expeditionHandler.Create)

			// Paniers de demande, un par magasin (ville). Ouvert à tout compte authentifié :
			// c'est le chatbot qui y dépose une ligne à chaque recherche de monture.
			baskets := inventory.Group("/baskets")
			{
				baskets.POST("", demandBasketHandler.Create)
				baskets.GET("", demandBasketHandler.List)
				baskets.GET("/counts", demandBasketHandler.Counts)
				baskets.POST("/sent", demandBasketHandler.MarkSent)
			}

			// Listes d'envoi. La création est réservée à la direction/admin ; la lecture et
			// l'accusé de réception restent ouverts, c'est le poste de scan (MAGASINIER)
			// qui les consulte pour préparer les colis.
			sendLists := inventory.Group("/send-lists")
			{
				sendLists.POST("", authMiddleware.RequireRoles(1, 2, 8, 12), sendListHandler.Create)
				sendLists.GET("", sendListHandler.List)
				sendLists.GET("/:id/items", sendListHandler.GetItems)
				sendLists.POST("/seen", sendListHandler.MarkSeen)
				sendLists.POST("/processed", sendListHandler.MarkProcessed)
				// Envoi effectif du colis vers le magasin de la ville de la liste : déplace les
				// montures et clôt la liste. Même ouverture que /processed, c'est le magasinier
				// qui déclenche l'envoi une fois toutes les montures vérifiées.
				sendLists.POST("/dispatch", sendListHandler.Dispatch)
			}

			// Cartons expédiés. Le poste de magasin demande à son ouverture s'il attend un
			// colis, puis scanne l'étiquette pour démarrer la session de réception. Ouvert à
			// tout compte authentifié : la station est vérifiée côté serveur, un poste ne peut
			// ouvrir que les cartons destinés à sa propre ville.
			sendBoxes := inventory.Group("/send-boxes")
			{
				sendBoxes.GET("", sendBoxHandler.List)
				sendBoxes.GET("/restock", sendBoxHandler.Restock)
				sendBoxes.GET("/pending", sendBoxHandler.Pending)
				// /open est reprenable : il rouvre un carton déjà ouvert avec l'avancement
				// du pointage, au lieu de le condamner. /receive fait entrer une monture au
				// stock, /close acte l'arrivée même incomplète.
				sendBoxes.POST("/open", sendBoxHandler.Open)
				sendBoxes.POST("/receive", sendBoxHandler.Receive)
				sendBoxes.POST("/close", sendBoxHandler.Close)
			}

			// Proformas : émises au Présentoir quand un client choisit ses montures, puis
			// arbitrées ligne par ligne à la Caisse (encaissement ou retour en rayon).
			proformas := inventory.Group("/proformas")
			{
				proformas.POST("", proformaHandler.Create)
				proformas.GET("", proformaHandler.List)
				proformas.GET("/:id", proformaHandler.Get)
				proformas.POST("/:id/settle", proformaHandler.Settle)
			}
			claims := inventory.Group("/claims")
			{
				claims.POST("", claimHandler.Create)
			}

			// Poste SAV : le suivi client. Il se greffe sur les proformas, qui portent
			// déjà le client, sa commande et son paiement.
			sav := inventory.Group("/sav")
			{
				sav.GET("/followups", savFollowupHandler.List)
				sav.PUT("/followups/:proformaId", savFollowupHandler.Save)
			}
			inventory.POST("/reception", receptionHandler.HandleReception)
			// Créer une session de réception reste réservé à la direction/admin ; la lister,
			// consulter un code précis et l'incrémenter sont ouverts au magasinier (rôle 3),
			// car c'est lui qui réceptionne physiquement les montures.
			//
			// La lecture lui a été ouverte pour que le poste de scan puisse afficher les
			// commandes en attente et reprendre une réception entamée depuis n'importe quel
			// poste : sans elle, la reprise ne tenait qu'au localStorage du navigateur et
			// changer de tablette obligeait à rescanner l'étiquette.
			receptionCommands := inventory.Group("/reception-commands")
			{
				receptionCommands.POST("", authMiddleware.RequireRoles(1, 2, 8, 12), receptionCommandHandler.Create)
				receptionCommands.GET("", authMiddleware.RequireRoles(1, 2, 3, 8, 12), receptionCommandHandler.List)
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
			inventory.POST("/glasses/:barcode/relocate", glassHandler.RelocateGlass)
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
				// POST /api/v1/inventory/presentoir/send-to-caisse
				// body: { station_id, barcodes: [...] }
				presentoir.POST("/send-to-caisse", presentoirHandler.SendToCaisse)
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

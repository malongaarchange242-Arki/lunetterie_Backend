package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/lunetterie?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("erreur connexion BD: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("BD injoignable: %v", err)
	}

	log.Println("Connecte a PostgreSQL")

	// Créer la table de tracking des migrations
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			migration VARCHAR(255) UNIQUE NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`); err != nil {
		log.Fatalf("erreur creation table schema_migrations: %v", err)
	}

	// Pré-remplir avec les migrations déjà appliquées (celles qui existaient avant le système de tracking)
	alreadyApplied := []string{
		"migrations/002_webauthn.up.sql",
		"migrations/003_webauthn_pending_credentials.up.sql",
		"migrations/004_users_gender.up.sql",
		"migrations/005_glasses_price.up.sql",
		"migrations/006_glasses_photos.up.sql",
		"migrations/038_store_geography.up.sql",
		"migrations/039_supplier_orders_gender.up.sql",
		"migrations/040_feature_settings.up.sql",
		"migrations/045_supplier_orders_frontend.up.sql",
		"migrations/047_pre_enregistrement_cases.up.sql",
		"migrations/014_seed_directeur.up.sql",
	}
	
	for _, path := range alreadyApplied {
		var exists int
		db.Get(&exists, `SELECT COUNT(*) FROM schema_migrations WHERE migration = $1`, path)
		if exists == 0 {
			// Ajouter comme déjà appliquée
			db.Exec(`INSERT INTO schema_migrations (migration, applied_at) VALUES ($1, $2)`, 
				path, time.Now())
		}
	}

	migrations := []string{
		"migrations/002_webauthn.up.sql",
		"migrations/003_webauthn_pending_credentials.up.sql",
		"migrations/004_users_gender.up.sql",
		"migrations/005_glasses_price.up.sql",
		"migrations/006_glasses_photos.up.sql",
		"migrations/038_store_geography.up.sql",
		"migrations/039_supplier_orders_gender.up.sql",
		"migrations/040_feature_settings.up.sql",
		"migrations/045_supplier_orders_frontend.up.sql",
		"migrations/047_pre_enregistrement_cases.up.sql",
		"migrations/048_pre_registration_shipment.up.sql",
		"migrations/014_seed_directeur.up.sql",
	}

	for _, path := range migrations {
		// Vérifier si la migration a déjà été appliquée
		var exists int
		err := db.Get(&exists, `
			SELECT COUNT(*) FROM schema_migrations WHERE migration = $1
		`, path)
		if err != nil {
			log.Fatalf("erreur verification migration %s: %v", path, err)
		}
		
		if exists > 0 {
			log.Printf("Migration déjà appliquée (ignorée): %s", path)
			continue
		}

		migration, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("erreur lecture migration %s: %v", path, err)
		}

		if _, err := db.Exec(string(migration)); err != nil {
			log.Fatalf("erreur execution migration %s: %v", path, err)
		}
		
		// Enregistrer la migration comme appliquée
		if _, err := db.Exec(`
			INSERT INTO schema_migrations (migration, applied_at) VALUES ($1, $2)
		`, path, time.Now()); err != nil {
			log.Fatalf("erreur enregistrement migration %s: %v", path, err)
		}
		
		log.Printf("Migration executee: %s", path)
	}

	fmt.Println("Migrations executees avec succes")
}

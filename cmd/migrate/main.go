package main

import (
	"fmt"
	"log"
	"os"

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

	migrations := []string{
		"migrations/002_webauthn.up.sql",
		"migrations/003_webauthn_pending_credentials.up.sql",
		"migrations/004_users_gender.up.sql",
		"migrations/005_glasses_price.up.sql",
		"migrations/006_glasses_photos.up.sql",
		"migrations/038_store_geography.up.sql",
		"migrations/014_seed_directeur.up.sql",
	}

	for _, path := range migrations {
		migration, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("erreur lecture migration %s: %v", path, err)
		}

		if _, err := db.Exec(string(migration)); err != nil {
			log.Fatalf("erreur execution migration %s: %v", path, err)
		}
		log.Printf("Migration executee: %s", path)
	}

	fmt.Println("Migrations WebAuthn executees avec succes")
}

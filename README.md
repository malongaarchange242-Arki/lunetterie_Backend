# 🌐 Lunetterie - Backend Go

Backend API pour le système de gestion d'inventaire de lunetterie. Orchestrate les workflows métier et communique avec le service IA Python.

## 🏗️ Architecture

```
┌──────────────────┐         ┌──────────────────────┐
│  AI Service      │  HTTP   │   Backend Go         │
│  (Python)        │◄───────►│   (Gestion métier)   │
│  FastAPI         │         │   Gin/Echo           │
│  YOLOv8, CNN     │         │   Modular Monolith   │
└──────────────────┘         └────────┬─────────────┘
                                      │
                            ┌─────────▼──────────┐
                            │   PostgreSQL       │
                            │   (Base de données)│
                            └────────────────────┘
```

## 📦 Structure

```
backend/
├── cmd/api/
│   └── main.go                 # Point d'entrée
├── internal/
│   ├── inventory/
│   │   ├── handlers/           # HTTP handlers
│   │   ├── services/           # Logique métier
│   │   ├── repositories/       # Data access
│   │   ├── models/             # Entités
│   │   └── dto/                # DTOs
│   ├── workflows/              # Orchestration
│   └── shared/                 # Utilitaires
├── migrations/                 # SQL migrations
├── go.mod
├── Dockerfile
└── Makefile
```

## 🚀 Démarrage rapide

### Avec Docker Compose (recommandé)

```bash
# Démarrer tous les services
make docker-up

# Vérifier l'état
docker ps

# Voir les logs
make logs-backend
make logs-ai
```

### Localement

**Prérequis:**
- Go 1.21+
- PostgreSQL 15+
- Python 3.11+ (pour le service IA)

```bash
# 1. Cloner et installer
cd backend
go mod download

# 2. Configurer la BD
createdb lunetterie
psql lunetterie < migrations/001_init.up.sql

# 3. Configurer l'env
cp .env.example .env

# 4. Démarrer le backend
make run

# 5. Démarrer le service IA (dans un autre terminal)
cd ai-service
python -m uvicorn app.api.main:app --reload
```

## 📚 API Endpoints

### Health Check
```bash
GET /health
```

**Réponse:**
```json
{
  "status": "ok",
  "service": "lunetterie-backend"
}
```

### Réception de monture
```bash
POST /api/v1/inventory/reception
Content-Type: multipart/form-data

image: <file>
station_id: 1
supplier_id: (optionnel)
delivery_ref: (optionnel)
notes: (optionnel)
```

**Réponse (201 Created):**
```json
{
  "success": true,
  "data": {
    "glass_id": 42,
    "barcode": "LB-20240127-A7B3C9F2",
    "status": "EN_STOCK_GENERAL",
    "location_code": "A-02-05-P13",
    "analysis": {
      "shape": "Rectangle",
      "shape_confidence": 98.5,
      "color": "Noir",
      "color_confidence": 95.2,
      "material": "Acétate",
      "material_confidence": 91.8,
      "mount_type": "Pleine monture",
      "mount_type_confidence": 97.1,
      "brand": "Ray-Ban",
      "reference": "RX7140",
      "processing_time_ms": 1250.5
    },
    "movement": {
      "id": 156,
      "action": "RECEPTION_FOURNISSEUR",
      "to_location": "A-02-05-P13",
      "timestamp": "2024-01-27T10:30:00Z"
    }
  }
}
```

## 🔧 Configuration

### Variables d'environnement

Créer `.env` à la racine du backend:

```env
# Base de données
DATABASE_URL=postgres://postgres:postgres@localhost:5432/lunetterie?sslmode=disable

# Service IA
AI_SERVICE_URL=http://localhost:8000

# Serveur
PORT=8080
GIN_MODE=debug

# Logs
LOG_LEVEL=debug
```

## 🧪 Tests

```bash
# Tests unitaires
make test

# Tests avec couverture
make test-coverage

# Linter
make lint

# Format code
make fmt
```

## 🗄️ Base de données

### Migrations

```bash
# Exécuter les migrations
make migrate

# Depuis Docker Compose
make docker-up
docker exec lunetterie-postgres psql -U postgres -d lunetterie -f /docker-entrypoint-initdb.d/001_init.up.sql
```

### Schema principal

**glasses** - Montures physiques
- `id` - Identifiant unique
- `barcode` - Code-barres unique (LB-YYYYMMDD-UUID8)
- `status` - État (EN_STOCK_GENERAL, EN_PRESENTOIR, etc.)
- `location_id` - Emplacement actuel
- `analysis_id` - Dernière analyse IA

**movements** - Historique des mouvements
- `glass_id` - Référence à la monture
- `action` - Type d'action (RECEPTION, TRANSFERT, etc.)
- `user_id` - Utilisateur qui a effectué l'action
- `created_at` - Timestamp

**glass_analysis** - Analyses IA
- `glass_id` - Référence à la monture
- `shape`, `color`, `material`, `mount_type` - Attributs détectés
- `[shape|color|material|mount_type]_confidence` - Confiances

## 🔌 Intégration avec le service IA

Le backend appelle le service IA pour l'analyse d'images:

```go
POST http://ai-service:8000/analyze
Content-Type: multipart/form-data

image: <bytes>
```

Réponse attendue:
```json
{
  "success": true,
  "data": {
    "shape": "Rectangle",
    "shape_confidence": 98.5,
    "color": "Noir",
    "color_confidence": 95.2,
    "material": "Acétate",
    "material_confidence": 91.8,
    "mount_type": "Pleine monture",
    "mount_type_confidence": 97.1,
    "brand": "Ray-Ban",
    "reference": "RX7140",
    "processing_time_ms": 1250.5
  }
}
```

## 📋 Commandes utiles

```bash
# Démarrer les services
make docker-up

# Voir les logs en direct
make logs-backend
make logs-ai
make logs-db

# Tests
make test

# Format et lint
make fmt
make lint

# Nettoyage
make docker-down
make docker-clean

# Compilation locale
make build
make run
```

## 🐳 Docker

### Build image
```bash
docker build -t lunetterie-backend:latest backend/
```

### Exécuter seul
```bash
docker run -p 8080:8080 \
  -e DATABASE_URL=postgres://... \
  -e AI_SERVICE_URL=http://ai-service:8000 \
  lunetterie-backend:latest
```

## 📚 Ressources

- [Gin Documentation](https://gin-gonic.com/)
- [sqlx Documentation](http://jmoiron.github.io/sqlx/)
- [PostgreSQL Docs](https://www.postgresql.org/docs/)
- [Go Best Practices](https://golang.org/doc/effective_go)

## 📝 Notes de développement

### Prochains workflows à implémenter

1. **Transfert entre stations** (`/api/v1/transfers`)
2. **Mise en présentoir** (`/api/v1/displays`)
3. **Vente** (`/api/v1/sales`)
4. **Laboratoire** (`/api/v1/laboratory`)
5. **Inventaire** (`/api/v1/inventory`)

### Authentification

À implémenter avec JWT middleware (voir `internal/shared/middleware` - à créer)

### Cache

À ajouter: Redis pour le cache des analyses et catalogue

## 📞 Support

Pour toute question ou problème, ouvrir une issue sur le dépôt.

---

**Dernière mise à jour:** 2024-01-27
**Version:** 0.1.0
**Statut:** Développement 🚧

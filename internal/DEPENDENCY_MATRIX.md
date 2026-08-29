# Matrice des dépendances du backend

Ce document fixe le périmètre logique des modules avant toute migration plus profonde.

## 1. Module Identity

### Rôle
Gère tout ce qui concerne l'identité, l'authentification, les rôles, les permissions et les sessions.

### Fichiers actuellement concernés
- `backend/internal/auth/handlers/auth_handler.go`
- `backend/internal/auth/handlers/webauthn_handler.go`
- `backend/internal/auth/middleware/jwt_middleware.go`
- `backend/internal/auth/repositories/user_repository.go`
- `backend/internal/auth/repositories/station_repository.go`
- `backend/internal/auth/repositories/webauthn_repository.go`
- `backend/internal/auth/services/auth_service.go`
- `backend/internal/auth/services/webauthn_service.go`
- `backend/internal/auth/models/*`
- `backend/internal/auth/dto/*`

### Règles de frontière
- Identity ne doit pas dépendre de `inventory` pour des règles métier de stock.
- Les tokens JWT et les rôles doivent être gérés uniquement par Identity.
- Les middlewares de sécurité HTTP doivent être considérés comme du module Identity.

### À migrer vers
- `backend/internal/identity/handlers/`
- `backend/internal/identity/services/`
- `backend/internal/identity/repositories/`
- `backend/internal/identity/models/`
- `backend/internal/identity/dto/`
- `backend/internal/identity/ports/`

---

## 2. Module Inventory

### Rôle
Gère le stock physique, les montures, les emplacements, les mouvements et les règles de stockage.

### Fichiers actuellement concernés
- `backend/internal/inventory/handlers/*.go`
- `backend/internal/inventory/services/*.go`
- `backend/internal/inventory/repositories/*.go`
- `backend/internal/inventory/models/*.go`
- `backend/internal/inventory/dto/*.go`
- `backend/internal/inventory/ports/*.go`
- `backend/internal/workflows/reception_workflow.go` (workflow de réception encore hybride, à réintégrer dans réception puis dépendre de Inventory)

### Sous-domaines fonctionnels
- Lunette / Glass
- Barcode
- Valise / StorageLocation (nouvelle hiérarchie)
- Carton / container physically assigned
- Emplacement / location
- Affectation / allocation
- Déplacement / movement
- Réservation / reserve
- Stock / stock summary
- Analyse IA de produit, liée à Inventory

### Règles de frontière
- Inventory est propriétaire du stock.
- Inventory décide si une monture est validée dans le stock.
- Receipt/Reception ne doit pas créer directement des emplacements ou modifier directement le stock.

### Hiérarchie cible
- VALISE
  - CARTON
    - LUNETTE
    - LUNETTE
    - LUNETTE
- `MEUBLE` n'est plus dans la structure cible.

---

## 3. Module Reception

### Rôle
Orchestre la réception fournisseur, la collecte de photo, la validation IA et la transmission du résultat vers Inventory.

### Fichiers actuellement identifiés comme réception
- `backend/internal/inventory/handlers/reception_handler.go`
- `backend/internal/inventory/handlers/reception_command_handler.go`
- `backend/internal/inventory/dto/reception.go`
- `backend/internal/workflows/reception_workflow.go`
- `backend/internal/inventory/models/reception_command.go`

### Règle métier
Reception ne possède pas le stock. Elle appelle Inventory et ne modifie pas directement les tables de stock.

### Flux cible
- Fournisseur
- Réception
- Photo monture
- AI Service
- Informations produit
- Inventory
- Lunette + Carton

### À migrer vers
- `backend/internal/reception/handlers/`
- `backend/internal/reception/services/`
- `backend/internal/reception/repositories/`
- `backend/internal/reception/models/`
- `backend/internal/reception/dto/`
- `backend/internal/reception/ports/`

---

## 4. Dépendances actuelles réelles

### Identity -> Inventory
Aucune dépendance métier directe n'est souhaitée.

### Inventory -> Identity
Le code existant a encore des références croisées, notamment dans des services de stock qui importent des repositories auth pour des données utilisateur ou station.
Exemples identifiés :
- `backend/internal/inventory/services/display_service.go`
- `backend/internal/inventory/services/send_list_dispatch_service.go`
- `backend/internal/inventory/services/sales_and_reserves_service.go`
- `backend/internal/inventory/services/transfer_service.go`

Ces dépendances doivent être réduites à un contrat de service, pas à un accès direct au repository auth.

### Reception -> Inventory
C'est la bonne dépendance cible : Reception appelle Inventory, n'a pas de propriété de stock.

### Workflow -> Inventory
Le workflow de réception est actuellement hybride et dépend directement des repos Inventory. C’est la zone principale à convertir en service Reception + ports Inventory.

---

## 5. Conventions de migration

### Objectif architecture cible
- Handler -> Service métier -> Port -> Repository -> PostgreSQL
- Pas de handler qui appelle directement le repository
- Pas de workflow de réception qui crée le stock directement sans passer par Inventory
- Pas de création automatique de carton au niveau réception

### Règle transactionnelle cible
- `ReceptionService`
  - `Inventory Port`
  - `Transaction`
    - créer la lunette
    - affecter le carton
    - créer le mouvement
- En cas d’échec : `ROLLBACK`
- Aucune donnée partielle ne doit subsister

---

## 6. Prochaine migration concrète recommandée

### Priorité 1 : convertir les dépendances de frontière
- sortir `auth` de l’ancien runtime dans le module Identity réel
- sécuriser les imports de `inventory` vers `auth`
- empêcher que `reception` manipule le stock directement

### Priorité 2 : migrer le workflow actuel vers ReceptionService
- `backend/internal/workflows/reception_workflow.go` doit devenir un orchestrateur interne ou être remplacé par un service Reception
- conserver les API actuelles tant que les nouveaux ports ne sont pas branchés

### Priorité 3 : tenir les modules isolés dans le monolithe
- `identity`
- `inventory`
- `reception`
- `ai` hors backend Go

---

## 7. Conclusion

Le backend est bien structuré en trois familles fonctionnelles réelles :
- Identity = identité / auth
- Inventory = stock physique
- Reception = entrée fournisseur / flux d'arrivée

La vraie refonte n'est pas de créer plus de dossiers, mais de déplacer les responsabilités existantes vers ces frontières, puis de supprimer les accès directs aux repositories dans les handlers/workflows.

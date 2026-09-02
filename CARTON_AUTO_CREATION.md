# 🚀 Correctif : Création automatique de cartons STOCK

## 📋 Résumé du problème

**Avant** : L'enregistrement des montures échouait si tous les cartons étaient pleins.
```
Reception → AllocationService.FindFreeLocation()
  └─ Query : "SELECT carton WHERE station_id=X AND capacity pas pleine"
  └─ 0 résultats → ❌ Erreur "aucun emplacement libre trouvé"
```

**Impact** : Une fois tous les cartons remplis, impossible d'enregistrer une nouvelle monture.

---

## ✅ Correction appliquée

### 1. Modifications `allocation_service.go`

#### Nouvelle méthode : `findOrCreateLocation()`
```go
// Cherche un emplacement libre, ou en crée un automatiquement en STOCK
if location, err := s.FindFreeLocation(...); err == nil {
    return location, nil
}
if zone == models.ZoneStock {
    return s.createStockCarton(stationID)  // ← CRÉE AUTO
}
```

#### Nouvelle méthode : `createStockCarton()`
```
1. Résoudre la station de stockage (Stock Général)
2. Chercher/créer une VALISE
3. Générer code carton (CTN-XXXXX)
4. INSERT dans storage_locations
   - type='CARTON'
   - capacity=50
   - parent_location_id=valise.id
5. Retourner le carton créé
```

#### Nouvelle méthode : `findOrCreateValise()`
```
1. Chercher une VALISE libre existante
2. Si trouvée : la retourner
3. Sinon :
   - Générer code valise (VAL-XXXXX)
   - INSERT dans storage_locations
     - type='VALISE'
     - parent_location_id=NULL
4. Retourner la valise
```

### 2. Mise à jour `FindOrCreateStockLocation()`
```go
// AVANT
return s.FindFreeLocation(stationID, models.ZoneStock)

// APRÈS
return s.findOrCreateLocation(stationID, models.ZoneStock)
```

Désormais elle crée automatiquement si besoin.

---

## 🔄 Flux de réception avec auto-création

```
POST /api/v1/inventory/reception
  ↓
ReceptionWorkflow.Execute()
  ├─ Upload photos
  ├─ Générer barcode
  ├─ Chercher emplacement
  │   ├─ FindFreeLocation() : OK → réutiliser
  │   └─ si KO : FindOrCreateStockLocation()
  │       └─ findOrCreateLocation()
  │           └─ createStockCarton()
  │               ├─ findOrCreateValise()
  │               │   ├─ Chercher VAL existante
  │               │   └─ Créer VAL-XXXXX si besoin
  │               └─ Créer CTN-XXXXX sous VAL
  ├─ Créer Glass
  └─ Créer Movement
```

---

## 📊 Hiérarchie VALISE → CARTON

### Avant (manuel)
```
Administrateur crée manuellement les valises/cartons
↓
Risque: Oublie OU crée pas assez
↓
Réception bloquée
```

### Après (auto)
```
VAL-00001 (station_id=X, zone=STOCK, type=VALISE, parent_id=NULL)
  ├─ CTN-00001 (parent_id=VAL-00001, capacity=50)
  │   ├─ Glass #1-50
  ├─ CTN-00002 (parent_id=VAL-00001, capacity=50)
  │   ├─ Glass #51-100
  └─ CTN-00003 (créé automatiquement à la demande)
      └─ Glass #101-...
```

---

## 🔐 Séquences utilisées

| Séquence | Format | Exemple |
|----------|--------|---------|
| `valise_code_seq` | VAL-XXXXX | VAL-00001, VAL-00002 |
| `carton_code_seq` | CTN-XXXXX | CTN-00001, CTN-00002 |

- Créées dans migration `049_container_barcode_sequences.up.sql`
- Incrémentées par `nextval()` au besoin

---

## 🧪 Tests

### Test 1 : Création automatique de VALISE + CARTON
```bash
cd backend
go test ./internal/inventory/services \
  -run TestCreateStockCartonAutomatically -v
```

**Vérifications** :
- ✅ CARTON créé (type=CARTON, capacity=50)
- ✅ VALISE créée (type=VALISE, parent_id=NULL)
- ✅ CARTON pointe vers VALISE (parent_location_id=valise.id)
- ✅ 2e appel réutilise le carton

### Test 2 : Remplissage du carton → création nouveau
```bash
go test ./internal/inventory/services \
  -run TestFillCartonThenCreateNew -v
```

**Vérifications** :
- ✅ Carton rempli avec 50 montures
- ✅ Nouvelle allocation crée nouveau carton
- ✅ Deux cartons sous la même VALISE

---

## 📝 Logs générés

```
📦 Pas de carton libre en STOCK, création automatique pour station 1...
📦 Résolu station stockage: 2 pour station demandée: 1
♻️  Valise existante trouvée: VAL-00001 (id=42)
📦 Création carton: CTN-00123 (capacity=50)
✅ Carton créé automatiquement: CTN-00123 (id=1052, parent_valise=42)
```

---

## 🚨 Points importants

### Capacité CARTON
- **Valeur par défaut** : 50 montures par carton
- Modifiable dans `createStockCarton()` ligne `capacity: 50`
- Peut être NULL (illimitée) si besoin

### Résolution station
- Réception peut venir de n'importe quelle station (magasin, caisse, etc.)
- `resolveStorageStationID()` trouve automatiquement le "Stock Général"
- Les cartons sont toujours centralisés au Stock Général

### Transactions
- Pas de transaction explicite dans `createStockCarton()`
- Les INSERTs utilisent `ON CONFLICT` pour gérer les doublons en concurrence
- La réception échoue globalement si création carton échoue

### Migration
- Migration `055_validate_auto_carton_schema.up.sql` valide le schéma
- Aucun changement de table, juste des vérifications

---

## 🔄 Compatibilité

- ✅ Réception fonctionne sans changement de code client
- ✅ `FindFreeLocation()` : inchangée, reste disponible
- ✅ `FindOrCreateStockLocation()` : signature inchangée
- ✅ Logs ajoutés pour debugging

---

## 🎯 Validation post-déploiement

1. Vider tous les cartons STOCK
2. Tenter une réception
3. Vérifier que :
   - ✅ Nouvelle VALISE créée
   - ✅ Nouveau CARTON créé
   - ✅ Monture enregistrée dans le carton
4. Vérifier DB :
   ```sql
   SELECT id, code, type, parent_location_id, capacity
   FROM storage_locations
   WHERE zone = 'STOCK'
   ORDER BY created_at DESC LIMIT 5;
   ```
   Doit montrer : VAL-XXXXX → CTN-XXXXX

---

## 📚 Références

- `allocation_service.go` : Nouvelles méthodes
- `allocation_service_test.go` : Tests
- `055_validate_auto_carton_schema.up.sql` : Validation schéma
- `/memories/repo/auto-carton-creation.md` : Notes de développement

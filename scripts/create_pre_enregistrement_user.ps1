# ============================================
# Script PowerShell de création d'utilisateur PRE_ENREGISTREMENT
# ============================================
# Ce script crée un utilisateur avec le rôle pré-enregistrement via l'API backend.
# L'utilisateur devra ensuite définir son propre mot de passe via /auth/set-password
# Prérequis : le backend doit être en cours d'exécution sur localhost:8080
#            et vous devez avoir un compte admin pour vous authentifier.

# Configuration
$API_URL = "http://localhost:8080/api/v1"
$ADMIN_NAME = "Directeur"  # Nom d'un administrateur existant
$ADMIN_PASSWORD = "test"   # Mot de passe de cet admin

# Paramètres du nouvel utilisateur PRE_ENREGISTREMENT
$NEW_USER = @{
    first_name = "Préparateur"
    last_name = "Arrivages"
    email = "preparateur@lunetterie.local"
    phone = "+242 123 45 67"
    gender = "Homme"
    city = "Pointe-Noire"
    role_id = 11  # PRE_ENREGISTREMENT
    station_id = 2  # Adapter selon vos stations
}

Write-Host "🔐 Étape 1 : Authentification admin..." -ForegroundColor Cyan

# Étape 1 : Se connecter en tant qu'admin
$loginResponse = Invoke-RestMethod -Uri "$API_URL/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body (ConvertTo-Json @{
        name = $ADMIN_NAME
        password = $ADMIN_PASSWORD
        roleIndex = 3  # Admin
    })

if (-not $loginResponse.token) {
    Write-Host "❌ Erreur : impossible de se connecter en tant qu'admin." -ForegroundColor Red
    Write-Host "Réponse : $($loginResponse | ConvertTo-Json)" -ForegroundColor Red
    exit 1
}

$adminToken = $loginResponse.token
Write-Host "✅ Connexion réussie. Token obtenu." -ForegroundColor Green

Write-Host ""
Write-Host "👤 Étape 2 : Création de l'utilisateur PRE_ENREGISTREMENT (SANS mot de passe)..." -ForegroundColor Cyan

# Étape 2 : Créer le nouvel utilisateur SANS mot de passe
$headers = @{
    "Authorization" = "Bearer $adminToken"
    "Content-Type" = "application/json"
}

$createUserBody = @{
    first_name = $NEW_USER.first_name
    last_name = $NEW_USER.last_name
    email = $NEW_USER.email
    phone = $NEW_USER.phone
    gender = $NEW_USER.gender
    city = $NEW_USER.city
    role_id = $NEW_USER.role_id
    station_id = $NEW_USER.station_id
    password = ""  # Pas de mot de passe - l'utilisateur le définira lui-même
} | ConvertTo-Json

try {
    $createResponse = Invoke-RestMethod -Uri "$API_URL/auth/users" `
        -Method Post `
        -Headers $headers `
        -Body $createUserBody

    if ($createResponse.user) {
        $user = $createResponse.user
        Write-Host "✅ Utilisateur créé avec succès !" -ForegroundColor Green
        Write-Host ""
        Write-Host "Détails :" -ForegroundColor Yellow
        Write-Host "  ID         : $($user.id)"
        Write-Host "  Nom        : $($user.first_name) $($user.last_name)"
        Write-Host "  Email      : $($user.email)"
        Write-Host "  Rôle       : $($user.role_name) (ID: $($user.role_id))"
        Write-Host "  Station    : $($user.station_name)"
        Write-Host "  Actif      : $($user.is_active)"
        Write-Host ""
        Write-Host "🔑 Étape 3 : L'utilisateur doit définir son mot de passe" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Demandez à $($NEW_USER.first_name) $($NEW_USER.last_name) de définir son mot de passe :" -ForegroundColor Yellow
        Write-Host "  Utilisez cette commande curl :" -ForegroundColor Gray
        Write-Host "curl -X POST http://localhost:8080/api/v1/auth/set-password \" -ForegroundColor Gray
        Write-Host "  -H 'Content-Type: application/json' \" -ForegroundColor Gray
        Write-Host "  -d '{" -ForegroundColor Gray
        Write-Host "    ""email"": ""$($NEW_USER.email)""," -ForegroundColor Gray
        Write-Host "    ""password"": ""VOTRE_MOT_DE_PASSE_SECURISE""" -ForegroundColor Gray
        Write-Host "  }'" -ForegroundColor Gray
    } else {
        Write-Host "❌ Erreur lors de la création de l'utilisateur." -ForegroundColor Red
        Write-Host "Réponse : $($createResponse | ConvertTo-Json)" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "❌ Erreur HTTP : $_" -ForegroundColor Red
    Write-Host "Corps de réponse : $($_.Exception.Response.Content)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✨ Opération terminée." -ForegroundColor Green

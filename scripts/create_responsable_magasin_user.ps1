# =============================================================
# Création d'un utilisateur Responsable magasin
# =============================================================
# Ce script crée un compte avec le rôle RESPONSABLE_STATION et le rattache
# automatiquement à la station locale de la ville souhaitée.
#
# Prérequis :
# - le backend tourne sur http://localhost:8080
# - un compte admin existe (Directeur / test par défaut)
#
# Ajustez les valeurs ci-dessous si nécessaire.

$API_URL = "http://localhost:8080/api/v1"
$ADMIN_NAME = "Directeur"
$ADMIN_PASSWORD = "test"

$NEW_USER = @{
    first_name = "Responsable"
    last_name  = "Magasin"
    email      = "responsable.magasin@lunetterie.local"
    phone      = "+242 000 00 00"
    gender     = "Homme"
    city       = "Pointe-Noire"
    role_id    = 6   # RESPONSABLE_STATION
    password   = "test123"
}

Write-Host "🔐 Connexion admin..." -ForegroundColor Cyan
$loginResponse = Invoke-RestMethod -Uri "$API_URL/auth/login" -Method Post -ContentType "application/json" -Body (ConvertTo-Json @{
    name = $ADMIN_NAME
    password = $ADMIN_PASSWORD
    roleIndex = 2
})

if (-not $loginResponse.token) {
    throw "Impossible de se connecter en tant qu'admin. Vérifie le backend et les identifiants."
}

$adminToken = $loginResponse.token
$headers = @{
    "Authorization" = "Bearer $adminToken"
    "Content-Type" = "application/json"
}

Write-Host "🏬 Recherche de la station locale..." -ForegroundColor Cyan
$stationsResponse = Invoke-RestMethod -Uri "$API_URL/auth/stations" -Headers $headers -Method Get
$station = $stationsResponse.stations | Where-Object { $_.name -ieq "Station $($NEW_USER.city)" } | Select-Object -First 1

if (-not $station) {
    throw "Aucune station nommée 'Station $($NEW_USER.city)' n'a été trouvée. Crée la station avant de créer le compte."
}

$NEW_USER.station_id = [int64]$station.id

Write-Host "👤 Création du compte Responsable magasin..." -ForegroundColor Cyan
$createUserBody = @{
    first_name = $NEW_USER.first_name
    last_name  = $NEW_USER.last_name
    email      = $NEW_USER.email
    phone      = $NEW_USER.phone
    gender     = $NEW_USER.gender
    city       = $NEW_USER.city
    role_id    = $NEW_USER.role_id
    station_id = $NEW_USER.station_id
    password   = $NEW_USER.password
} | ConvertTo-Json

$createResponse = Invoke-RestMethod -Uri "$API_URL/auth/users" -Method Post -Headers $headers -Body $createUserBody

Write-Host "✅ Utilisateur créé avec succès !" -ForegroundColor Green
Write-Host "  Nom         : $($createResponse.user.first_name) $($createResponse.user.last_name)"
Write-Host "  Email       : $($createResponse.user.email)"
Write-Host "  Rôle        : $($createResponse.user.role_name) (ID: $($createResponse.user.role_id))"
Write-Host "  Station     : $($createResponse.user.station_name)"
Write-Host "  Login       : $($NEW_USER.email)"
Write-Host "  Mot de passe: $($NEW_USER.password)"
Write-Host ""
Write-Host "👉 Il peut maintenant se connecter sur le poste Responsable magasin." -ForegroundColor Yellow

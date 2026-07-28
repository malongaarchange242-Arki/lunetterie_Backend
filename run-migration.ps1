# Script to run WebAuthn migration
$dbHost = "localhost"
$dbPort = 5432
$dbName = "lunetterie"
$dbUser = "postgres"
$dbPassword = "postgres"

# Run the migration
Write-Host "🔄 Exécution de la migration WebAuthn..."

# Using psql
$env:PGPASSWORD = $dbPassword
psql -h $dbHost -p $dbPort -U $dbUser -d $dbName -f "migrations/002_webauthn.up.sql"

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Migration exécutée avec succès"
} else {
    Write-Host "❌ Erreur lors de la migration"
}

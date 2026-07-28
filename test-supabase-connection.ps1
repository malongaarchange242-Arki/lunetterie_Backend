# Script pour tester la connexion Supabase
# Windows PowerShell

$env:PGPASSWORD = "3C1JtEZ1xdeT9QNl"

Write-Host "🔍 Test de connexion à Supabase PostgreSQL..." -ForegroundColor Green
Write-Host ""

# Tester la connexion
psql -h db.okhofmjabstbnpwuqunt.supabase.co `
     -p 5432 `
     -U postgres `
     -d postgres `
     -c "SELECT version();"

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ Connexion à Supabase réussie!" -ForegroundColor Green
    Write-Host ""
    
    # Afficher les tables existantes
    Write-Host "📊 Tables existantes:" -ForegroundColor Cyan
    psql -h db.okhofmjabstbnpwuqunt.supabase.co `
         -p 5432 `
         -U postgres `
         -d postgres `
         -c "\dt public.*"
} else {
    Write-Host "❌ Impossible de se connecter à Supabase" -ForegroundColor Red
    Write-Host "Vérifiez votre connexion Internet et les identifiants" -ForegroundColor Yellow
}

# Nettoyer la variable temporaire
Remove-Item Env:\PGPASSWORD

# Script pour exécuter les migrations Supabase
# Windows PowerShell

$env:PGPASSWORD = "3C1JtEZ1xdeT9QNl"

Write-Host "🔄 Exécution des migrations sur Supabase..." -ForegroundColor Green

# Exécuter le fichier de migration
psql -h db.okhofmjabstbnpwuqunt.supabase.co `
     -p 5432 `
     -U postgres `
     -d postgres `
     -f "d:\Pojet la lunetterie\backend\migrations\001_init.up.sql"

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Migrations exécutées avec succès!" -ForegroundColor Green
} else {
    Write-Host "❌ Erreur lors de l'exécution des migrations" -ForegroundColor Red
}

# Nettoyer la variable temporaire
Remove-Item Env:\PGPASSWORD

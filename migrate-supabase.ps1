# Script pour exécuter les migrations Supabase
# Windows PowerShell

$env:PGPASSWORD = "3C1JtEZ1xdeT9QNl"

Write-Host "🔄 Exécution des migrations sur Supabase..." -ForegroundColor Green

$migrationDirectory = Join-Path $PSScriptRoot "migrations"
$migrationFiles = Get-ChildItem -Path $migrationDirectory -Filter "*.up.sql" | Sort-Object Name

foreach ($migrationFile in $migrationFiles) {
    Write-Host "▶ $($migrationFile.Name)" -ForegroundColor Cyan
    psql -h db.okhofmjabstbnpwuqunt.supabase.co `
         -p 5432 `
         -U postgres `
         -d postgres `
         -v ON_ERROR_STOP=1 `
         -f $migrationFile.FullName

    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Échec de la migration $($migrationFile.Name)" -ForegroundColor Red
        Remove-Item Env:\PGPASSWORD
        exit $LASTEXITCODE
    }
}

Write-Host "✅ Toutes les migrations ont été exécutées avec succès!" -ForegroundColor Green

# Nettoyer la variable temporaire
Remove-Item Env:\PGPASSWORD

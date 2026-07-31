#!/usr/bin/env bash
# ============================================================
# Vide entièrement le bucket Supabase Storage "glasses-photos".
#
# ⚠️ IRRÉVERSIBLE. À lancer après (ou avant, peu importe l'ordre)
# reset_inventory_data.sql, puisque les deux repartent de zéro ensemble.
#
# Usage :
#   SUPABASE_URL=https://xxx.supabase.co \
#   SUPABASE_SERVICE_ROLE_KEY=eyJ... \
#   bash backend/scripts/empty_storage_bucket.sh
# ============================================================
set -euo pipefail

BUCKET="glasses-photos"
: "${SUPABASE_URL:?Variable SUPABASE_URL manquante}"
: "${SUPABASE_SERVICE_ROLE_KEY:?Variable SUPABASE_SERVICE_ROLE_KEY manquante}"

# Liste récursivement tous les fichiers d'un "dossier" (chaque barcode est un
# sous-dossier : glasses-photos/LUN-CNG-00000007/monture.jpg, .../branche.jpg).
list_paths() {
    local prefix="$1"
    curl -s -X POST "${SUPABASE_URL}/storage/v1/object/list/${BUCKET}" \
        -H "Authorization: Bearer ${SUPABASE_SERVICE_ROLE_KEY}" \
        -H "Content-Type: application/json" \
        -d "{\"prefix\":\"${prefix}\",\"limit\":1000}" \
    | jq -r '.[] | select(.id != null) | .name' \
    | while read -r name; do
        echo "${prefix}${name}"
    done

    curl -s -X POST "${SUPABASE_URL}/storage/v1/object/list/${BUCKET}" \
        -H "Authorization: Bearer ${SUPABASE_SERVICE_ROLE_KEY}" \
        -H "Content-Type: application/json" \
        -d "{\"prefix\":\"${prefix}\",\"limit\":1000}" \
    | jq -r '.[] | select(.id == null) | .name' \
    | while read -r folder; do
        list_paths "${prefix}${folder}/"
    done
}

echo "🔍 Listage des fichiers dans ${BUCKET}..."
mapfile -t ALL_PATHS < <(list_paths "")

if [ "${#ALL_PATHS[@]}" -eq 0 ]; then
    echo "✅ Bucket déjà vide."
    exit 0
fi

echo "🗑️  ${#ALL_PATHS[@]} fichier(s) à supprimer."

# L'API accepte un tableau de chemins en une seule requête DELETE.
PAYLOAD=$(printf '%s\n' "${ALL_PATHS[@]}" | jq -R . | jq -s '{prefixes: .}')
curl -s -X DELETE "${SUPABASE_URL}/storage/v1/object/${BUCKET}" \
    -H "Authorization: Bearer ${SUPABASE_SERVICE_ROLE_KEY}" \
    -H "Content-Type: application/json" \
    -d "${PAYLOAD}" \
    -o /dev/null -w "Statut HTTP: %{http_code}\n"

echo "✅ Terminé."

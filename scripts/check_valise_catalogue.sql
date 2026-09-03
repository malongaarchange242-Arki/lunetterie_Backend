-- Vérification du catalogue d'une valise.
-- Exemple : VAL-001 doit contenir 95 cartons et 1 839 montures.

WITH cible AS (
    SELECT id, code, gamme, genre, montures
    FROM pre_registration_cases
    WHERE code = 'VAL-001'
), cartons AS (
    SELECT
        c.code AS valise_code,
        c.gamme AS valise_gamme,
        c.genre AS valise_genre,
        b.id AS carton_id,
        b.code AS carton_code,
        b.quantity AS quantite_declaree,
        b.gamme AS carton_gamme,
        b.type_lunette,
        b.prix,
        b.marques,
        b.couleurs,
        b.matieres,
        COUNT(DISTINCT g.id) AS montures_physiques
    FROM cible c
    JOIN pre_registration_boxes b ON b.case_id = c.id
    LEFT JOIN storage_locations sl
      ON sl.type = 'CARTON'
     AND (
            sl.code = c.code || '-' || b.code
            OR (
                sl.code = b.code
                AND EXISTS (
                    SELECT 1
                    FROM storage_locations parent_sl
                    WHERE parent_sl.id = sl.parent_location_id
                      AND parent_sl.code = c.code
                )
            )
         )
    LEFT JOIN glasses g ON g.location_id = sl.id
    GROUP BY c.code, c.gamme, c.genre, c.montures,
             b.id, b.code, b.quantity, b.gamme, b.type_lunette,
             b.prix, b.marques, b.couleurs, b.matieres
), resume AS (
    SELECT
        valise_code,
        MAX(valise_gamme) AS gamme,
        MAX(valise_genre) AS genre,
        MAX((SELECT montures FROM cible)) AS montures_declarees_valise,
        COUNT(*) AS nombre_cartons,
        SUM(quantite_declaree) AS montures_declarees_cartons,
        SUM(montures_physiques) AS montures_physiques,
        SUM(quantite_declaree) - SUM(montures_physiques) AS ecart_declare_physique
    FROM cartons
    GROUP BY valise_code
)
SELECT
    valise_code,
    gamme,
    genre,
    montures_declarees_valise,
    nombre_cartons,
    montures_declarees_cartons,
    montures_physiques,
    ecart_declare_physique,
    CASE
        WHEN nombre_cartons = 95
         AND montures_declarees_cartons = 1839
         AND montures_physiques = 1839
        THEN 'OK'
        ELSE 'A_VERIFIER'
    END AS controle
FROM resume;

-- Détail des cartons : repère immédiatement un carton manquant ou incomplet.
WITH cible AS (
    SELECT id, code
    FROM pre_registration_cases
    WHERE code = 'VAL-001'
)
SELECT
    c.code AS valise_code,
    b.code AS carton_code,
    b.quantity AS quantite_declaree,
    COUNT(DISTINCT g.id) AS montures_physiques,
    b.quantity - COUNT(DISTINCT g.id) AS ecart,
    b.gamme,
    b.type_lunette,
    b.prix,
    b.marques,
    b.couleurs,
    b.matieres
FROM cible c
JOIN pre_registration_boxes b ON b.case_id = c.id
LEFT JOIN storage_locations sl
  ON sl.type = 'CARTON'
 AND (
        sl.code = c.code || '-' || b.code
        OR (
            sl.code = b.code
            AND EXISTS (
                SELECT 1
                FROM storage_locations parent_sl
                WHERE parent_sl.id = sl.parent_location_id
                  AND parent_sl.code = c.code
            )
        )
     )
LEFT JOIN glasses g ON g.location_id = sl.id
GROUP BY c.code, b.id, b.code, b.quantity, b.gamme, b.type_lunette,
         b.prix, b.marques, b.couleurs, b.matieres
ORDER BY b.created_at, b.code;

-- Attributs de la valise tels qu'ils sont déclarés dans ses cartons.
WITH cible AS (
    SELECT id, code, gamme, genre, montures
    FROM pre_registration_cases
    WHERE code = 'VAL-001'
)
SELECT
    c.code AS valise_code,
    c.gamme,
    c.genre,
    c.montures AS montures_declarees_valise,
    ARRAY(
        SELECT DISTINCT value
        FROM pre_registration_boxes b
        CROSS JOIN LATERAL unnest(b.marques) AS value
        WHERE b.case_id = c.id
        ORDER BY value
    ) AS marques,
    ARRAY(
        SELECT DISTINCT value
        FROM pre_registration_boxes b
        CROSS JOIN LATERAL unnest(b.couleurs) AS value
        WHERE b.case_id = c.id
        ORDER BY value
    ) AS couleurs,
    ARRAY(
        SELECT DISTINCT value
        FROM pre_registration_boxes b
        CROSS JOIN LATERAL unnest(b.matieres) AS value
        WHERE b.case_id = c.id
        ORDER BY value
    ) AS matieres,
    ARRAY(
        SELECT DISTINCT b.type_lunette
        FROM pre_registration_boxes b
        WHERE b.case_id = c.id
        ORDER BY b.type_lunette
    ) AS types_lunette,
    ARRAY(
        SELECT DISTINCT b.gamme
        FROM pre_registration_boxes b
        WHERE b.case_id = c.id
        ORDER BY b.gamme
    ) AS gammes_cartons
FROM cible c;

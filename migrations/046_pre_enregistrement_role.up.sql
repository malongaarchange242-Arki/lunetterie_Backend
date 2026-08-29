-- ============================================
-- POSTE PRÉ-ENREGISTREMENT
-- ============================================
-- Le poste de pré-enregistrement est distinct du stock général : il prépare les
-- arrivages fournisseurs avant leur validation et leur mise en stock.

INSERT INTO roles (id, name, description)
SELECT 11, 'PRE_ENREGISTREMENT', 'Pré-enregistrement - validation des arrivages fournisseurs avant stockage'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'PRE_ENREGISTREMENT')
  AND NOT EXISTS (SELECT 1 FROM roles WHERE id = 11);

INSERT INTO roles (name, description)
SELECT 'PRE_ENREGISTREMENT', 'Pré-enregistrement - validation des arrivages fournisseurs avant stockage'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'PRE_ENREGISTREMENT');

SELECT setval(pg_get_serial_sequence('roles', 'id'), (SELECT MAX(id) FROM roles));

-- Où sont parties les montures d'une proforma réglée, en colonne plutôt que recalculé à
-- chaque affichage.
--
-- Jusqu'ici rien ne le stockait (voir le commentaire historique resté sur
-- resolveDestination() côté front, caisse.tsx) : la Caisse le déduisait de l'état courant
-- des montures, croisé à la volée avec les lignes de la proforma.
--
-- On ne persiste que la moitié « labo » de ce calcul. Une monture vendue et expédiée au
-- laboratoire ne revient jamais en arrière — CloseIfComplete peut l'écrire une fois pour
-- toutes au moment où la proforma se ferme. Une monture réservée, en revanche, peut être
-- libérée plus tard sans que la proforma ne bouge : figer « reserve » ici referait exactement
-- le bug que le calcul à la volée existait pour éviter (des proformas « en réserve » qui ne
-- le sont plus). Le front continue donc de déduire « reserve » du statut RESERVEE courant des
-- montures, et ne lit cette colonne que pour « labo ».
ALTER TABLE proformas ADD COLUMN IF NOT EXISTS destination VARCHAR(20) NULL;

COMMENT ON COLUMN proformas.destination IS
    'Écrite uniquement à ''labo'' par CloseIfComplete quand au moins une ligne est VENDUE. '
    'NULL sinon — le cas ''reserve'' reste recalculé côté front depuis le statut RESERVEE.';

-- Rattrape les proformas déjà réglées avant cette migration : sans ce backfill, seules les
-- clôtures futures porteraient la colonne, et « déjà réglée mais destination NULL » serait
-- indiscernable de « jamais vendue ».
UPDATE proformas p
SET destination = 'labo'
WHERE p.destination IS NULL
  AND EXISTS (
      SELECT 1 FROM proforma_items i
      WHERE i.proforma_id = p.id AND i.outcome = 'VENDUE'
  );

-- Ordonnance et détail de facturation d'une proforma, en colonnes.
--
-- Jusqu'ici tout cela transitait dans proformas.note : type de verres, teinte, les deux
-- yeux, accessoires, montage, remise. Conservé, mais illisible pour SQL — impossible de
-- compter les progressifs d'un mois ou de sortir le chiffre des accessoires.
--
-- proformas.note N'EST PAS supprimée et continue d'être écrite. Deux raisons :
--   1. la Caisse réaffiche la note telle quelle, elle doit y lire une ordonnance ;
--   2. elle sert de filet si une valeur saisie à la main ne se convertit pas en nombre
--      (« +1. » à moitié tapé) : la colonne reste NULL, la note garde le texte.
--
-- Table séparée plutôt que colonnes ajoutées à proformas : l'en-tête est lu en boucle
-- par l'écran de la Caisse, qui n'a que faire de l'axe de l'œil gauche.
CREATE TABLE IF NOT EXISTS proforma_prescriptions (
    proforma_id BIGINT PRIMARY KEY REFERENCES proformas(id) ON DELETE CASCADE,

    societe VARCHAR(160) NULL,
    -- Simple foyer / Double foyer / Progressif, et Teinte A,B,C / Photo-Transit° / Blanc.
    -- En texte : ce sont les libellés du formulaire papier, ils bougent sans migration.
    foyer VARCHAR(40) NULL,
    teinte VARCHAR(40) NULL,

    -- Œil droit puis œil gauche. NUMERIC et non VARCHAR : c'est ce qui rend une
    -- ordonnance interrogeable, et c'était tout l'objet de cette table. NULL quand le
    -- champ est vide ou illisible — un 0 signifierait « sphère nulle », ce qui est une
    -- correction valide et donc un mensonge.
    od_sphere NUMERIC(5,2) NULL,
    od_cylindre NUMERIC(5,2) NULL,
    od_axe SMALLINT NULL CHECK (od_axe IS NULL OR (od_axe >= 0 AND od_axe <= 180)),
    od_addition NUMERIC(5,2) NULL,
    od_prix NUMERIC(12,2) NOT NULL DEFAULT 0,

    og_sphere NUMERIC(5,2) NULL,
    og_cylindre NUMERIC(5,2) NULL,
    og_axe SMALLINT NULL CHECK (og_axe IS NULL OR (og_axe >= 0 AND og_axe <= 180)),
    og_addition NUMERIC(5,2) NULL,
    og_prix NUMERIC(12,2) NOT NULL DEFAULT 0,

    accessoires_label VARCHAR(200) NULL,
    accessoires_prix NUMERIC(12,2) NOT NULL DEFAULT 0,
    montage_prix NUMERIC(12,2) NOT NULL DEFAULT 0,
    remise_pct NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (remise_pct >= 0 AND remise_pct <= 100),

    -- Ce que la vendeuse écrit en clair, hors grille (« retrait prévu vendredi »).
    note_libre TEXT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- « Combien de progressifs ce mois-ci » : la requête qui a motivé cette table.
CREATE INDEX IF NOT EXISTS idx_proforma_prescriptions_foyer ON proforma_prescriptions(foyer);

-- Les attributs de la monture que la 025 disait recopier « pour que le document reste
-- lisible même si la monture est supprimée ». reference et brand existaient déjà mais le
-- handler ne les remplissait jamais ; forme et couleur n'avaient pas de colonne du tout.
-- Une ligne dont le glass_id est passé à NULL ne portait donc plus qu'un code-barres.
ALTER TABLE proforma_items ADD COLUMN IF NOT EXISTS shape VARCHAR(60) NULL;
ALTER TABLE proforma_items ADD COLUMN IF NOT EXISTS color VARCHAR(60) NULL;

-- Le geste commercial « monture offerte » existait à l'écran et sur le papier, jamais en
-- base : le serveur relisait le prix de la fiche monture et facturait plein tarif. La
-- colonne le dit explicitement, plutôt que de le déduire d'un unit_price à 0 — une
-- monture peut valoir 0 pour d'autres raisons (prix non renseigné).
ALTER TABLE proforma_items ADD COLUMN IF NOT EXISTS offerte BOOLEAN NOT NULL DEFAULT FALSE;

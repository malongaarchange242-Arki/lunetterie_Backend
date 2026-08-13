-- Table de référence des sociétés conventionnées.
--
-- `societe` était un champ texte libre, ressaisi à chaque proforma (027) : « TOTAL »,
-- « Total » et « TOTAL E&P » y devenaient trois sociétés distinctes, et compter les
-- proformas d'un compte client relevait du rapprochement de chaînes. La liste est
-- désormais tenue par la Direction ; la vendeuse choisit, elle ne saisit plus.
CREATE TABLE IF NOT EXISTS societes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(160) NOT NULL,

    -- Facultatifs : une convention se négocie avec un interlocuteur, qu'on veut pouvoir
    -- rappeler sans sortir de l'application.
    contact VARCHAR(160) NULL,
    phone VARCHAR(60) NULL,

    -- Désactivée plutôt que supprimée. Une société qui met fin à sa convention doit
    -- disparaître de la liste de la vendeuse, mais les proformas déjà émises gardent leur
    -- lien : une suppression casserait la clé étrangère ou effacerait l'historique.
    active BOOLEAN NOT NULL DEFAULT TRUE,

    created_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unicité insensible à la casse et aux espaces de bord : c'est exactement le doublon que
-- cette table existe pour empêcher. Index d'expression, faute de pouvoir poser une
-- contrainte UNIQUE sur une fonction.
CREATE UNIQUE INDEX IF NOT EXISTS idx_societes_name_unique ON societes (LOWER(TRIM(name)));

-- La liste que demande la vendeuse : les actives, par ordre alphabétique.
CREATE INDEX IF NOT EXISTS idx_societes_active ON societes (active, name);

-- PARTICULIER est la valeur par défaut du formulaire depuis toujours. Sans elle en base,
-- la liste s'ouvrirait sans option pour le cas de loin le plus courant.
INSERT INTO societes (name) VALUES ('PARTICULIER') ON CONFLICT DO NOTHING;

-- Le lien vers la table, à côté du nom déjà recopié — les deux, et non l'un ou l'autre.
-- `societe_id` rend les proformas d'une société comptables en SQL ; `societe` garde le
-- document lisible si la société est renommée ou désactivée. C'est le motif déjà appliqué
-- aux lignes de proforma, qui recopient les attributs de la monture à l'émission.
ALTER TABLE proforma_prescriptions
    ADD COLUMN IF NOT EXISTS societe_id BIGINT NULL REFERENCES societes(id);

CREATE INDEX IF NOT EXISTS idx_proforma_prescriptions_societe
    ON proforma_prescriptions(societe_id);

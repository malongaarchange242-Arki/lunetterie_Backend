-- Listes d'envoi : quand la direction envoie la liste d'une session de réception terminée
-- vers un magasin, elle est archivée ici. Le poste de scan (stock général) les scrute pour
-- prévenir le magasinier qu'une nouvelle liste est à préparer.
CREATE TABLE IF NOT EXISTS send_lists (
    id BIGSERIAL PRIMARY KEY,
    session_code VARCHAR(60) NOT NULL,
    city VARCHAR(100) NOT NULL,
    item_count INTEGER NOT NULL DEFAULT 0,
    -- NOUVELLE : jamais ouverte, c'est elle qui déclenche la notification.
    -- VUE      : le poste de scan l'a affichée.
    -- TRAITEE  : le colis est préparé.
    status VARCHAR(20) NOT NULL DEFAULT 'NOUVELLE' CHECK (status IN ('NOUVELLE', 'VUE', 'TRAITEE')),
    created_by BIGINT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Le détail est recopié (barcode, référence, marque, emplacement) plutôt que seulement
-- référencé : une liste envoyée doit rester lisible telle qu'elle était, même si la monture
-- bouge ou disparaît ensuite. D'où aussi le ON DELETE SET NULL sur glass_id.
CREATE TABLE IF NOT EXISTS send_list_items (
    id BIGSERIAL PRIMARY KEY,
    list_id BIGINT NOT NULL REFERENCES send_lists(id) ON DELETE CASCADE,
    glass_id BIGINT NULL REFERENCES glasses(id) ON DELETE SET NULL,
    barcode VARCHAR(128) NULL,
    reference VARCHAR(120) NULL,
    brand VARCHAR(120) NULL,
    location_code VARCHAR(80) NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Le poste de scan interroge en boucle les listes NOUVELLE, les plus récentes d'abord.
CREATE INDEX IF NOT EXISTS idx_send_lists_status_created ON send_lists(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_send_lists_city ON send_lists(city);
CREATE INDEX IF NOT EXISTS idx_send_list_items_list ON send_list_items(list_id);

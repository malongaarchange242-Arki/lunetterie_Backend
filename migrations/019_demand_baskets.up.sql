-- Paniers de demande : chaque recherche de monture faite via le chatbot pour un magasin
-- y dépose une ligne. Le « panier » n'est pas une entité propre — c'est le regroupement
-- des lignes d'une même ville, et son compteur est le nombre de lignes encore OUVERTE.
CREATE TABLE IF NOT EXISTS demand_baskets (
    id BIGSERIAL PRIMARY KEY,
    city VARCHAR(100) NOT NULL,
    genre VARCHAR(30) NULL,
    forme VARCHAR(50) NULL,
    gamme VARCHAR(30) NULL,
    taille VARCHAR(30) NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'CHATBOT' CHECK (source IN ('CHATBOT', 'MANUEL')),
    -- OUVERTE : demande en attente, compte dans le panier.
    -- ENVOYEE : reprise dans une demande adressée au stock principal, sort du compteur.
    -- ANNULEE : écartée à la main.
    status VARCHAR(20) NOT NULL DEFAULT 'OUVERTE' CHECK (status IN ('OUVERTE', 'ENVOYEE', 'ANNULEE')),
    created_by BIGINT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Le compteur de chaque panier est un COUNT filtré sur (city, status) : c'est la requête
-- la plus fréquente de l'écran stock.
CREATE INDEX IF NOT EXISTS idx_demand_baskets_city_status ON demand_baskets(city, status);
CREATE INDEX IF NOT EXISTS idx_demand_baskets_created_at ON demand_baskets(created_at DESC);

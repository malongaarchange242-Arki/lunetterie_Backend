CREATE TABLE IF NOT EXISTS pre_registration_cases (
    id BIGSERIAL PRIMARY KEY,
    reception_command_id BIGINT NOT NULL REFERENCES reception_commands(id) ON DELETE CASCADE,
    code VARCHAR(80) NOT NULL UNIQUE,
    couleur VARCHAR(80) NOT NULL,
    hex VARCHAR(16) NULL,
    gamme VARCHAR(80) NOT NULL,
    genre VARCHAR(40) NOT NULL,
    montures INTEGER NOT NULL CHECK (montures > 0),
    validated BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pre_registration_cases_command
    ON pre_registration_cases(reception_command_id);

CREATE TABLE IF NOT EXISTS pre_registration_boxes (
    id BIGSERIAL PRIMARY KEY,
    case_id BIGINT NOT NULL REFERENCES pre_registration_cases(id) ON DELETE CASCADE,
    code VARCHAR(120) NOT NULL UNIQUE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    formes JSONB NOT NULL DEFAULT '{}'::jsonb,
    marques TEXT[] NOT NULL DEFAULT '{}',
    couleurs TEXT[] NOT NULL DEFAULT '{}',
    matieres TEXT[] NOT NULL DEFAULT '{}',
    gamme VARCHAR(80) NOT NULL,
    type_lunette VARCHAR(80) NOT NULL,
    prix NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (prix >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pre_registration_boxes_case
    ON pre_registration_boxes(case_id);

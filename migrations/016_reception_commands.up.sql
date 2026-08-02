CREATE TABLE IF NOT EXISTS reception_commands (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    target_count INTEGER NOT NULL CHECK (target_count > 0),
    registered_count INTEGER NOT NULL DEFAULT 0 CHECK (registered_count >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by BIGINT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reception_commands_code ON reception_commands(code);
CREATE INDEX IF NOT EXISTS idx_reception_commands_status ON reception_commands(status);

CREATE TABLE IF NOT EXISTS messages (
                                        id         BIGSERIAL PRIMARY KEY,
                                        sender     TEXT NOT NULL,
                                        body       TEXT NOT NULL,
                                        created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );
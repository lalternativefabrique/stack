-- The local account row everything in this core is keyed on. Authentication
-- is urbangate's: id is the Kratos identity id, and no credential lives here.
-- Domain tables that reference a person use TEXT to match "user"(id), with
-- ON DELETE CASCADE so an account erasure takes their rows along.

CREATE TABLE IF NOT EXISTS "user" (
    id           TEXT PRIMARY KEY,
    identity_id  TEXT NOT NULL UNIQUE,
    email        TEXT NOT NULL,
    name         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

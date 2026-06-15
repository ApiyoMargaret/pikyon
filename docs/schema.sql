CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    password        TEXT NOT NULL,
    pin_hash        TEXT,
    avatar_url      TEXT,
    preferred_lang  TEXT DEFAULT 'en',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS memories (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    story           TEXT,
    location        TEXT,
    memory_date     DATE,
    visibility      TEXT NOT NULL DEFAULT 'private'
                    CHECK (visibility IN ('private', 'shared')),
    is_locked       BOOLEAN DEFAULT FALSE,
    media_key       TEXT,
    media_type      TEXT DEFAULT 'none'
                    CHECK (media_type IN ('image', 'video', 'audio', 'none')),
    media_thumbnail TEXT,
    ai_vibe         TEXT,
    ai_vibe_reason  TEXT,
    ai_caption      TEXT,
    ai_tone         TEXT,
    tags            TEXT[],
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_memories_user_id ON memories(user_id);
CREATE INDEX idx_memories_created_at ON memories(created_at DESC);
CREATE INDEX idx_memories_visibility ON memories(visibility);

CREATE TABLE IF NOT EXISTS shared_access (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_token TEXT UNIQUE NOT NULL DEFAULT encode(gen_random_bytes(32), 'hex'),
    expires_at  TIMESTAMPTZ,
    view_count  INT DEFAULT 0,
    max_views   INT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_shared_access_token ON shared_access(share_token);

CREATE TABLE IF NOT EXISTS ai_analyses (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_id      UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    vibe           TEXT,
    vibe_reason    TEXT,
    tone           TEXT,
    captions       JSONB,
    suggested_tags TEXT[],
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER memories_updated_at
    BEFORE UPDATE ON memories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    
-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 001 — Schéma initial Home Cooking
-- Appliquée une seule fois au démarrage (gérée par internal/db/db.go).
-- ─────────────────────────────────────────────────────────────────────────────

-- ── Utilisateurs ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    username      TEXT     NOT NULL UNIQUE COLLATE NOCASE,  -- 3-32 chars, alphanum + ._-
    password_hash TEXT     NOT NULL,                        -- bcrypt hash (coût 12)
    role          TEXT     NOT NULL DEFAULT 'admin',        -- 'admin' | 'user'
    created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at    DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Index pour les lookups par username (login, vérification doublon)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- ── Refresh tokens (rotation + révocation) ───────────────────────────────────
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT     NOT NULL UNIQUE,                   -- JWT signé complet
    expires_at DATETIME NOT NULL,
    revoked    INTEGER  NOT NULL DEFAULT 0,                -- 0 = valide, 1 = révoqué
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);

-- ── Recettes ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS recipes (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT     NOT NULL CHECK(length(name) BETWEEN 2 AND 120),
    description TEXT,
    servings    INTEGER  NOT NULL DEFAULT 4 CHECK(servings BETWEEN 1 AND 50),
    prep_time   INTEGER,                                    -- minutes
    cook_time   INTEGER,                                    -- minutes
    difficulty  TEXT     CHECK(difficulty IN ('facile','moyen','difficile')),
    tags        TEXT     DEFAULT '[]',                      -- JSON array ["végé","riz"]
    image_url   TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_recipes_user ON recipes(user_id);

-- ── Ingrédients d'une recette ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS recipe_ingredients (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    quantity   REAL,
    unit       TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_recipe ON recipe_ingredients(recipe_id);

-- ── Étapes de préparation ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS recipe_steps (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id  INTEGER NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_order INTEGER NOT NULL,
    content    TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recipe_steps_recipe ON recipe_steps(recipe_id);

-- ── Recherche plein-texte (FTS5) ──────────────────────────────────────────────
-- Table virtuelle FTS5 synchronisée avec recipes via triggers
CREATE VIRTUAL TABLE IF NOT EXISTS recipes_fts USING fts5(
    name,
    description,
    tags,
    content='recipes',
    content_rowid='id',
    tokenize='unicode61'
);

-- Trigger de synchronisation FTS après INSERT
CREATE TRIGGER IF NOT EXISTS recipes_ai AFTER INSERT ON recipes BEGIN
    INSERT INTO recipes_fts(rowid, name, description, tags)
    VALUES (new.id, new.name, new.description, new.tags);
END;

-- Trigger de synchronisation FTS après UPDATE
CREATE TRIGGER IF NOT EXISTS recipes_au AFTER UPDATE ON recipes BEGIN
    INSERT INTO recipes_fts(recipes_fts, rowid, name, description, tags)
    VALUES ('delete', old.id, old.name, old.description, old.tags);
    INSERT INTO recipes_fts(rowid, name, description, tags)
    VALUES (new.id, new.name, new.description, new.tags);
END;

-- Trigger de synchronisation FTS après DELETE
CREATE TRIGGER IF NOT EXISTS recipes_ad AFTER DELETE ON recipes BEGIN
    INSERT INTO recipes_fts(recipes_fts, rowid, name, description, tags)
    VALUES ('delete', old.id, old.name, old.description, old.tags);
END;

-- ── Stockage / Inventaire ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS storage_items (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT     NOT NULL CHECK(length(name) BETWEEN 1 AND 120),
    quantity   REAL     NOT NULL DEFAULT 0 CHECK(quantity >= 0),
    unit       TEXT     NOT NULL DEFAULT '',
    category   TEXT     NOT NULL DEFAULT '',
    expiry     TEXT,                                        -- ISO date "2026-04-30"
    alert_at   REAL     NOT NULL DEFAULT 0,                 -- seuil d'alerte (même unité)
    notes      TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_storage_user     ON storage_items(user_id);
CREATE INDEX IF NOT EXISTS idx_storage_category ON storage_items(user_id, category);

-- ── Trigger updated_at automatique ───────────────────────────────────────────
CREATE TRIGGER IF NOT EXISTS users_updated_at
    AFTER UPDATE ON users FOR EACH ROW
    BEGIN UPDATE users SET updated_at = datetime('now') WHERE id = old.id; END;

CREATE TRIGGER IF NOT EXISTS recipes_updated_at
    AFTER UPDATE ON recipes FOR EACH ROW
    BEGIN UPDATE recipes SET updated_at = datetime('now') WHERE id = old.id; END;

CREATE TRIGGER IF NOT EXISTS storage_updated_at
    AFTER UPDATE ON storage_items FOR EACH ROW
    BEGIN UPDATE storage_items SET updated_at = datetime('now') WHERE id = old.id; END;

-- migrations/0001_init.sql
-- The first migration. Creates the four core tables from PLAN.md.
--
-- Apply with:
--   rm -f music.db
--   sqlite3 music.db < migrations/0001_init.sql
--
-- (L9 will add the unique index for duplicate detection.)

-- ---------------------------------------------------------------------------
-- artists
-- ---------------------------------------------------------------------------
CREATE TABLE artists (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id  TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    genres      TEXT,                          -- JSON array, e.g. '["synthwave","electronic"]'
    followers   INTEGER,
    popularity  INTEGER,                       -- 0-100, Spotify's scale
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- ---------------------------------------------------------------------------
-- albums
-- ---------------------------------------------------------------------------
CREATE TABLE albums (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id   TEXT    NOT NULL UNIQUE,
    name         TEXT    NOT NULL,
    release_date TEXT,                         -- 'YYYY-MM-DD' or 'YYYY'
    total_tracks INTEGER,
    cover_url    TEXT,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- ---------------------------------------------------------------------------
-- tracks
-- ---------------------------------------------------------------------------
CREATE TABLE tracks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id   TEXT    NOT NULL UNIQUE,
    name         TEXT    NOT NULL,
    duration_ms  INTEGER NOT NULL,
    explicit     BOOLEAN NOT NULL DEFAULT 0,
    disc_number  INTEGER,
    track_number INTEGER,
    popularity   INTEGER,
    preview_url  TEXT,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- ---------------------------------------------------------------------------
-- play_events — the most important table. One row per play.
-- ---------------------------------------------------------------------------
CREATE TABLE play_events (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id  INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    played_at TEXT    NOT NULL,               -- RFC3339, e.g. '2024-03-15T14:30:00Z'
    device    TEXT,
    shuffle   BOOLEAN,
    repeat    TEXT,                            -- 'off' | 'track' | 'context'
    context   TEXT,                            -- URI of playlist/album/etc.
    source    TEXT                             -- 'collector' | 'manual' | ...
);

-- ---------------------------------------------------------------------------
-- Indexes for the common queries.
-- The duplicate-detection unique index lands in L9 (migrations/0002_*.sql).
-- ---------------------------------------------------------------------------
CREATE INDEX idx_play_events_played_at ON play_events(played_at);
CREATE INDEX idx_play_events_track_id  ON play_events(track_id);

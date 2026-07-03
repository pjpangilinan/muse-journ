CREATE UNIQUE INDEX IF NOT EXISTS idx_play_events_dedup
    ON play_events(track_id, played_at);

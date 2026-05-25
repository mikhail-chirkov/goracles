CREATE TABLE IF NOT EXISTS users (
    telegram_user_id INTEGER PRIMARY KEY,
    default_radius_km REAL DEFAULT 1,
    default_trip_duration_min INTEGER DEFAULT 20,
    default_time_frame_min INTEGER DEFAULT 60,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

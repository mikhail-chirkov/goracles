package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed db/schema.sql
var schemaFS embed.FS

type userDefaults struct {
	TelegramUserID  int64
	DefaultRadiusKM float64
	TripDurationMin int
	TimeFrameMin    int
}

type userStore struct {
	db *sql.DB
}

func openUserStore(ctx context.Context, databasePath string) (*userStore, error) {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	store := &userStore{db: db}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *userStore) close() error {
	return s.db.Close()
}

func (s *userStore) migrate(ctx context.Context) error {
	schema, err := schemaFS.ReadFile("db/schema.sql")
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, string(schema))
	return err
}

func (s *userStore) ensureUser(ctx context.Context, telegramUserID int64) (userDefaults, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (telegram_user_id)
		VALUES (?)
		ON CONFLICT(telegram_user_id) DO NOTHING
	`, telegramUserID)
	if err != nil {
		return userDefaults{}, err
	}

	return s.getDefaults(ctx, telegramUserID)
}

func (s *userStore) getDefaults(ctx context.Context, telegramUserID int64) (userDefaults, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT telegram_user_id, default_radius_km, default_trip_duration_min, default_time_frame_min
		FROM users
		WHERE telegram_user_id = ?
	`, telegramUserID)

	var defaults userDefaults
	err := row.Scan(&defaults.TelegramUserID, &defaults.DefaultRadiusKM, &defaults.TripDurationMin, &defaults.TimeFrameMin)
	if err != nil {
		return userDefaults{}, err
	}

	return defaults, nil
}

func (s *userStore) updateRadius(ctx context.Context, telegramUserID int64, radiusKM float64) (userDefaults, error) {
	if radiusKM <= 0 {
		return userDefaults{}, fmt.Errorf("radius must be positive")
	}

	if _, err := s.ensureUser(ctx, telegramUserID); err != nil {
		return userDefaults{}, err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET default_radius_km = ?
		WHERE telegram_user_id = ?
	`, radiusKM, telegramUserID)
	if err != nil {
		return userDefaults{}, err
	}

	return s.getDefaults(ctx, telegramUserID)
}

func (s *userStore) updateTripDuration(ctx context.Context, telegramUserID int64, minutes int) (userDefaults, error) {
	if minutes < 5 {
		return userDefaults{}, fmt.Errorf("trip duration must be at least 5 minutes")
	}

	if _, err := s.ensureUser(ctx, telegramUserID); err != nil {
		return userDefaults{}, err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET
			default_trip_duration_min = ?,
			default_time_frame_min = max(default_time_frame_min, ?)
		WHERE telegram_user_id = ?
	`, minutes, minutes, telegramUserID)
	if err != nil {
		return userDefaults{}, err
	}

	return s.getDefaults(ctx, telegramUserID)
}

func (s *userStore) updateTimeFrame(ctx context.Context, telegramUserID int64, minutes int) (userDefaults, error) {
	if minutes < 5 {
		return userDefaults{}, fmt.Errorf("time frame must be at least 5 minutes")
	}

	if _, err := s.ensureUser(ctx, telegramUserID); err != nil {
		return userDefaults{}, err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET
			default_time_frame_min = ?,
			default_trip_duration_min = min(default_trip_duration_min, ?)
		WHERE telegram_user_id = ?
	`, minutes, minutes, telegramUserID)
	if err != nil {
		return userDefaults{}, err
	}

	return s.getDefaults(ctx, telegramUserID)
}

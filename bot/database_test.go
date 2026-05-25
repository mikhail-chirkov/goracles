package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserStoreDefaults(t *testing.T) {
	ctx := context.Background()
	store, err := openUserStore(ctx, filepath.Join(t.TempDir(), "bot.sqlite"))
	require.NoError(t, err)
	defer store.close()

	defaults, err := store.ensureUser(ctx, 42)
	require.NoError(t, err)

	assert.Equal(t, int64(42), defaults.TelegramUserID)
	assert.Equal(t, 1.0, defaults.DefaultRadiusKM)
	assert.Equal(t, 20, defaults.TripDurationMin)
	assert.Equal(t, 60, defaults.TimeFrameMin)
}

func TestUserStoreKeepsDurationWithinTimeFrame(t *testing.T) {
	ctx := context.Background()
	store, err := openUserStore(ctx, filepath.Join(t.TempDir(), "bot.sqlite"))
	require.NoError(t, err)
	defer store.close()

	defaults, err := store.updateTripDuration(ctx, 42, 120)
	require.NoError(t, err)
	assert.Equal(t, 120, defaults.TripDurationMin)
	assert.Equal(t, 120, defaults.TimeFrameMin)

	defaults, err = store.updateTimeFrame(ctx, 42, 30)
	require.NoError(t, err)
	assert.Equal(t, 30, defaults.TripDurationMin)
	assert.Equal(t, 30, defaults.TimeFrameMin)
}

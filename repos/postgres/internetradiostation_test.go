package postgres

import (
	"context"
	"errors"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInternetRadioStationRepository(t *testing.T) {
	db, _ := thSetupDatabase(t)

	repo := db.InternetRadioStation()

	ctx := context.Background()

	assert.Equalf(t, 0, thCount(t, db, "internet_radio_stations"),
		"there should be no internet radio stations at beginning of test")

	createStation := func(user string) string {
		station, err := repo.Create(ctx, user, repos.CreateInternetRadioStationParams{
			Name:        "Test Station",
			StreamURL:   "https://radio.example.com/stream",
			HomepageURL: util.ToPtr("https://radio.example.com"),
		})
		require.NoErrorf(t, err, "create internet radio station: %v", err)
		return station.ID
	}

	user := thCreateUser(t, db)
	user2 := thCreateUser(t, db)

	t.Run("Create", func(t *testing.T) {
		t.Run("create with homepage url", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station, err := repo.Create(ctx, user, repos.CreateInternetRadioStationParams{
				Name:        "Test Station",
				StreamURL:   "https://radio.example.com/stream",
				HomepageURL: util.ToPtr("https://radio.example.com"),
			})
			require.NoErrorf(t, err, "create test station")
			require.NotNil(t, station)
			assert.Truef(t, crossonic.IsIDType(station.ID, crossonic.IDTypeInternetRadioStation),
				"expected valid ID, got: %s", station.ID)
			assert.Equal(t, "Test Station", station.Name)
			assert.Equal(t, "https://radio.example.com/stream", station.StreamURL)
			assert.Equal(t, util.ToPtr("https://radio.example.com"), station.HomepageURL)

			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station.ID,
				"user_name":    user,
				"name":         station.Name,
				"stream_url":   "https://radio.example.com/stream",
				"homepage_url": "https://radio.example.com",
			}), "created station should exist in db")
		})

		t.Run("create without homepage url", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station, err := repo.Create(ctx, user, repos.CreateInternetRadioStationParams{
				Name:      "Test Station",
				StreamURL: "https://radio.example.com/stream",
			})
			require.NoErrorf(t, err, "create test station")
			require.NotNil(t, station)
			assert.Truef(t, crossonic.IsIDType(station.ID, crossonic.IDTypeInternetRadioStation),
				"expected valid ID, got: %s", station.ID)
			assert.Equal(t, "Test Station", station.Name)
			assert.Equal(t, "https://radio.example.com/stream", station.StreamURL)
			assert.Nil(t, station.HomepageURL)

			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station.ID,
				"name":         station.Name,
				"user_name":    user,
				"stream_url":   "https://radio.example.com/stream",
				"homepage_url": nil,
			}), "created station should exist in db")
		})
	})

	t.Run("FindAll", func(t *testing.T) {

		t.Run("no stations", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			stations, err := repo.FindAll(ctx, user)
			require.NoErrorf(t, err, "find all stations: %v", err)
			assert.Equal(t, 0, len(stations), "there should be no stations")
		})

		t.Run("two stations same user", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")

			station1 := createStation(user)
			station2 := createStation(user)

			stations, err := repo.FindAll(ctx, user)
			require.NoErrorf(t, err, "find all stations: %v", err)
			assert.Equal(t, 2, len(stations), "there should be two stations")
			assert.Contains(t, util.Map(stations, func(s *repos.InternetRadioStation) string {
				return s.ID
			}), station1)
			assert.Contains(t, util.Map(stations, func(s *repos.InternetRadioStation) string {
				return s.ID
			}), station2)
		})

		t.Run("different users", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")

			createStation(user)
			createStation(user)

			station1U2 := createStation(user2)
			station2U2 := createStation(user2)

			stations, err := repo.FindAll(ctx, user2)
			require.NoErrorf(t, err, "find all stations: %v", err)
			assert.Equal(t, 2, len(stations), "there should be two stations")
			assert.Contains(t, util.Map(stations, func(s *repos.InternetRadioStation) string {
				return s.ID
			}), station1U2)
			assert.Contains(t, util.Map(stations, func(s *repos.InternetRadioStation) string {
				return s.ID
			}), station2U2)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("wrong id", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			createStation(user)
			err := repo.Update(ctx, user, "asdf", repos.UpdateInternetRadioStationParams{
				Name:        repos.NewOptionalFull("updated"),
				StreamURL:   repos.NewOptionalFull("updated"),
				HomepageURL: repos.NewOptionalFull(util.ToPtr("updated")),
			})
			assert.Truef(t, errors.Is(err, repos.ErrNotFound), "expected ErrNotFound, got %v", err)
			assert.False(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"name": "updated",
			}))
		})

		t.Run("wrong user", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			err := repo.Update(ctx, user2, station, repos.UpdateInternetRadioStationParams{
				Name:        repos.NewOptionalFull("updated"),
				StreamURL:   repos.NewOptionalFull("updated"),
				HomepageURL: repos.NewOptionalFull(util.ToPtr("updated")),
			})
			assert.Truef(t, errors.Is(err, repos.ErrNotFound), "expected ErrNotFound, got %v", err)
			assert.False(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"name": "updated",
			}))
		})

		t.Run("empty update", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			err := repo.Update(ctx, user, station, repos.UpdateInternetRadioStationParams{})
			require.NoErrorf(t, err, "update internet radio station: %v", err)
			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station,
				"user_name":    user,
				"name":         "Test Station",
				"stream_url":   "https://radio.example.com/stream",
				"homepage_url": "https://radio.example.com",
			}))
		})

		t.Run("full update", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			err := repo.Update(ctx, user, station, repos.UpdateInternetRadioStationParams{
				Name:        repos.NewOptionalFull("updated"),
				StreamURL:   repos.NewOptionalFull("updated-stream"),
				HomepageURL: repos.NewOptionalFull(util.ToPtr("updated-homepage")),
			})
			require.NoErrorf(t, err, "update internet radio station: %v", err)
			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station,
				"user_name":    user,
				"name":         "updated",
				"stream_url":   "updated-stream",
				"homepage_url": "updated-homepage",
			}))
		})

		t.Run("full update with null homepage", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			err := repo.Update(ctx, user, station, repos.UpdateInternetRadioStationParams{
				Name:        repos.NewOptionalFull("updated"),
				StreamURL:   repos.NewOptionalFull("updated-stream"),
				HomepageURL: repos.NewOptionalFull[*string](nil),
			})
			require.NoErrorf(t, err, "update internet radio station: %v", err)
			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station,
				"user_name":    user,
				"name":         "updated",
				"stream_url":   "updated-stream",
				"homepage_url": nil,
			}))
		})

		t.Run("only update name", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			err := repo.Update(ctx, user, station, repos.UpdateInternetRadioStationParams{
				Name: repos.NewOptionalFull("updated"),
			})
			require.NoErrorf(t, err, "update internet radio stations: %v", err)
			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station,
				"user_name":    user,
				"name":         "updated",
				"stream_url":   "https://radio.example.com/stream",
				"homepage_url": util.ToPtr("https://radio.example.com"),
			}))
		})

		t.Run("only one station is updated", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			station2 := createStation(user)
			station3 := createStation(user2)
			err := repo.Update(ctx, user, station, repos.UpdateInternetRadioStationParams{
				Name:        repos.NewOptionalFull("updated"),
				StreamURL:   repos.NewOptionalFull("updated-stream"),
				HomepageURL: repos.NewOptionalFull[*string](nil),
			})
			require.NoErrorf(t, err, "update internet radio station: %v", err)
			assert.Truef(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station,
				"user_name":    user,
				"name":         "updated",
				"stream_url":   "updated-stream",
				"homepage_url": nil,
			}), "selected station is updated")
			assert.Truef(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station2,
				"user_name":    user,
				"name":         "Test Station",
				"stream_url":   "https://radio.example.com/stream",
				"homepage_url": util.ToPtr("https://radio.example.com"),
			}), "not selected station is not updated")
			assert.Truef(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id":           station3,
				"user_name":    user2,
				"name":         "Test Station",
				"stream_url":   "https://radio.example.com/stream",
				"homepage_url": util.ToPtr("https://radio.example.com"),
			}), "not selected station is not updated")
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("wrong id", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			err := repo.Delete(ctx, user, "asdf")
			assert.Truef(t, errors.Is(err, repos.ErrNotFound), "expected ErrNotFound, got %v", err)
			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id": station,
			}))
		})
		t.Run("wrong user", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station := createStation(user)
			err := repo.Delete(ctx, user2, station)
			assert.Truef(t, errors.Is(err, repos.ErrNotFound), "expected ErrNotFound, got %v", err)
			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id": station,
			}))
		})
		t.Run("correct", func(t *testing.T) {
			thDeleteAll(t, db, "internet_radio_stations")
			station1 := createStation(user)
			station2 := createStation(user)
			err := repo.Delete(ctx, user, station1)
			require.NoErrorf(t, err, "delete internet radio station: %v", err)
			assert.False(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id": station1,
			}))
			assert.True(t, thExists(t, db, "internet_radio_stations", map[string]any{
				"id": station2,
			}))
		})
	})
}

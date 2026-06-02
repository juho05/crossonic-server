package postgres

import (
	"context"
	"testing"
	"time"

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlbumRepository(t *testing.T) {
	db, _ := thSetupDatabase(t)
	ctx := context.Background()
	repo := db.Album()

	user := thCreateUser(t, db)
	user2 := thCreateUser(t, db)

	t.Run("Create", func(t *testing.T) {
		t.Run("creates album with correct fields", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			mbid := "test-album-mbid"
			id, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "My Album",
				MusicBrainzID: &mbid,
				MusicFolderID: folderID,
			})
			require.NoErrorf(t, err, "create album: %v", err)
			assert.True(t, crossonic.IsIDType(id, crossonic.IDTypeAlbum), "expected album ID type, got: %s", id)
			assert.True(t, thExists(t, db, "albums", map[string]any{
				"id":              id,
				"name":            "My Album",
				"music_brainz_id": mbid,
				"music_folder_id": folderID,
			}))
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("update name", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			err := repo.Update(ctx, albumID, repos.UpdateAlbumParams{
				Name:        repos.NewOptionalFull("Updated Album"),
				ArtistNames: repos.NewOptionalFull([]string{}),
			})
			require.NoErrorf(t, err, "update album: %v", err)
			assert.True(t, thExists(t, db, "albums", map[string]any{"id": albumID, "name": "Updated Album"}))
		})

		t.Run("update music folder", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			newFolderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			err := repo.Update(ctx, albumID, repos.UpdateAlbumParams{
				MusicFolderID: repos.NewOptionalFull(newFolderID),
			})
			require.NoErrorf(t, err, "update album: %v", err)
			assert.True(t, thExists(t, db, "albums", map[string]any{"id": albumID, "music_folder_id": newFolderID}))
		})

		t.Run("empty update does not error", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			err := repo.Update(ctx, albumID, repos.UpdateAlbumParams{})
			assert.NoErrorf(t, err, "empty update: %v", err)
		})

		t.Run("album does not exist", func(t *testing.T) {
			err := repo.Update(ctx, "al_doesnotexist", repos.UpdateAlbumParams{
				Name:        repos.NewOptionalFull("X"),
				ArtistNames: repos.NewOptionalFull([]string{}),
			})
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})

		t.Run("error when Name set without ArtistNames", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			err := repo.Update(ctx, albumID, repos.UpdateAlbumParams{
				Name: repos.NewOptionalFull("X"),
			})
			assert.Error(t, err)
		})

		t.Run("error when ArtistNames set without Name", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			err := repo.Update(ctx, albumID, repos.UpdateAlbumParams{
				ArtistNames: repos.NewOptionalFull([]string{"Artist"}),
			})
			assert.Error(t, err)
		})
	})

	t.Run("DeleteIfNoTracks", func(t *testing.T) {
		t.Run("deletes album with no songs", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			err := repo.DeleteIfNoTracks(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.False(t, thExists(t, db, "albums", map[string]any{"id": albumID}))
		})

		t.Run("keeps album with at least one song", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			thCreateSong(t, db, &albumID, folderID)
			err := repo.DeleteIfNoTracks(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.True(t, thExists(t, db, "albums", map[string]any{"id": albumID}))
		})
	})

	t.Run("FindByID", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)

		t.Run("returns album for authorized user", func(t *testing.T) {
			a, err := repo.FindByID(ctx, albumID, user, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a)
			assert.Equal(t, albumID, a.ID)
		})

		t.Run("returns not found for unauthorized user", func(t *testing.T) {
			isolatedFolder := thCreateMusicFolder(t, db)
			isolatedAlbum := thCreateAlbum(t, db, isolatedFolder)
			_, err := repo.FindByID(ctx, isolatedAlbum, user, repos.IncludeAlbumInfoBare())
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})

		t.Run("includes track count and duration", func(t *testing.T) {
			thCreateSong(t, db, &albumID, folderID)
			a, err := repo.FindByID(ctx, albumID, user, repos.IncludeAlbumInfo{TrackInfo: true})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a.AlbumTrackInfo)
			assert.GreaterOrEqual(t, a.TrackCount, 1)
			assert.Greater(t, a.Duration.Millis(), int64(0))
		})

		t.Run("includes annotation info", func(t *testing.T) {
			require.NoError(t, repo.Star(ctx, user, albumID))
			a, err := repo.FindByID(ctx, albumID, user, repos.IncludeAlbumInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a.AlbumAnnotations)
			assert.NotNil(t, a.Starred)
		})

		t.Run("includes artists", func(t *testing.T) {
			artistID := thCreateArtist(t, db)
			thAssociateMusicFolderArtist(t, db, artistID, folderID)
			_, err := db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING", albumID, artistID)
			require.NoError(t, err)
			a, err := repo.FindByID(ctx, albumID, user, repos.IncludeAlbumInfo{Artists: true})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a.AlbumLists)
			ids := util.Map(a.Artists, func(ar repos.ArtistRef) string { return ar.ID })
			assert.Contains(t, ids, artistID)
		})

		t.Run("includes genres", func(t *testing.T) {
			gFolderID := thCreateMusicFolder(t, db, user)
			gAlbumID := thCreateAlbum(t, db, gFolderID)
			songID := thCreateSong(t, db, &gAlbumID, gFolderID)
			thCreateGenre(t, db, "blues")
			require.NoError(t, db.Song().CreateGenreConnections(ctx, []repos.SongGenreConnection{{SongID: songID, Genre: "blues"}}))
			a, err := repo.FindByID(ctx, gAlbumID, user, repos.IncludeAlbumInfo{Genres: true})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a.AlbumLists)
			assert.Contains(t, a.Genres, "blues")
		})

		t.Run("includes play info", func(t *testing.T) {
			pFolderID := thCreateMusicFolder(t, db, user)
			pAlbumID := thCreateAlbum(t, db, pFolderID)
			songID := thCreateSong(t, db, &pAlbumID, pFolderID)
			require.NoError(t, db.Scrobble().CreateMultiple(ctx, []repos.CreateScrobbleParams{
				{
					User:         user,
					SongID:       songID,
					AlbumID:      &pAlbumID,
					Time:         time.Now().Add(-time.Hour),
					SongDuration: repos.NewDurationMS(300000),
					Duration:     repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true},
					NowPlaying:   false,
				},
			}))
			a, err := repo.FindByID(ctx, pAlbumID, user, repos.IncludeAlbumInfo{PlayInfo: true, User: user})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a.AlbumPlayInfo)
			assert.Equal(t, 1, a.PlayCount)
			assert.NotNil(t, a.LastPlayed)
		})
	})

	t.Run("FindAll", func(t *testing.T) {
		t.Run("returns albums in music folder", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			a1 := thCreateAlbum(t, db, folderID)
			a2 := thCreateAlbum(t, db, folderID)

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, a1)
			assert.Contains(t, ids, a2)
		})

		t.Run("does not return albums from other folders", func(t *testing.T) {
			folderA := thCreateMusicFolder(t, db, user)
			folderB := thCreateMusicFolder(t, db, user)
			inA := thCreateAlbum(t, db, folderA)
			inB := thCreateAlbum(t, db, folderB)

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				MusicFolderIDs: []int{folderA},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, inA)
			assert.NotContains(t, ids, inB)
		})

		t.Run("sort by created", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			a1 := thCreateAlbum(t, db, folderID)
			a2 := thCreateAlbum(t, db, folderID)

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByCreated,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, a1)
			assert.Contains(t, ids, a2)
		})

		t.Run("filter by genre", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumWithGenre := thCreateAlbum(t, db, folderID)
			albumWithoutGenre := thCreateAlbum(t, db, folderID)
			songID := thCreateSong(t, db, &albumWithGenre, folderID)
			thCreateGenre(t, db, "jazz")
			require.NoError(t, db.Song().CreateGenreConnections(ctx, []repos.SongGenreConnection{{SongID: songID, Genre: "jazz"}}))

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				Genres:         []string{"jazz"},
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, albumWithGenre)
			assert.NotContains(t, ids, albumWithoutGenre)
		})

		t.Run("sort by rating requires annotations and user", func(t *testing.T) {
			_, err := repo.FindAll(ctx, repos.FindAlbumParams{SortBy: repos.FindAlbumSortByRating}, repos.IncludeAlbumInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("sort by rating", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			lowRated := thCreateAlbum(t, db, folderID)
			highRated := thCreateAlbum(t, db, folderID)
			require.NoError(t, repo.SetRating(ctx, user, lowRated, 1))
			require.NoError(t, repo.SetRating(ctx, user, highRated, 5))

			include := repos.IncludeAlbumInfo{Annotations: true, User: user}
			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByRating,
				MusicFolderIDs: []int{folderID},
			}, include)
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			require.Contains(t, ids, lowRated)
			require.Contains(t, ids, highRated)
			assert.Less(t, indexOf(ids, lowRated), indexOf(ids, highRated))
		})

		t.Run("sort by starred requires annotations and user", func(t *testing.T) {
			_, err := repo.FindAll(ctx, repos.FindAlbumParams{SortBy: repos.FindAlbumSortByStarred}, repos.IncludeAlbumInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("sort by starred returns only starred albums", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			starred := thCreateAlbum(t, db, folderID)
			unstarred := thCreateAlbum(t, db, folderID)
			require.NoError(t, repo.Star(ctx, user, starred))
			require.NoError(t, repo.UnStar(ctx, user, unstarred))

			include := repos.IncludeAlbumInfo{Annotations: true, User: user}
			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByStarred,
				MusicFolderIDs: []int{folderID},
			}, include)
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, starred)
			assert.NotContains(t, ids, unstarred)
		})

		t.Run("sort random returns all albums", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			a1 := thCreateAlbum(t, db, folderID)
			a2 := thCreateAlbum(t, db, folderID)

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortRandom,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, a1)
			assert.Contains(t, ids, a2)
		})

		t.Run("sort random with seed is stable", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			thCreateAlbum(t, db, folderID)
			thCreateAlbum(t, db, folderID)
			thCreateAlbum(t, db, folderID)
			seed := "stable-seed-xyz"

			results1, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortRandom,
				RandomSeed:     &seed,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all 1: %v", err)

			results2, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortRandom,
				RandomSeed:     &seed,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all 2: %v", err)

			ids1 := util.Map(results1, func(a *repos.CompleteAlbum) string { return a.ID })
			ids2 := util.Map(results2, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Equal(t, ids1, ids2)
		})

		t.Run("sort by original date filters albums without original date", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date := repos.NewDate(2020, nil, nil)
			withDate, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "WithOriginalDate",
				OriginalDate:  &date,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)
			withoutDate := thCreateAlbum(t, db, folderID)

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByOriginalDate,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, withDate)
			assert.NotContains(t, ids, withoutDate)
		})

		t.Run("sort by release date filters albums without release date", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date := repos.NewDate(2021, nil, nil)
			withDate, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "WithReleaseDate",
				ReleaseDate:   &date,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)
			withoutDate := thCreateAlbum(t, db, folderID)

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByReleaseDate,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, withDate)
			assert.NotContains(t, ids, withoutDate)
		})

		t.Run("sort by frequent requires play info and user", func(t *testing.T) {
			_, err := repo.FindAll(ctx, repos.FindAlbumParams{SortBy: repos.FindAlbumSortByFrequent}, repos.IncludeAlbumInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("sort by frequent orders by play count descending", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			morePlayedAlbum := thCreateAlbum(t, db, folderID)
			lessPlayedAlbum := thCreateAlbum(t, db, folderID)
			song1 := thCreateSong(t, db, &morePlayedAlbum, folderID)
			song2 := thCreateSong(t, db, &lessPlayedAlbum, folderID)

			scrobble := func(songID, albumID string, t time.Time) repos.CreateScrobbleParams {
				return repos.CreateScrobbleParams{
					User:         user,
					SongID:       songID,
					AlbumID:      &albumID,
					Time:         t,
					SongDuration: repos.NewDurationMS(300000),
					Duration:     repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true},
					NowPlaying:   false,
				}
			}
			base := time.Now().Add(-10 * time.Hour)
			require.NoError(t, db.Scrobble().CreateMultiple(ctx, []repos.CreateScrobbleParams{
				scrobble(song1, morePlayedAlbum, base.Add(-2*time.Hour)),
				scrobble(song1, morePlayedAlbum, base.Add(-3*time.Hour)),
				scrobble(song2, lessPlayedAlbum, base.Add(-4*time.Hour)),
			}))

			include := repos.IncludeAlbumInfo{PlayInfo: true, User: user}
			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByFrequent,
				MusicFolderIDs: []int{folderID},
			}, include)
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			require.Contains(t, ids, morePlayedAlbum)
			require.Contains(t, ids, lessPlayedAlbum)
			assert.Less(t, indexOf(ids, morePlayedAlbum), indexOf(ids, lessPlayedAlbum))
		})

		t.Run("sort by recent requires play info and user", func(t *testing.T) {
			_, err := repo.FindAll(ctx, repos.FindAlbumParams{SortBy: repos.FindAlbumSortByRecent}, repos.IncludeAlbumInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("sort by recent orders by last played descending", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			olderPlayed := thCreateAlbum(t, db, folderID)
			newerPlayed := thCreateAlbum(t, db, folderID)
			song1 := thCreateSong(t, db, &olderPlayed, folderID)
			song2 := thCreateSong(t, db, &newerPlayed, folderID)

			scrobble := func(songID, albumID string, t time.Time) repos.CreateScrobbleParams {
				return repos.CreateScrobbleParams{
					User:         user,
					SongID:       songID,
					AlbumID:      &albumID,
					Time:         t,
					SongDuration: repos.NewDurationMS(300000),
					Duration:     repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true},
					NowPlaying:   false,
				}
			}
			now := time.Now()
			require.NoError(t, db.Scrobble().CreateMultiple(ctx, []repos.CreateScrobbleParams{
				scrobble(song1, olderPlayed, now.Add(-2*time.Hour)),
				scrobble(song2, newerPlayed, now.Add(-1*time.Hour)),
			}))

			include := repos.IncludeAlbumInfo{PlayInfo: true, User: user}
			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByRecent,
				MusicFolderIDs: []int{folderID},
			}, include)
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			require.Contains(t, ids, olderPlayed)
			require.Contains(t, ids, newerPlayed)
			assert.Less(t, indexOf(ids, newerPlayed), indexOf(ids, olderPlayed))
		})

		t.Run("filter by from year", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date2000 := repos.NewDate(2000, nil, nil)
			date2010 := repos.NewDate(2010, nil, nil)
			album2000, err := repo.Create(ctx, repos.CreateAlbumParams{Name: "Album2000", OriginalDate: &date2000, MusicFolderID: folderID})
			require.NoError(t, err)
			album2010, err := repo.Create(ctx, repos.CreateAlbumParams{Name: "Album2010", OriginalDate: &date2010, MusicFolderID: folderID})
			require.NoError(t, err)
			fromYear := 2005

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				FromYear:       &fromYear,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.NotContains(t, ids, album2000)
			assert.Contains(t, ids, album2010)
		})

		t.Run("filter by to year", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date2000 := repos.NewDate(2000, nil, nil)
			date2010 := repos.NewDate(2010, nil, nil)
			album2000, err := repo.Create(ctx, repos.CreateAlbumParams{Name: "Album2000b", OriginalDate: &date2000, MusicFolderID: folderID})
			require.NoError(t, err)
			album2010, err := repo.Create(ctx, repos.CreateAlbumParams{Name: "Album2010b", OriginalDate: &date2010, MusicFolderID: folderID})
			require.NoError(t, err)
			toYear := 2005

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				ToYear:         &toYear,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, album2000)
			assert.NotContains(t, ids, album2010)
		})

		t.Run("pagination limit", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			thCreateAlbum(t, db, folderID)
			thCreateAlbum(t, db, folderID)
			thCreateAlbum(t, db, folderID)
			limit := 2

			results, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				MusicFolderIDs: []int{folderID},
				Paginate:       repos.Paginate{Limit: &limit},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			assert.Len(t, results, 2)
		})

		t.Run("pagination offset", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			thCreateAlbum(t, db, folderID)
			thCreateAlbum(t, db, folderID)
			thCreateAlbum(t, db, folderID)

			allResults, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all no offset: %v", err)

			offsetResults, err := repo.FindAll(ctx, repos.FindAlbumParams{
				SortBy:         repos.FindAlbumSortByName,
				MusicFolderIDs: []int{folderID},
				Paginate:       repos.Paginate{Offset: 1},
			}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find all with offset: %v", err)
			assert.Len(t, offsetResults, len(allResults)-1)
			assert.Equal(t, allResults[1].ID, offsetResults[0].ID)
		})
	})

	t.Run("FindBySearch", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		id, err := repo.Create(ctx, repos.CreateAlbumParams{
			Name:          "SearchableAlbumXYZ",
			MusicFolderID: folderID,
		})
		require.NoError(t, err)

		t.Run("finds album by name", func(t *testing.T) {
			results, err := repo.FindBySearch(ctx, "SearchableAlbumXYZ", []int{folderID}, repos.Paginate{}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find by search: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, id)
		})

		t.Run("does not find album outside music folder", func(t *testing.T) {
			otherFolder := thCreateMusicFolder(t, db, user)
			results, err := repo.FindBySearch(ctx, "SearchableAlbumXYZ", []int{otherFolder}, repos.Paginate{}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "find by search: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.NotContains(t, ids, id)
		})
	})

	t.Run("Star and UnStar", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)

		t.Run("star album", func(t *testing.T) {
			err := repo.Star(ctx, user, albumID)
			require.NoErrorf(t, err, "star: %v", err)
			assert.True(t, thExists(t, db, "album_stars", map[string]any{"album_id": albumID, "user_name": user}))
		})

		t.Run("star is idempotent", func(t *testing.T) {
			err := repo.Star(ctx, user, albumID)
			assert.NoErrorf(t, err, "re-star should not fail: %v", err)
		})

		t.Run("unstar album", func(t *testing.T) {
			err := repo.UnStar(ctx, user, albumID)
			require.NoErrorf(t, err, "unstar: %v", err)
			assert.False(t, thExists(t, db, "album_stars", map[string]any{"album_id": albumID, "user_name": user}))
		})

		t.Run("star for one user does not affect another", func(t *testing.T) {
			require.NoError(t, repo.Star(ctx, user, albumID))
			assert.False(t, thExists(t, db, "album_stars", map[string]any{"album_id": albumID, "user_name": user2}))
		})
	})

	t.Run("FindStarred", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)
		_ = repo.UnStar(ctx, user, albumID)

		t.Run("not returned when not starred", func(t *testing.T) {
			results, err := repo.FindStarred(ctx, []int{folderID}, repos.Paginate{}, repos.IncludeAlbumInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find starred: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.NotContains(t, ids, albumID)
		})

		t.Run("returned when starred", func(t *testing.T) {
			require.NoError(t, repo.Star(ctx, user, albumID))
			results, err := repo.FindStarred(ctx, []int{folderID}, repos.Paginate{}, repos.IncludeAlbumInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find starred: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, albumID)
		})

		t.Run("requires include.Annotations and User", func(t *testing.T) {
			_, err := repo.FindStarred(ctx, nil, repos.Paginate{}, repos.IncludeAlbumInfoBare())
			assert.Error(t, err)
		})
	})

	t.Run("SetRating and RemoveRating", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)

		t.Run("set rating", func(t *testing.T) {
			err := repo.SetRating(ctx, user, albumID, 4)
			require.NoErrorf(t, err, "set rating: %v", err)
			assert.True(t, thExists(t, db, "album_ratings", map[string]any{"album_id": albumID, "user_name": user, "rating": 4}))
		})

		t.Run("update existing rating", func(t *testing.T) {
			err := repo.SetRating(ctx, user, albumID, 2)
			require.NoErrorf(t, err, "update rating: %v", err)
			assert.True(t, thExists(t, db, "album_ratings", map[string]any{"album_id": albumID, "user_name": user, "rating": 2}))
			assert.Equal(t, 1, thCountWhere(t, db, "album_ratings", "album_id = '"+albumID+"' AND user_name = '"+user+"'"))
		})

		t.Run("rating for one user does not affect another", func(t *testing.T) {
			require.NoError(t, repo.SetRating(ctx, user, albumID, 5))
			assert.False(t, thExists(t, db, "album_ratings", map[string]any{"album_id": albumID, "user_name": user2}))
		})

		t.Run("remove rating", func(t *testing.T) {
			err := repo.RemoveRating(ctx, user, albumID)
			require.NoErrorf(t, err, "remove rating: %v", err)
			assert.False(t, thExists(t, db, "album_ratings", map[string]any{"album_id": albumID, "user_name": user}))
		})
	})

	t.Run("GetTracks", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)
		songID := thCreateSong(t, db, &albumID, folderID)

		t.Run("returns songs for album", func(t *testing.T) {
			songs, err := repo.GetTracks(ctx, albumID, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "get tracks: %v", err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("returns empty for album with no songs", func(t *testing.T) {
			emptyAlbum := thCreateAlbum(t, db, folderID)
			songs, err := repo.GetTracks(ctx, emptyAlbum, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "get tracks: %v", err)
			assert.Empty(t, songs)
		})
	})

	t.Run("GetInfo and SetInfo", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)

		t.Run("not found for unknown album", func(t *testing.T) {
			_, err := repo.GetInfo(ctx, "al_doesnotexist", user)
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})

		t.Run("set and get info", func(t *testing.T) {
			desc := "A great album"
			url := "https://last.fm/album"
			err := repo.SetInfo(ctx, albumID, repos.SetAlbumInfo{
				Description: &desc,
				LastFMURL:   &url,
			})
			require.NoErrorf(t, err, "set info: %v", err)

			info, err := repo.GetInfo(ctx, albumID, user)
			require.NoErrorf(t, err, "get info: %v", err)
			require.NotNil(t, info)
			assert.Equal(t, albumID, info.AlbumID)
			require.NotNil(t, info.Description)
			assert.Equal(t, desc, *info.Description)
			require.NotNil(t, info.LastFMURL)
			assert.Equal(t, url, *info.LastFMURL)
		})

		t.Run("not accessible by user without music folder access", func(t *testing.T) {
			isolatedFolder := thCreateMusicFolder(t, db)
			isolatedAlbum := thCreateAlbum(t, db, isolatedFolder)
			_, err := repo.GetInfo(ctx, isolatedAlbum, user)
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})
	})

	t.Run("GetAllArtistConnections", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)
		artistID := thCreateArtist(t, db)
		require.NoError(t, repo.CreateArtistConnections(ctx, []repos.AlbumArtistConnection{
			{AlbumID: albumID, ArtistID: artistID, Index: 0},
		}))

		connections, err := repo.GetAllArtistConnections(ctx)
		require.NoErrorf(t, err, "get all artist connections: %v", err)
		albumIDs := util.Map(connections, func(c repos.AlbumArtistConnection) string { return c.AlbumID })
		assert.Contains(t, albumIDs, albumID)
	})

	t.Run("CreateArtistConnections", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)
		artistID := thCreateArtist(t, db)

		err := repo.CreateArtistConnections(ctx, []repos.AlbumArtistConnection{
			{AlbumID: albumID, ArtistID: artistID, Index: 0},
		})
		require.NoErrorf(t, err, "create artist connections: %v", err)
		assert.True(t, thExists(t, db, "album_artist", map[string]any{"album_id": albumID, "artist_id": artistID}))
	})

	t.Run("RemoveAllArtistConnections", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)
		artistID := thCreateArtist(t, db)
		require.NoError(t, repo.CreateArtistConnections(ctx, []repos.AlbumArtistConnection{
			{AlbumID: albumID, ArtistID: artistID, Index: 0},
		}))
		require.True(t, thExists(t, db, "album_artist", map[string]any{"album_id": albumID, "artist_id": artistID}))

		err := repo.RemoveAllArtistConnections(ctx)
		require.NoErrorf(t, err, "remove all artist connections: %v", err)
		assert.False(t, thExists(t, db, "album_artist", map[string]any{"album_id": albumID, "artist_id": artistID}))
	})

	t.Run("GetAlternateVersions", func(t *testing.T) {
		t.Run("finds alternate by matching music brainz id", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			mbid := "alt-versions-mbid-" + crossonic.GenIDAlbum()
			a1, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "Album Version 1",
				MusicBrainzID: &mbid,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)
			a2, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "Album Version 2",
				MusicBrainzID: &mbid,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)

			results, err := repo.GetAlternateVersions(ctx, a1, []int{folderID}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get alternate versions: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, a2)
			assert.NotContains(t, ids, a1)
		})

		t.Run("finds alternate by matching name and shared artist", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			artistID := thCreateArtist(t, db)
			a1 := thCreateAlbum(t, db, folderID)
			a2 := thCreateAlbum(t, db, folderID)
			require.NoError(t, repo.Update(ctx, a1, repos.UpdateAlbumParams{
				Name:        repos.NewOptionalFull("SharedNameAlbum"),
				ArtistNames: repos.NewOptionalFull([]string{}),
			}))
			require.NoError(t, repo.Update(ctx, a2, repos.UpdateAlbumParams{
				Name:        repos.NewOptionalFull("SharedNameAlbum"),
				ArtistNames: repos.NewOptionalFull([]string{}),
			}))
			_, err := db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING", a1, artistID)
			require.NoError(t, err)
			_, err = db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING", a2, artistID)
			require.NoError(t, err)

			results, err := repo.GetAlternateVersions(ctx, a1, []int{folderID}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get alternate versions: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, a2)
			assert.NotContains(t, ids, a1)
		})

		t.Run("does not return results from other music folders", func(t *testing.T) {
			folderA := thCreateMusicFolder(t, db, user)
			folderB := thCreateMusicFolder(t, db, user)
			mbid := "alt-folder-mbid-" + crossonic.GenIDAlbum()
			a1, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "Folder Alt 1",
				MusicBrainzID: &mbid,
				MusicFolderID: folderA,
			})
			require.NoError(t, err)
			a2, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "Folder Alt 2",
				MusicBrainzID: &mbid,
				MusicFolderID: folderB,
			})
			require.NoError(t, err)

			results, err := repo.GetAlternateVersions(ctx, a1, []int{folderA}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get alternate versions: %v", err)
			ids := util.Map(results, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.NotContains(t, ids, a2)
		})
	})

	t.Run("MigrateAnnotations", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		oldAlbum := thCreateAlbum(t, db, folderID)
		newAlbum := thCreateAlbum(t, db, folderID)

		require.NoError(t, repo.Star(ctx, user, oldAlbum))
		require.NoError(t, repo.SetRating(ctx, user, oldAlbum, 3))

		err := repo.MigrateAnnotations(ctx, oldAlbum, newAlbum)
		require.NoErrorf(t, err, "migrate annotations: %v", err)

		assert.True(t, thExists(t, db, "album_stars", map[string]any{"album_id": newAlbum, "user_name": user}))
		assert.True(t, thExists(t, db, "album_ratings", map[string]any{"album_id": newAlbum, "user_name": user, "rating": 3}))
	})

	t.Run("FindAlbumIDsToMigrate", func(t *testing.T) {
		t.Run("finds old album to migrate to newer album with same mbid", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			mbid := "to-migrate-mbid-" + crossonic.GenIDAlbum()
			oldAlbum, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "Old MigrateTest",
				MusicBrainzID: &mbid,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)

			scanStartTime := time.Now().Add(-time.Millisecond)

			newAlbum, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "New MigrateTest",
				MusicBrainzID: &mbid,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)

			results, err := repo.FindAlbumIDsToMigrate(ctx, scanStartTime)
			require.NoErrorf(t, err, "find album ids to migrate: %v", err)

			var found bool
			for _, r := range results {
				if r.OldID == oldAlbum && r.NewID == newAlbum {
					found = true
					break
				}
			}
			assert.True(t, found, "expected migration pair (%s -> %s) in results", oldAlbum, newAlbum)
		})

		t.Run("does not return album that has songs", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			mbid := "has-songs-mbid-" + crossonic.GenIDAlbum()
			albumWithSongs, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "HasSongs",
				MusicBrainzID: &mbid,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)
			thCreateSong(t, db, &albumWithSongs, folderID)

			scanStartTime := time.Now().Add(-time.Millisecond)

			_, err = repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "HasSongsDuplicate",
				MusicBrainzID: &mbid,
				MusicFolderID: folderID,
			})
			require.NoError(t, err)

			results, err := repo.FindAlbumIDsToMigrate(ctx, scanStartTime)
			require.NoErrorf(t, err, "find album ids to migrate: %v", err)
			oldIDs := util.Map(results, func(r repos.FindAlbumIDsToMigrateResult) string { return r.OldID })
			assert.NotContains(t, oldIDs, albumWithSongs)
		})

		t.Run("does not return album without music brainz id", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumNoMBID := thCreateAlbum(t, db, folderID)

			scanStartTime := time.Now().Add(-time.Millisecond)

			_, err := repo.Create(ctx, repos.CreateAlbumParams{
				Name:          "NoMBIDDuplicate",
				MusicFolderID: folderID,
			})
			require.NoError(t, err)

			results, err := repo.FindAlbumIDsToMigrate(ctx, scanStartTime)
			require.NoErrorf(t, err, "find album ids to migrate: %v", err)
			oldIDs := util.Map(results, func(r repos.FindAlbumIDsToMigrateResult) string { return r.OldID })
			assert.NotContains(t, oldIDs, albumNoMBID)
		})
	})

	t.Run("DeleteAllWithoutMusicFolderID", func(t *testing.T) {
		t.Run("deletes albums with null music_folder_id", func(t *testing.T) {
			albumID := crossonic.GenIDAlbum()
			_, err := db.db.ExecContext(ctx,
				`INSERT INTO albums (id, name, created, updated, search_text) VALUES ($1, $2, NOW(), NOW(), '')`,
				albumID, "Orphan Album")
			require.NoError(t, err)
			require.True(t, thExists(t, db, "albums", map[string]any{"id": albumID}))

			err = repo.DeleteAllWithoutMusicFolderID(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.False(t, thExists(t, db, "albums", map[string]any{"id": albumID}))
		})

		t.Run("keeps albums with a music_folder_id", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)

			err := repo.DeleteAllWithoutMusicFolderID(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.True(t, thExists(t, db, "albums", map[string]any{"id": albumID}))
		})
	})
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

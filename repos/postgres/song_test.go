package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSongRepository(t *testing.T) {
	db, _ := thSetupDatabase(t)
	ctx := context.Background()
	repo := db.Song()

	user := thCreateUser(t, db)
	user2 := thCreateUser(t, db)

	t.Run("CreateAll", func(t *testing.T) {
		t.Run("creates songs with correct fields", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			id := crossonic.GenIDSong()
			path := "/test/create-" + id + ".mp3"
			err := repo.CreateAll(ctx, []repos.CreateSongParams{
				{
					ID:            &id,
					Path:          path,
					Title:         "My Song",
					Size:          5000,
					ContentType:   "audio/mpeg",
					Duration:      repos.NewDurationMS(210000),
					BitRate:       320,
					SamplingRate:  44100,
					ChannelCount:  2,
					MusicFolderID: folderID,
				},
			})
			require.NoErrorf(t, err, "create all: %v", err)
			assert.True(t, thExists(t, db, "songs", map[string]any{
				"id":              id,
				"path":            path,
				"title":           "My Song",
				"music_folder_id": folderID,
			}))
		})

		t.Run("creates multiple songs", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			id1 := crossonic.GenIDSong()
			id2 := crossonic.GenIDSong()
			err := repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &id1, Path: "/test/" + id1 + ".mp3", Title: "Song 1", Size: 1000, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(60000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &id2, Path: "/test/" + id2 + ".mp3", Title: "Song 2", Size: 2000, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(120000), BitRate: 256, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			})
			require.NoErrorf(t, err, "create all: %v", err)
			assert.True(t, thExists(t, db, "songs", map[string]any{"id": id1}))
			assert.True(t, thExists(t, db, "songs", map[string]any{"id": id2}))
		})
	})

	t.Run("TryUpdateAll", func(t *testing.T) {
		t.Run("updates existing songs", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			songID := thCreateSong(t, db, nil, folderID)

			count, err := repo.TryUpdateAll(ctx, []repos.UpdateSongAllParams{
				{
					ID:            songID,
					Path:          "/test/updated-" + songID + ".mp3",
					Title:         "Updated Title",
					Size:          9999,
					ContentType:   "audio/flac",
					Duration:      repos.NewDurationMS(300000),
					BitRate:       1000,
					SamplingRate:  48000,
					ChannelCount:  2,
					MusicFolderID: &folderID,
				},
			})
			require.NoErrorf(t, err, "try update all: %v", err)
			assert.Equal(t, 1, count)
			assert.True(t, thExists(t, db, "songs", map[string]any{"id": songID, "title": "Updated Title", "content_type": "audio/flac"}))
		})

		t.Run("skips non-existent songs and returns count of updated", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			existingID := thCreateSong(t, db, nil, folderID)
			fakeID := crossonic.GenIDSong()

			count, err := repo.TryUpdateAll(ctx, []repos.UpdateSongAllParams{
				{ID: existingID, Path: "/x.mp3", Title: "X", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: &folderID},
				{ID: fakeID, Path: "/y.mp3", Title: "Y", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: &folderID},
			})
			require.NoErrorf(t, err, "try update all: %v", err)
			assert.Equal(t, 1, count)
		})

		t.Run("empty input returns zero count", func(t *testing.T) {
			count, err := repo.TryUpdateAll(ctx, []repos.UpdateSongAllParams{})
			require.NoErrorf(t, err, "try update all: %v", err)
			assert.Equal(t, 0, count)
		})
	})

	t.Run("FindByID", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)

		t.Run("returns song for authorized user", func(t *testing.T) {
			s, err := repo.FindByID(ctx, songID, user, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, s)
			assert.Equal(t, songID, s.ID)
		})

		t.Run("returns not found for unauthorized user", func(t *testing.T) {
			isolatedFolder := thCreateMusicFolder(t, db)
			isolatedSong := thCreateSong(t, db, nil, isolatedFolder)
			_, err := repo.FindByID(ctx, isolatedSong, user, repos.IncludeSongInfoBare())
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})

		t.Run("includes album info", func(t *testing.T) {
			albumID := thCreateAlbum(t, db, folderID)
			songWithAlbum := thCreateSong(t, db, &albumID, folderID)
			s, err := repo.FindByID(ctx, songWithAlbum, user, repos.IncludeSongInfo{Album: true})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, s.SongAlbumInfo)
			require.NotNil(t, s.AlbumName)
		})

		t.Run("includes annotation info", func(t *testing.T) {
			require.NoError(t, repo.Star(ctx, user, songID))
			s, err := repo.FindByID(ctx, songID, user, repos.IncludeSongInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, s.SongAnnotations)
			assert.NotNil(t, s.Starred)
		})
	})

	t.Run("FindByIDs", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		id1 := thCreateSong(t, db, nil, folderID)
		id2 := thCreateSong(t, db, nil, folderID)

		t.Run("returns matching songs", func(t *testing.T) {
			songs, err := repo.FindByIDs(ctx, []string{id1, id2}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by ids: %v", err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, id1)
			assert.Contains(t, ids, id2)
		})

		t.Run("returns empty for no matching IDs", func(t *testing.T) {
			songs, err := repo.FindByIDs(ctx, []string{crossonic.GenIDSong()}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by ids: %v", err)
			assert.Empty(t, songs)
		})
	})

	t.Run("FindByPath", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := crossonic.GenIDSong()
		path := "/test/findbypath-" + songID + ".mp3"
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &songID, Path: path, Title: "Test", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
		}))

		t.Run("returns song by path", func(t *testing.T) {
			s, err := repo.FindByPath(ctx, path, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by path: %v", err)
			assert.Equal(t, songID, s.ID)
		})

		t.Run("returns not found for unknown path", func(t *testing.T) {
			_, err := repo.FindByPath(ctx, "/does/not/exist.mp3", repos.IncludeSongInfoBare())
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})
	})

	t.Run("FindAllFiltered", func(t *testing.T) {
		t.Run("returns songs in music folder", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			s1 := thCreateSong(t, db, nil, folderID)
			s2 := thCreateSong(t, db, nil, folderID)

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, s1)
			assert.Contains(t, ids, s2)
		})

		t.Run("does not return songs from other folders", func(t *testing.T) {
			folderA := thCreateMusicFolder(t, db, user)
			folderB := thCreateMusicFolder(t, db, user)
			inA := thCreateSong(t, db, nil, folderA)
			inB := thCreateSong(t, db, nil, folderB)

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				MusicFolderIDs: []int{folderA},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, inA)
			assert.NotContains(t, ids, inB)
		})

		t.Run("filter by genre", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			withGenre := thCreateSong(t, db, nil, folderID)
			withoutGenre := thCreateSong(t, db, nil, folderID)
			thCreateGenre(t, db, "rock")
			require.NoError(t, repo.CreateGenreConnections(ctx, []repos.SongGenreConnection{{SongID: withGenre, Genre: "rock"}}))

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Genres:         []string{"rock"},
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, withGenre)
			assert.NotContains(t, ids, withoutGenre)
		})

		t.Run("filter only starred", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			starred := thCreateSong(t, db, nil, folderID)
			notStarred := thCreateSong(t, db, nil, folderID)
			require.NoError(t, repo.Star(ctx, user, starred))

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				OnlyStarred:    true,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, starred)
			assert.NotContains(t, ids, notStarred)
		})

		t.Run("order by title", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			order := repos.SongOrderTitle
			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			assert.NotNil(t, results)
		})

		t.Run("filter by artist ids", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			withArtist := thCreateSong(t, db, nil, folderID)
			withoutArtist := thCreateSong(t, db, nil, folderID)
			artistID := thCreateArtist(t, db)
			require.NoError(t, repo.CreateArtistConnections(ctx, []repos.SongArtistConnection{
				{SongID: withArtist, ArtistID: artistID, Index: 0},
			}))

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				ArtistIDs:      []string{artistID},
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, withArtist)
			assert.NotContains(t, ids, withoutArtist)
		})

		t.Run("filter by album ids", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumA := thCreateAlbum(t, db, folderID)
			albumB := thCreateAlbum(t, db, folderID)
			inAlbumA := thCreateSong(t, db, &albumA, folderID)
			inAlbumB := thCreateSong(t, db, &albumB, folderID)

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				AlbumIDs:       []string{albumA},
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, inAlbumA)
			assert.NotContains(t, ids, inAlbumB)
		})

		t.Run("filter by min bpm", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			bpmLow, bpmHigh := 80, 160
			idLow, idHigh := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &idLow, Path: "/test/bpmlow-" + idLow + ".mp3", Title: "Low BPM", BPM: &bpmLow, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &idHigh, Path: "/test/bpmhigh-" + idHigh + ".mp3", Title: "High BPM", BPM: &bpmHigh, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			minBPM := 120

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				MinBPM:         &minBPM,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, idHigh)
			assert.NotContains(t, ids, idLow)
		})

		t.Run("filter by max bpm", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			bpmLow, bpmHigh := 80, 160
			idLow, idHigh := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &idLow, Path: "/test/maxbpmlow-" + idLow + ".mp3", Title: "Low BPM", BPM: &bpmLow, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &idHigh, Path: "/test/maxbpmhigh-" + idHigh + ".mp3", Title: "High BPM", BPM: &bpmHigh, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			maxBPM := 120

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				MaxBPM:         &maxBPM,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, idLow)
			assert.NotContains(t, ids, idHigh)
		})

		t.Run("filter by from year", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date2000 := repos.NewDate(2000, nil, nil)
			date2020 := repos.NewDate(2020, nil, nil)
			id2000, id2020 := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &id2000, Path: "/test/fromyr2000-" + id2000 + ".mp3", Title: "Song 2000", ReleaseDate: &date2000, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &id2020, Path: "/test/fromyr2020-" + id2020 + ".mp3", Title: "Song 2020", ReleaseDate: &date2020, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			fromYear := 2010

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				FromYear:       &fromYear,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, id2020)
			assert.NotContains(t, ids, id2000)
		})

		t.Run("filter by to year", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date2000 := repos.NewDate(2000, nil, nil)
			date2020 := repos.NewDate(2020, nil, nil)
			id2000, id2020 := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &id2000, Path: "/test/toyr2000-" + id2000 + ".mp3", Title: "Song 2000", ReleaseDate: &date2000, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &id2020, Path: "/test/toyr2020-" + id2020 + ".mp3", Title: "Song 2020", ReleaseDate: &date2020, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			toYear := 2010

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				ToYear:         &toYear,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, id2000)
			assert.NotContains(t, ids, id2020)
		})

		t.Run("only starred requires annotations and user", func(t *testing.T) {
			_, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{OnlyStarred: true}, repos.IncludeSongInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)

			_, err = repo.FindAllFiltered(ctx, repos.SongFindAllFilter{OnlyStarred: true}, repos.IncludeSongInfo{Annotations: true})
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("order by random with seed is stable", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			thCreateSong(t, db, nil, folderID)
			thCreateSong(t, db, nil, folderID)
			thCreateSong(t, db, nil, folderID)
			order := repos.SongOrderRandom
			seed := "stable-test-seed"

			results1, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				RandomSeed:     &seed,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered 1: %v", err)

			results2, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				RandomSeed:     &seed,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered 2: %v", err)

			ids1 := util.Map(results1, func(s *repos.CompleteSong) string { return s.ID })
			ids2 := util.Map(results2, func(s *repos.CompleteSong) string { return s.ID })
			assert.Equal(t, ids1, ids2)
		})

		t.Run("order by bpm ascending", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			bpmLow, bpmHigh := 80, 160
			idLow, idHigh := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &idLow, Path: "/test/orderbpm-low-" + idLow + ".mp3", Title: "Low BPM Song", BPM: &bpmLow, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &idHigh, Path: "/test/orderbpm-high-" + idHigh + ".mp3", Title: "High BPM Song", BPM: &bpmHigh, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			order := repos.SongOrderBPM

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ids, idLow)
			require.Contains(t, ids, idHigh)
			assert.Less(t, indexOf(ids, idLow), indexOf(ids, idHigh))
		})

		t.Run("order by original date ascending", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date2000 := repos.NewDate(2000, nil, nil)
			date2020 := repos.NewDate(2020, nil, nil)
			idOlder, idNewer := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &idOlder, Path: "/test/origdate-old-" + idOlder + ".mp3", Title: "Older", OriginalDate: &date2000, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &idNewer, Path: "/test/origdate-new-" + idNewer + ".mp3", Title: "Newer", OriginalDate: &date2020, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			order := repos.SongOrderOriginalDate

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ids, idOlder)
			require.Contains(t, ids, idNewer)
			assert.Less(t, indexOf(ids, idOlder), indexOf(ids, idNewer))
		})

		t.Run("order by release date ascending", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			date2000 := repos.NewDate(2000, nil, nil)
			date2020 := repos.NewDate(2020, nil, nil)
			idOlder, idNewer := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &idOlder, Path: "/test/reldate-old-" + idOlder + ".mp3", Title: "Older Release", ReleaseDate: &date2000, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &idNewer, Path: "/test/reldate-new-" + idNewer + ".mp3", Title: "Newer Release", ReleaseDate: &date2020, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			order := repos.SongOrderReleaseDate

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ids, idOlder)
			require.Contains(t, ids, idNewer)
			assert.Less(t, indexOf(ids, idOlder), indexOf(ids, idNewer))
		})

		t.Run("order by added", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			idFirst := thCreateSong(t, db, nil, folderID)
			idSecond := thCreateSong(t, db, nil, folderID)
			order := repos.SongOrderAdded

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ids, idFirst)
			require.Contains(t, ids, idSecond)
			assert.LessOrEqual(t, indexOf(ids, idFirst), indexOf(ids, idSecond))
		})

		t.Run("order by play count", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			lessPlayed := thCreateSong(t, db, nil, folderID)
			morePlayed := thCreateSong(t, db, nil, folderID)
			base := time.Now().Add(-10 * time.Hour)
			require.NoError(t, db.Scrobble().CreateMultiple(ctx, []repos.CreateScrobbleParams{
				{User: user, SongID: morePlayed, Time: base.Add(-1 * time.Hour), SongDuration: repos.NewDurationMS(300000), Duration: repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true}, NowPlaying: false},
				{User: user, SongID: morePlayed, Time: base.Add(-2 * time.Hour), SongDuration: repos.NewDurationMS(300000), Duration: repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true}, NowPlaying: false},
				{User: user, SongID: lessPlayed, Time: base.Add(-3 * time.Hour), SongDuration: repos.NewDurationMS(300000), Duration: repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true}, NowPlaying: false},
			}))
			order := repos.SongOrderPlayCount

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfo{PlayInfo: true, User: user})
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ids, lessPlayed)
			require.Contains(t, ids, morePlayed)
			assert.Less(t, indexOf(ids, lessPlayed), indexOf(ids, morePlayed))
		})

		t.Run("order by last played", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			playedOlder := thCreateSong(t, db, nil, folderID)
			playedNewer := thCreateSong(t, db, nil, folderID)
			now := time.Now()
			require.NoError(t, db.Scrobble().CreateMultiple(ctx, []repos.CreateScrobbleParams{
				{User: user, SongID: playedOlder, Time: now.Add(-2 * time.Hour), SongDuration: repos.NewDurationMS(300000), Duration: repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true}, NowPlaying: false},
				{User: user, SongID: playedNewer, Time: now.Add(-1 * time.Hour), SongDuration: repos.NewDurationMS(300000), Duration: repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true}, NowPlaying: false},
			}))
			order := repos.SongOrderLastPlayed

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfo{PlayInfo: true, User: user})
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ids, playedOlder)
			require.Contains(t, ids, playedNewer)
			assert.Less(t, indexOf(ids, playedOlder), indexOf(ids, playedNewer))
		})

		t.Run("order by starred", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			starredFirst := thCreateSong(t, db, nil, folderID)
			starredSecond := thCreateSong(t, db, nil, folderID)
			require.NoError(t, repo.Star(ctx, user, starredFirst))
			require.NoError(t, repo.Star(ctx, user, starredSecond))
			order := repos.SongOrderStarred

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ids, starredFirst)
			require.Contains(t, ids, starredSecond)
			assert.LessOrEqual(t, indexOf(ids, starredFirst), indexOf(ids, starredSecond))
		})

		t.Run("order by last played requires play info and user", func(t *testing.T) {
			order := repos.SongOrderLastPlayed
			_, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{Order: &order}, repos.IncludeSongInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)

			_, err = repo.FindAllFiltered(ctx, repos.SongFindAllFilter{Order: &order}, repos.IncludeSongInfo{PlayInfo: true})
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("order by play count requires play info and user", func(t *testing.T) {
			order := repos.SongOrderPlayCount
			_, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{Order: &order}, repos.IncludeSongInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("order by starred requires annotations and user", func(t *testing.T) {
			order := repos.SongOrderStarred
			_, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{Order: &order}, repos.IncludeSongInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("order descending", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			idA, idZ := crossonic.GenIDSong(), crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &idA, Path: "/test/desc-a-" + idA + ".mp3", Title: "AAA Song", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
				{ID: &idZ, Path: "/test/desc-z-" + idZ + ".mp3", Title: "ZZZ Song", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))
			order := repos.SongOrderTitle

			ascResults, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				OrderDesc:      false,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered asc: %v", err)

			descResults, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				OrderDesc:      true,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered desc: %v", err)

			ascIDs := util.Map(ascResults, func(s *repos.CompleteSong) string { return s.ID })
			descIDs := util.Map(descResults, func(s *repos.CompleteSong) string { return s.ID })
			require.Contains(t, ascIDs, idA)
			require.Contains(t, ascIDs, idZ)
			assert.Less(t, indexOf(ascIDs, idA), indexOf(ascIDs, idZ))
			assert.Greater(t, indexOf(descIDs, idA), indexOf(descIDs, idZ))
		})

		t.Run("search returns matching songs", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			uniqueTitle := "FindSearchSongXYZ-" + crossonic.GenIDSong()
			id := crossonic.GenIDSong()
			require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
				{ID: &id, Path: "/test/search-" + id + ".mp3", Title: uniqueTitle, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			}))

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Search:         uniqueTitle,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			ids := util.Map(results, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, id)
		})

		t.Run("includes song lists", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			artistID := thCreateArtist(t, db)
			songID := thCreateSong(t, db, nil, folderID)
			thCreateGenre(t, db, "country")
			require.NoError(t, repo.CreateArtistConnections(ctx, []repos.SongArtistConnection{{SongID: songID, ArtistID: artistID, Index: 0}}))
			require.NoError(t, repo.CreateGenreConnections(ctx, []repos.SongGenreConnection{{SongID: songID, Genre: "country"}}))

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfo{Lists: true})
			require.NoErrorf(t, err, "find all filtered: %v", err)
			require.Len(t, results, 1)
			require.NotNil(t, results[0].SongLists)
			artistIDs := util.Map(results[0].Artists, func(a repos.ArtistRef) string { return a.ID })
			assert.Contains(t, artistIDs, artistID)
			assert.Contains(t, results[0].Genres, "country")
		})

		t.Run("includes play info", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			songID := thCreateSong(t, db, nil, folderID)
			require.NoError(t, db.Scrobble().CreateMultiple(ctx, []repos.CreateScrobbleParams{
				{User: user, SongID: songID, Time: time.Now().Add(-30 * time.Minute), SongDuration: repos.NewDurationMS(300000), Duration: repos.NullDurationMS{Duration: repos.NewDurationMS(300000), Valid: true}, NowPlaying: false},
			}))

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfo{PlayInfo: true, User: user})
			require.NoErrorf(t, err, "find all filtered: %v", err)
			require.Len(t, results, 1)
			require.NotNil(t, results[0].SongPlayInfo)
			assert.Equal(t, 1, results[0].PlayCount)
			assert.NotNil(t, results[0].LastPlayed)
		})

		t.Run("pagination limit", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			thCreateSong(t, db, nil, folderID)
			thCreateSong(t, db, nil, folderID)
			thCreateSong(t, db, nil, folderID)
			limit := 2

			results, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				MusicFolderIDs: []int{folderID},
				Paginate:       repos.Paginate{Limit: &limit},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered: %v", err)
			assert.Len(t, results, 2)
		})

		t.Run("pagination offset", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			thCreateSong(t, db, nil, folderID)
			thCreateSong(t, db, nil, folderID)
			thCreateSong(t, db, nil, folderID)
			order := repos.SongOrderAdded

			allResults, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered no offset: %v", err)

			offsetResults, err := repo.FindAllFiltered(ctx, repos.SongFindAllFilter{
				Order:          &order,
				MusicFolderIDs: []int{folderID},
				Paginate:       repos.Paginate{Offset: 1},
			}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all filtered with offset: %v", err)
			assert.Len(t, offsetResults, len(allResults)-1)
			assert.Equal(t, allResults[1].ID, offsetResults[0].ID)
		})
	})

	t.Run("FindPaths", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		id := crossonic.GenIDSong()
		path := "/test/findpaths-" + id + ".mp3"
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &id, Path: path, Title: "Test", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
		}))

		paths, err := repo.FindPaths(ctx, time.Now().Add(time.Hour), repos.Paginate{})
		require.NoErrorf(t, err, "find paths: %v", err)
		assert.Contains(t, paths, path)
	})

	t.Run("DeleteByPaths", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		id1 := crossonic.GenIDSong()
		id2 := crossonic.GenIDSong()
		path1 := "/test/del-" + id1 + ".mp3"
		path2 := "/test/del-" + id2 + ".mp3"
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &id1, Path: path1, Title: "Del1", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			{ID: &id2, Path: path2, Title: "Del2", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
		}))

		t.Run("deletes matching songs", func(t *testing.T) {
			err := repo.DeleteByPaths(ctx, []string{path1})
			require.NoErrorf(t, err, "delete by paths: %v", err)
			assert.False(t, thExists(t, db, "songs", map[string]any{"id": id1}))
			assert.True(t, thExists(t, db, "songs", map[string]any{"id": id2}))
		})
	})

	t.Run("Star and UnStar", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)

		t.Run("star song", func(t *testing.T) {
			err := repo.Star(ctx, user, songID)
			require.NoErrorf(t, err, "star: %v", err)
			assert.True(t, thExists(t, db, "song_stars", map[string]any{"song_id": songID, "user_name": user}))
		})

		t.Run("star is idempotent", func(t *testing.T) {
			err := repo.Star(ctx, user, songID)
			assert.NoErrorf(t, err, "re-star should not fail: %v", err)
		})

		t.Run("unstar song", func(t *testing.T) {
			err := repo.UnStar(ctx, user, songID)
			require.NoErrorf(t, err, "unstar: %v", err)
			assert.False(t, thExists(t, db, "song_stars", map[string]any{"song_id": songID, "user_name": user}))
		})

		t.Run("star for one user does not affect another", func(t *testing.T) {
			require.NoError(t, repo.Star(ctx, user, songID))
			assert.False(t, thExists(t, db, "song_stars", map[string]any{"song_id": songID, "user_name": user2}))
		})
	})

	t.Run("StarMultiple and UnStarMultiple", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		id1 := thCreateSong(t, db, nil, folderID)
		id2 := thCreateSong(t, db, nil, folderID)
		ids := []string{id1, id2}

		t.Run("stars multiple songs", func(t *testing.T) {
			count, err := repo.StarMultiple(ctx, user, ids)
			require.NoErrorf(t, err, "star multiple: %v", err)
			assert.Equal(t, 2, count)
			assert.True(t, thExists(t, db, "song_stars", map[string]any{"song_id": id1, "user_name": user}))
			assert.True(t, thExists(t, db, "song_stars", map[string]any{"song_id": id2, "user_name": user}))
		})

		t.Run("StarMultiple is idempotent", func(t *testing.T) {
			count, err := repo.StarMultiple(ctx, user, ids)
			require.NoErrorf(t, err, "re-star multiple: %v", err)
			assert.Equal(t, 0, count, "already starred songs should not be counted")
		})

		t.Run("unstars multiple songs", func(t *testing.T) {
			count, err := repo.UnStarMultiple(ctx, user, ids)
			require.NoErrorf(t, err, "unstar multiple: %v", err)
			assert.Equal(t, 2, count)
			assert.False(t, thExists(t, db, "song_stars", map[string]any{"song_id": id1, "user_name": user}))
			assert.False(t, thExists(t, db, "song_stars", map[string]any{"song_id": id2, "user_name": user}))
		})
	})

	t.Run("SetRating and RemoveRating", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)

		t.Run("set rating", func(t *testing.T) {
			err := repo.SetRating(ctx, user, songID, 4)
			require.NoErrorf(t, err, "set rating: %v", err)
			assert.True(t, thExists(t, db, "song_ratings", map[string]any{"song_id": songID, "user_name": user, "rating": 4}))
		})

		t.Run("update existing rating", func(t *testing.T) {
			err := repo.SetRating(ctx, user, songID, 2)
			require.NoErrorf(t, err, "update rating: %v", err)
			assert.True(t, thExists(t, db, "song_ratings", map[string]any{"song_id": songID, "user_name": user, "rating": 2}))
			assert.Equal(t, 1, thCountWhere(t, db, "song_ratings", "song_id = '"+songID+"' AND user_name = '"+user+"'"))
		})

		t.Run("rating for one user does not affect another", func(t *testing.T) {
			require.NoError(t, repo.SetRating(ctx, user, songID, 5))
			assert.False(t, thExists(t, db, "song_ratings", map[string]any{"song_id": songID, "user_name": user2}))
		})

		t.Run("remove rating", func(t *testing.T) {
			err := repo.RemoveRating(ctx, user, songID)
			require.NoErrorf(t, err, "remove rating: %v", err)
			assert.False(t, thExists(t, db, "song_ratings", map[string]any{"song_id": songID, "user_name": user}))
		})
	})

	t.Run("CreateArtistConnections and DeleteArtistConnections", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)
		artistID := thCreateArtist(t, db)

		t.Run("create connections", func(t *testing.T) {
			err := repo.CreateArtistConnections(ctx, []repos.SongArtistConnection{
				{SongID: songID, ArtistID: artistID, Index: 0},
			})
			require.NoErrorf(t, err, "create artist connections: %v", err)
			assert.True(t, thExists(t, db, "song_artist", map[string]any{"song_id": songID, "artist_id": artistID}))
		})

		t.Run("delete connections", func(t *testing.T) {
			err := repo.DeleteArtistConnections(ctx, []string{songID})
			require.NoErrorf(t, err, "delete artist connections: %v", err)
			assert.False(t, thExists(t, db, "song_artist", map[string]any{"song_id": songID}))
		})
	})

	t.Run("CreateGenreConnections and DeleteGenreConnections", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)
		thCreateGenre(t, db, "pop")

		t.Run("create connections", func(t *testing.T) {
			err := repo.CreateGenreConnections(ctx, []repos.SongGenreConnection{
				{SongID: songID, Genre: "pop"},
			})
			require.NoErrorf(t, err, "create genre connections: %v", err)
			assert.True(t, thExists(t, db, "song_genre", map[string]any{"song_id": songID, "genre_name": "pop"}))
		})

		t.Run("delete connections", func(t *testing.T) {
			err := repo.DeleteGenreConnections(ctx, []string{songID})
			require.NoErrorf(t, err, "delete genre connections: %v", err)
			assert.False(t, thExists(t, db, "song_genre", map[string]any{"song_id": songID}))
		})
	})

	t.Run("Count", func(t *testing.T) {
		before := thCount(t, db, "songs")
		folderID := thCreateMusicFolder(t, db, user)
		thCreateSong(t, db, nil, folderID)
		thCreateSong(t, db, nil, folderID)

		count, err := repo.Count(ctx)
		require.NoErrorf(t, err, "count: %v", err)
		assert.Equal(t, before+2, count)
	})

	t.Run("DeleteLastUpdatedBefore", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)

		t.Run("keeps recently updated songs", func(t *testing.T) {
			err := repo.DeleteLastUpdatedBefore(ctx, time.Now().Add(-time.Hour))
			require.NoErrorf(t, err, "delete: %v", err)
			assert.True(t, thExists(t, db, "songs", map[string]any{"id": songID}))
		})

		t.Run("deletes old songs", func(t *testing.T) {
			err := repo.DeleteLastUpdatedBefore(ctx, time.Now().Add(time.Hour))
			require.NoErrorf(t, err, "delete: %v", err)
			assert.False(t, thExists(t, db, "songs", map[string]any{"id": songID}))
		})
	})

	t.Run("GetStreamInfo", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		id := crossonic.GenIDSong()
		path := "/test/stream-" + id + ".mp3"
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &id, Path: path, Title: "Stream Test", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(180000), BitRate: 320, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
		}))

		t.Run("returns stream info for authorized user", func(t *testing.T) {
			info, err := repo.GetStreamInfo(ctx, id, user)
			require.NoErrorf(t, err, "get stream info: %v", err)
			assert.Equal(t, path, info.Path)
			assert.Equal(t, "audio/mpeg", info.ContentType)
			assert.Equal(t, 320, info.BitRate)
		})

		t.Run("returns not found for unauthorized user", func(t *testing.T) {
			isolatedFolder := thCreateMusicFolder(t, db)
			isolatedID := thCreateSong(t, db, nil, isolatedFolder)
			_, err := repo.GetStreamInfo(ctx, isolatedID, user)
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})
	})

	t.Run("FindByMusicBrainzID", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		mbid := "song-mbid-" + crossonic.GenIDSong()
		id := crossonic.GenIDSong()
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &id, Path: "/test/bymbid-" + id + ".mp3", Title: "MBID Song", MusicBrainzID: &mbid, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
		}))

		t.Run("returns songs by mbid", func(t *testing.T) {
			songs, err := repo.FindByMusicBrainzID(ctx, mbid, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by music brainz id: %v", err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, id)
		})

		t.Run("returns empty for unknown mbid", func(t *testing.T) {
			songs, err := repo.FindByMusicBrainzID(ctx, "nonexistent-song-mbid-xyz", repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by music brainz id: %v", err)
			assert.Empty(t, songs)
		})
	})

	t.Run("FindByTitle", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		folderID2 := thCreateMusicFolder(t, db, user)
		folderID3 := thCreateMusicFolder(t, db, user2)
		songTitle := "UniqueTitle-" + crossonic.GenIDSong()
		id := crossonic.GenIDSong()
		id2 := crossonic.GenIDSong()
		id3 := crossonic.GenIDSong()
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &id, Path: "/test/bytitle-" + id + ".mp3", Title: songTitle, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			{ID: &id2, Path: "/test/bytitle-" + id2 + ".mp3", Title: songTitle, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID2},
			{ID: &id3, Path: "/test/bytitle-" + id3 + ".mp3", Title: songTitle, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID3},
		}))

		t.Run("returns accessible songs by title", func(t *testing.T) {
			songs, err := repo.FindByTitle(ctx, songTitle, user, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by title: %v", err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Equal(t, 2, len(ids))
			assert.Contains(t, ids, id)
			assert.Contains(t, ids, id2)
			assert.NotContains(t, ids, id3)
		})

		t.Run("returns empty for unknown title", func(t *testing.T) {
			songs, err := repo.FindByTitle(ctx, "no-such-title-xyz-123", user, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find by title: %v", err)
			assert.Empty(t, songs)
		})
	})

	t.Run("FindAllByPathOrMBID", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		mbid := "by-path-or-mbid-" + crossonic.GenIDSong()
		idByPath := crossonic.GenIDSong()
		pathForSearch := "/test/bypass-" + idByPath + ".mp3"
		idByMBID := crossonic.GenIDSong()
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &idByPath, Path: pathForSearch, Title: "Path Song", Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			{ID: &idByMBID, Path: "/test/bymbid2-" + idByMBID + ".mp3", Title: "MBID Song2", MusicBrainzID: &mbid, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
		}))

		t.Run("finds by path", func(t *testing.T) {
			songs, err := repo.FindAllByPathOrMBID(ctx, []string{pathForSearch}, nil, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all by path or mbid: %v", err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, idByPath)
		})

		t.Run("finds by mbid", func(t *testing.T) {
			songs, err := repo.FindAllByPathOrMBID(ctx, nil, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all by path or mbid: %v", err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, idByMBID)
		})

		t.Run("finds by path and mbid combined", func(t *testing.T) {
			songs, err := repo.FindAllByPathOrMBID(ctx, []string{pathForSearch}, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all by path or mbid: %v", err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, idByPath)
			assert.Contains(t, ids, idByMBID)
		})

		t.Run("returns empty for no matches", func(t *testing.T) {
			songs, err := repo.FindAllByPathOrMBID(ctx, []string{"/no/such/path.mp3"}, []string{"no-such-mbid-xyz"}, repos.IncludeSongInfoBare())
			require.NoErrorf(t, err, "find all by path or mbid: %v", err)
			assert.Empty(t, songs)
		})
	})

	t.Run("FindNonExistentIDs", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		existingID := thCreateSong(t, db, nil, folderID)
		fakeID1 := crossonic.GenIDSong()
		fakeID2 := crossonic.GenIDSong()

		t.Run("returns non-existent IDs", func(t *testing.T) {
			result, err := repo.FindNonExistentIDs(ctx, []string{existingID, fakeID1, fakeID2})
			require.NoErrorf(t, err, "find non-existent IDs: %v", err)
			assert.Contains(t, result, fakeID1)
			assert.Contains(t, result, fakeID2)
			assert.NotContains(t, result, existingID)
		})

		t.Run("returns empty when all IDs exist", func(t *testing.T) {
			result, err := repo.FindNonExistentIDs(ctx, []string{existingID})
			require.NoErrorf(t, err, "find non-existent IDs: %v", err)
			assert.Empty(t, result)
		})
	})

	t.Run("GetMedianReplayGain", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		rg2, rg4, rg6 := 2.0, 4.0, 6.0
		id1, id2, id3 := crossonic.GenIDSong(), crossonic.GenIDSong(), crossonic.GenIDSong()
		require.NoError(t, repo.CreateAll(ctx, []repos.CreateSongParams{
			{ID: &id1, Path: "/test/rg-" + id1 + ".mp3", Title: "RG Song 1", ReplayGain: &rg2, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			{ID: &id2, Path: "/test/rg-" + id2 + ".mp3", Title: "RG Song 2", ReplayGain: &rg4, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
			{ID: &id3, Path: "/test/rg-" + id3 + ".mp3", Title: "RG Song 3", ReplayGain: &rg6, Size: 1, ContentType: "audio/mpeg", Duration: repos.NewDurationMS(1000), BitRate: 128, SamplingRate: 44100, ChannelCount: 2, MusicFolderID: folderID},
		}))

		gain, err := repo.GetMedianReplayGain(ctx)
		require.NoErrorf(t, err, "get median replay gain: %v", err)
		assert.InDelta(t, 4.0, gain, 1e-9)
	})

	t.Run("DeleteAllWithoutMusicFolderID", func(t *testing.T) {
		t.Run("deletes songs without music folder id", func(t *testing.T) {
			songID := crossonic.GenIDSong()
			_, err := db.db.ExecContext(ctx,
				`INSERT INTO songs (id, path, title, size, content_type, duration_ms, bit_rate, sampling_rate, channel_count, created, updated, search_text) VALUES ($1, $2, 'Orphan Song', 1, 'audio/mpeg', 180000, 128, 44100, 2, NOW(), NOW(), '')`,
				songID, "/test/orphan-"+songID+".mp3")
			require.NoError(t, err)
			require.True(t, thExists(t, db, "songs", map[string]any{"id": songID}))

			err = repo.DeleteAllWithoutMusicFolderID(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.False(t, thExists(t, db, "songs", map[string]any{"id": songID}))
		})

		t.Run("keeps songs with music folder id", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			songID := thCreateSong(t, db, nil, folderID)

			err := repo.DeleteAllWithoutMusicFolderID(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.True(t, thExists(t, db, "songs", map[string]any{"id": songID}))
		})
	})

	t.Run("CreateArtistConnections is idempotent", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)
		artistID := thCreateArtist(t, db)

		err := repo.CreateArtistConnections(ctx, []repos.SongArtistConnection{{SongID: songID, ArtistID: artistID, Index: 0}})
		require.NoErrorf(t, err, "first create: %v", err)

		err = repo.CreateArtistConnections(ctx, []repos.SongArtistConnection{{SongID: songID, ArtistID: artistID, Index: 0}})
		assert.NoErrorf(t, err, "second create should not fail: %v", err)
		assert.Equal(t, 1, thCountWhere(t, db, "song_artist", "song_id = '"+songID+"' AND artist_id = '"+artistID+"'"))
	})

	t.Run("CreateGenreConnections is idempotent", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		songID := thCreateSong(t, db, nil, folderID)
		thCreateGenre(t, db, "classical")

		err := repo.CreateGenreConnections(ctx, []repos.SongGenreConnection{{SongID: songID, Genre: "classical"}})
		require.NoErrorf(t, err, "first create: %v", err)

		err = repo.CreateGenreConnections(ctx, []repos.SongGenreConnection{{SongID: songID, Genre: "classical"}})
		assert.NoErrorf(t, err, "second create should not fail: %v", err)
		assert.Equal(t, 1, thCountWhere(t, db, "song_genre", "song_id = '"+songID+"' AND genre_name = 'classical'"))
	})

	t.Run("FindNotUploadedLBFeedback", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)

		t.Run("includes starred song with no LB status when MBID not in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{"other-mbid"}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("excludes starred song with no LB status when MBID is already in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.NotContains(t, ids, songID)
		})

		t.Run("excludes unstarred song with no LB status", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{"other-mbid"}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.NotContains(t, ids, songID)
		})

		t.Run("includes unstarred song with uploaded=false when MBID is in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: false},
			}, false))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("includes unstarred song with uploaded=false when remoteMBID is in lbLovedMBIDs", func(t *testing.T) {
			remoteMBID := "remote-mbid-" + crossonic.GenIDSong()
			songID := thCreateSong(t, db, nil, folderID)
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, RemoteMBID: &remoteMBID, Uploaded: false},
			}, true))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{remoteMBID}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("includes starred song with uploaded=false when MBID not in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: false},
			}, false))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{"other-mbid"}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("excludes song with uploaded=true", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: true},
			}, false))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{"other-mbid"}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.NotContains(t, ids, songID)
		})

		t.Run("works with empty lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("does not return songs starred by other users", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user2, songID))

			songs, err := repo.FindNotUploadedLBFeedback(ctx, user, []string{"other-mbid"}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.NotContains(t, ids, songID)
		})
	})

	t.Run("FindLocalOutdatedFeedbackByLB", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)

		t.Run("includes unstarred song with no LB status when MBID is in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)

			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("excludes starred song with no LB status when MBID not in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))

			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{"other-mbid"}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.NotContains(t, ids, songID)
		})

		t.Run("includes unstarred song with uploaded=true when MBID is in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: true},
			}, false))

			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("includes unstarred song with uploaded=true when remoteMBID is in lbLovedMBIDs", func(t *testing.T) {
			remoteMBID := "remote-mbid-" + crossonic.GenIDSong()
			songID := thCreateSong(t, db, nil, folderID)
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, RemoteMBID: &remoteMBID, Uploaded: true},
			}, true))

			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{remoteMBID}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("includes starred song with uploaded=true when MBID not in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: true},
			}, false))

			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{"other-mbid"}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.Contains(t, ids, songID)
		})

		t.Run("excludes starred song with uploaded=true when MBID is in lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: true},
			}, false))

			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.NotContains(t, ids, songID)
		})

		t.Run("excludes songs with uploaded=false", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: false},
			}, false))

			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{mbid}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			ids := util.Map(songs, func(s *repos.CompleteSong) string { return s.ID })
			assert.NotContains(t, ids, songID)
		})

		t.Run("works with empty lbLovedMBIDs", func(t *testing.T) {
			songs, err := repo.FindLocalOutdatedFeedbackByLB(ctx, user, []string{}, repos.IncludeSongInfoBare())
			require.NoError(t, err)
			assert.NotNil(t, songs)
		})
	})

	t.Run("SetLBFeedbackUploaded", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)

		t.Run("creates new lb_feedback_status row", func(t *testing.T) {
			songID := thCreateSong(t, db, nil, folderID)
			err := repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: true},
			}, false)
			require.NoError(t, err)
			assert.True(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user, "uploaded": true}))
		})

		t.Run("updates uploaded on existing row", func(t *testing.T) {
			songID := thCreateSong(t, db, nil, folderID)
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: true},
			}, false))

			err := repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: false},
			}, false)
			require.NoError(t, err)
			assert.True(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user, "uploaded": false}))
			assert.Equal(t, 1, thCountWhere(t, db, "lb_feedback_status", "song_id = '"+songID+"' AND user_name = '"+user+"'"))
		})

		t.Run("updates remote_mbid when updateRemoteMBIDs is true", func(t *testing.T) {
			songID := thCreateSong(t, db, nil, folderID)
			remoteMBID := "remote-" + crossonic.GenIDSong()
			err := repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, RemoteMBID: &remoteMBID, Uploaded: true},
			}, true)
			require.NoError(t, err)
			assert.True(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user, "remote_mbid": remoteMBID}))
		})

		t.Run("does not update remote_mbid when updateRemoteMBIDs is false", func(t *testing.T) {
			songID := thCreateSong(t, db, nil, folderID)
			oldMBID := "old-remote-" + crossonic.GenIDSong()
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, RemoteMBID: &oldMBID, Uploaded: true},
			}, true))

			newMBID := "new-remote-" + crossonic.GenIDSong()
			err := repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, RemoteMBID: &newMBID, Uploaded: false},
			}, false)
			require.NoError(t, err)
			assert.True(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user, "remote_mbid": oldMBID}))
		})
	})

	t.Run("SetLBFeedbackUploadedForAllMatchingStarredSongs", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)

		t.Run("marks starred songs with matching MBID as uploaded", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))

			err := repo.SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx, user, []string{mbid})
			require.NoError(t, err)
			assert.True(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user, "uploaded": true}))
		})

		t.Run("does not mark unstarred songs even with matching MBID", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)

			err := repo.SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx, user, []string{mbid})
			require.NoError(t, err)
			assert.False(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user}))
		})

		t.Run("does not mark starred songs with non-matching MBID", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))

			err := repo.SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx, user, []string{"other-mbid"})
			require.NoError(t, err)
			assert.False(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user}))
		})

		t.Run("does nothing with empty lbLovedMBIDs", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))

			err := repo.SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx, user, []string{})
			require.NoError(t, err)
			assert.False(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user}))
		})

		t.Run("updates existing status to uploaded=true on conflict", func(t *testing.T) {
			mbid := "mbid-" + crossonic.GenIDSong()
			songID := thCreateSongWithMBID(t, db, folderID, mbid)
			require.NoError(t, repo.Star(ctx, user, songID))
			require.NoError(t, repo.SetLBFeedbackUploaded(ctx, user, []repos.SongSetLBFeedbackUploadedParams{
				{SongID: songID, Uploaded: false},
			}, false))

			err := repo.SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx, user, []string{mbid})
			require.NoError(t, err)
			assert.True(t, thExists(t, db, "lb_feedback_status", map[string]any{"song_id": songID, "user_name": user, "uploaded": true}))
		})
	})
}

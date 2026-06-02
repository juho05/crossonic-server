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

func TestArtistRepository(t *testing.T) {
	db, _ := thSetupDatabase(t)
	ctx := context.Background()
	repo := db.Artist()

	user := thCreateUser(t, db)
	user2 := thCreateUser(t, db)

	t.Run("Create", func(t *testing.T) {
		t.Run("creates artist with correct fields", func(t *testing.T) {
			mbid := "test-mbid"
			id, err := repo.Create(ctx, repos.CreateArtistParams{
				Name:          "My Artist",
				MusicBrainzID: &mbid,
			})
			require.NoErrorf(t, err, "create artist: %v", err)
			assert.True(t, crossonic.IsIDType(id, crossonic.IDTypeArtist), "expected artist ID type, got: %s", id)
			assert.True(t, thExists(t, db, "artists", map[string]any{
				"id":              id,
				"name":            "My Artist",
				"music_brainz_id": mbid,
			}))
		})

		t.Run("creates artist without mbid", func(t *testing.T) {
			id, err := repo.Create(ctx, repos.CreateArtistParams{Name: "No MBID Artist"})
			require.NoErrorf(t, err, "create artist: %v", err)
			assert.True(t, thExists(t, db, "artists", map[string]any{
				"id":              id,
				"music_brainz_id": nil,
			}))
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("update name", func(t *testing.T) {
			artistID := thCreateArtist(t, db)
			err := repo.Update(ctx, artistID, repos.UpdateArtistParams{
				Name: repos.NewOptionalFull("Updated Name"),
			})
			require.NoErrorf(t, err, "update artist: %v", err)
			assert.True(t, thExists(t, db, "artists", map[string]any{"id": artistID, "name": "Updated Name"}))
		})

		t.Run("update music brainz id", func(t *testing.T) {
			artistID := thCreateArtist(t, db)
			mbid := "new-mbid"
			err := repo.Update(ctx, artistID, repos.UpdateArtistParams{
				MusicBrainzID: repos.NewOptionalFull(&mbid),
			})
			require.NoErrorf(t, err, "update artist: %v", err)
			assert.True(t, thExists(t, db, "artists", map[string]any{"id": artistID, "music_brainz_id": mbid}))
		})

		t.Run("empty update does not error", func(t *testing.T) {
			artistID := thCreateArtist(t, db)
			err := repo.Update(ctx, artistID, repos.UpdateArtistParams{})
			assert.NoErrorf(t, err, "empty update: %v", err)
		})

		t.Run("artist does not exist", func(t *testing.T) {
			err := repo.Update(ctx, "ar_doesnotexist", repos.UpdateArtistParams{
				Name: repos.NewOptionalFull("X"),
			})
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})
	})

	t.Run("DeleteIfNoAlbumsAndNoSongs", func(t *testing.T) {
		t.Run("deletes artist with no connections", func(t *testing.T) {
			artistID := thCreateArtist(t, db)
			err := repo.DeleteIfNoAlbumsAndNoSongs(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.False(t, thExists(t, db, "artists", map[string]any{"id": artistID}))
		})

		t.Run("keeps artist connected to an album", func(t *testing.T) {
			artistID := thCreateArtist(t, db)
			folderID := thCreateMusicFolder(t, db, user)
			albumID := thCreateAlbum(t, db, folderID)
			_, err := db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0)", albumID, artistID)
			require.NoError(t, err)

			err = repo.DeleteIfNoAlbumsAndNoSongs(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.True(t, thExists(t, db, "artists", map[string]any{"id": artistID}))
		})

		t.Run("keeps artist connected to a song", func(t *testing.T) {
			artistID := thCreateArtist(t, db)
			folderID := thCreateMusicFolder(t, db, user)
			songID := thCreateSong(t, db, nil, folderID)
			_, err := db.db.ExecContext(ctx, "INSERT INTO song_artist (song_id, artist_id, index) VALUES ($1, $2, 0)", songID, artistID)
			require.NoError(t, err)

			err = repo.DeleteIfNoAlbumsAndNoSongs(ctx)
			require.NoErrorf(t, err, "delete: %v", err)
			assert.True(t, thExists(t, db, "artists", map[string]any{"id": artistID}))
		})
	})

	t.Run("FindByID", func(t *testing.T) {
		artistID, folderID := thCreateArtistInMusicFolder(t, db, user)

		t.Run("returns artist for authorized user", func(t *testing.T) {
			a, err := repo.FindByID(ctx, artistID, user, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a)
			assert.Equal(t, artistID, a.ID)
		})

		t.Run("returns not found for unauthorized user", func(t *testing.T) {
			isolatedFolder := thCreateMusicFolder(t, db)
			isolatedArtist := thCreateArtist(t, db)
			thAssociateMusicFolderArtist(t, db, isolatedArtist, isolatedFolder)
			_, err := repo.FindByID(ctx, isolatedArtist, user, repos.IncludeArtistInfoBare())
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})

		t.Run("includes album count", func(t *testing.T) {
			albumID := thCreateAlbum(t, db, folderID)
			_, err := db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0)", albumID, artistID)
			require.NoError(t, err)
			a, err := repo.FindByID(ctx, artistID, user, repos.IncludeArtistInfo{AlbumInfo: true})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a.ArtistAlbumInfo)
			assert.GreaterOrEqual(t, a.AlbumCount, 1)
		})

		t.Run("includes annotation info", func(t *testing.T) {
			err := repo.Star(ctx, user, artistID)
			require.NoError(t, err)
			a, err := repo.FindByID(ctx, artistID, user, repos.IncludeArtistInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find by id: %v", err)
			require.NotNil(t, a.ArtistAnnotations)
			assert.NotNil(t, a.Starred)
		})
	})

	t.Run("FindByNames", func(t *testing.T) {
		t.Run("empty slice returns empty result", func(t *testing.T) {
			results, err := repo.FindByNames(ctx, []string{}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find by names: %v", err)
			assert.Empty(t, results)
		})

		t.Run("returns matching artists", func(t *testing.T) {
			name1 := "FBN-Artist1-" + t.Name()
			name2 := "FBN-Artist2-" + t.Name()
			_, err := repo.Create(ctx, repos.CreateArtistParams{Name: name1})
			require.NoError(t, err)
			_, err = repo.Create(ctx, repos.CreateArtistParams{Name: name2})
			require.NoError(t, err)

			results, err := repo.FindByNames(ctx, []string{name1, name2}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find by names: %v", err)
			assert.Len(t, results, 2)
			names := util.Map(results, func(a *repos.CompleteArtist) string { return a.Name })
			assert.Contains(t, names, name1)
			assert.Contains(t, names, name2)
		})

		t.Run("returns empty when no match", func(t *testing.T) {
			results, err := repo.FindByNames(ctx, []string{"does-not-exist-xyz"}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find by names: %v", err)
			assert.Empty(t, results)
		})
	})

	t.Run("FindAll", func(t *testing.T) {
		t.Run("returns artists in music folder", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			a1 := thCreateArtist(t, db)
			a2 := thCreateArtist(t, db)
			thAssociateMusicFolderArtist(t, db, a1, folderID)
			thAssociateMusicFolderArtist(t, db, a2, folderID)

			results, err := repo.FindAll(ctx, repos.FindArtistsParams{
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteArtist) string { return a.ID })
			assert.Contains(t, ids, a1)
			assert.Contains(t, ids, a2)
		})

		t.Run("updatedAfter filters correctly", func(t *testing.T) {
			before := time.Now().Add(-time.Hour)
			a := thCreateArtist(t, db)
			after := time.Now().Add(time.Hour)
			folderID := thCreateMusicFolder(t, db, user)
			thAssociateMusicFolderArtist(t, db, a, folderID)

			results, err := repo.FindAll(ctx, repos.FindArtistsParams{
				UpdatedAfter:   &before,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			assert.Contains(t, util.Map(results, func(a *repos.CompleteArtist) string { return a.ID }), a)

			results, err = repo.FindAll(ctx, repos.FindArtistsParams{
				UpdatedAfter:   &after,
				MusicFolderIDs: []int{folderID},
			}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find all: %v", err)
			assert.NotContains(t, util.Map(results, func(a *repos.CompleteArtist) string { return a.ID }), a)
		})

		t.Run("onlyAlbumArtists filters artists without albums", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			albumArtist := thCreateArtist(t, db)
			nonAlbumArtist := thCreateArtist(t, db)
			thAssociateMusicFolderArtist(t, db, albumArtist, folderID)
			thAssociateMusicFolderArtist(t, db, nonAlbumArtist, folderID)

			albumID := thCreateAlbum(t, db, folderID)
			_, err := db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0)", albumID, albumArtist)
			require.NoError(t, err)

			results, err := repo.FindAll(ctx, repos.FindArtistsParams{
				OnlyAlbumArtists: true,
				MusicFolderIDs:   []int{folderID},
			}, repos.IncludeArtistInfo{AlbumInfo: true})
			require.NoErrorf(t, err, "find all: %v", err)
			ids := util.Map(results, func(a *repos.CompleteArtist) string { return a.ID })
			assert.Contains(t, ids, albumArtist)
			assert.NotContains(t, ids, nonAlbumArtist)
		})

		t.Run("onlyAlbumArtists requires include.AlbumInfo", func(t *testing.T) {
			_, err := repo.FindAll(ctx, repos.FindArtistsParams{OnlyAlbumArtists: true}, repos.IncludeArtistInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})
	})

	t.Run("FindBySearch", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		id, err := repo.Create(ctx, repos.CreateArtistParams{Name: "SearchableArtistXYZ"})
		require.NoError(t, err)
		thAssociateMusicFolderArtist(t, db, id, folderID)

		t.Run("finds artist by name", func(t *testing.T) {
			results, err := repo.FindBySearch(ctx, "SearchableArtistXYZ", false, []int{folderID}, repos.Paginate{}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find by search: %v", err)
			ids := util.Map(results, func(a *repos.CompleteArtist) string { return a.ID })
			assert.Contains(t, ids, id)
		})

		t.Run("does not find artist outside music folder", func(t *testing.T) {
			otherFolder := thCreateMusicFolder(t, db, user)
			results, err := repo.FindBySearch(ctx, "SearchableArtistXYZ", false, []int{otherFolder}, repos.Paginate{}, repos.IncludeArtistInfoBare())
			require.NoErrorf(t, err, "find by search: %v", err)
			ids := util.Map(results, func(a *repos.CompleteArtist) string { return a.ID })
			assert.NotContains(t, ids, id)
		})

		t.Run("onlyAlbumArtists requires include.AlbumInfo", func(t *testing.T) {
			_, err := repo.FindBySearch(ctx, "SearchableArtistXYZ", true, []int{folderID}, repos.Paginate{}, repos.IncludeArtistInfoBare())
			assert.ErrorIs(t, err, repos.ErrInvalidParams)
		})

		t.Run("onlyAlbumArtists filters non-album artists", func(t *testing.T) {
			searchFolder := thCreateMusicFolder(t, db, user)
			albumArtist, err := repo.Create(ctx, repos.CreateArtistParams{Name: "AlbumArtistSearchXYZ"})
			require.NoError(t, err)
			nonAlbumArtist, err := repo.Create(ctx, repos.CreateArtistParams{Name: "NonAlbumArtistSearchXYZ"})
			require.NoError(t, err)
			thAssociateMusicFolderArtist(t, db, albumArtist, searchFolder)
			thAssociateMusicFolderArtist(t, db, nonAlbumArtist, searchFolder)
			albumID := thCreateAlbum(t, db, searchFolder)
			_, err = db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0)", albumID, albumArtist)
			require.NoError(t, err)

			results, err := repo.FindBySearch(ctx, "AlbumArtistSearchXYZ", true, []int{searchFolder}, repos.Paginate{}, repos.IncludeArtistInfo{AlbumInfo: true})
			require.NoErrorf(t, err, "find by search album artist: %v", err)
			assert.Contains(t, util.Map(results, func(a *repos.CompleteArtist) string { return a.ID }), albumArtist)

			results, err = repo.FindBySearch(ctx, "NonAlbumArtistSearchXYZ", true, []int{searchFolder}, repos.Paginate{}, repos.IncludeArtistInfo{AlbumInfo: true})
			require.NoErrorf(t, err, "find by search non-album artist: %v", err)
			assert.NotContains(t, util.Map(results, func(a *repos.CompleteArtist) string { return a.ID }), nonAlbumArtist)
		})
	})

	t.Run("Star and UnStar", func(t *testing.T) {
		artistID, _ := thCreateArtistInMusicFolder(t, db, user)

		t.Run("star artist", func(t *testing.T) {
			err := repo.Star(ctx, user, artistID)
			require.NoErrorf(t, err, "star: %v", err)
			assert.True(t, thExists(t, db, "artist_stars", map[string]any{"artist_id": artistID, "user_name": user}))
		})

		t.Run("star is idempotent", func(t *testing.T) {
			err := repo.Star(ctx, user, artistID)
			assert.NoErrorf(t, err, "re-star should not fail: %v", err)
		})

		t.Run("unstar artist", func(t *testing.T) {
			err := repo.UnStar(ctx, user, artistID)
			require.NoErrorf(t, err, "unstar: %v", err)
			assert.False(t, thExists(t, db, "artist_stars", map[string]any{"artist_id": artistID, "user_name": user}))
		})

		t.Run("star for one user does not affect another", func(t *testing.T) {
			err := repo.Star(ctx, user, artistID)
			require.NoError(t, err)
			assert.False(t, thExists(t, db, "artist_stars", map[string]any{"artist_id": artistID, "user_name": user2}))
		})
	})

	t.Run("FindStarred", func(t *testing.T) {
		artistID, folderID := thCreateArtistInMusicFolder(t, db, user)
		_ = repo.UnStar(ctx, user, artistID)

		t.Run("not returned when not starred", func(t *testing.T) {
			results, err := repo.FindStarred(ctx, []int{folderID}, repos.Paginate{}, repos.IncludeArtistInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find starred: %v", err)
			ids := util.Map(results, func(a *repos.CompleteArtist) string { return a.ID })
			assert.NotContains(t, ids, artistID)
		})

		t.Run("returned when starred", func(t *testing.T) {
			require.NoError(t, repo.Star(ctx, user, artistID))
			results, err := repo.FindStarred(ctx, []int{folderID}, repos.Paginate{}, repos.IncludeArtistInfo{Annotations: true, User: user})
			require.NoErrorf(t, err, "find starred: %v", err)
			ids := util.Map(results, func(a *repos.CompleteArtist) string { return a.ID })
			assert.Contains(t, ids, artistID)
		})

		t.Run("requires include.Annotations and User", func(t *testing.T) {
			_, err := repo.FindStarred(ctx, nil, repos.Paginate{}, repos.IncludeArtistInfoBare())
			assert.Error(t, err)
		})
	})

	t.Run("SetRating and RemoveRating", func(t *testing.T) {
		artistID := thCreateArtist(t, db)

		t.Run("set rating", func(t *testing.T) {
			err := repo.SetRating(ctx, user, artistID, 4)
			require.NoErrorf(t, err, "set rating: %v", err)
			assert.True(t, thExists(t, db, "artist_ratings", map[string]any{"artist_id": artistID, "user_name": user, "rating": 4}))
		})

		t.Run("update existing rating", func(t *testing.T) {
			err := repo.SetRating(ctx, user, artistID, 2)
			require.NoErrorf(t, err, "update rating: %v", err)
			assert.True(t, thExists(t, db, "artist_ratings", map[string]any{"artist_id": artistID, "user_name": user, "rating": 2}))
			assert.Equal(t, 1, thCountWhere(t, db, "artist_ratings", "artist_id = '"+artistID+"' AND user_name = '"+user+"'"))
		})

		t.Run("rating for one user does not affect another", func(t *testing.T) {
			err := repo.SetRating(ctx, user, artistID, 5)
			require.NoError(t, err)
			assert.False(t, thExists(t, db, "artist_ratings", map[string]any{"artist_id": artistID, "user_name": user2}))
		})

		t.Run("remove rating", func(t *testing.T) {
			err := repo.RemoveRating(ctx, user, artistID)
			require.NoErrorf(t, err, "remove rating: %v", err)
			assert.False(t, thExists(t, db, "artist_ratings", map[string]any{"artist_id": artistID, "user_name": user}))
		})
	})

	t.Run("GetInfo and SetInfo", func(t *testing.T) {
		artistID, _ := thCreateArtistInMusicFolder(t, db, user)

		t.Run("not found for unknown artist", func(t *testing.T) {
			_, err := repo.GetInfo(ctx, "ar_doesnotexist", user)
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})

		t.Run("set and get info", func(t *testing.T) {
			bio := "A great artist"
			url := "https://last.fm/artist"
			err := repo.SetInfo(ctx, artistID, repos.SetArtistInfo{
				Biography: &bio,
				LastFMURL: &url,
			})
			require.NoErrorf(t, err, "set info: %v", err)

			info, err := repo.GetInfo(ctx, artistID, user)
			require.NoErrorf(t, err, "get info: %v", err)
			require.NotNil(t, info)
			assert.Equal(t, artistID, info.ArtistID)
			require.NotNil(t, info.Biography)
			assert.Equal(t, bio, *info.Biography)
			require.NotNil(t, info.LastFMURL)
			assert.Equal(t, url, *info.LastFMURL)
		})

		t.Run("not accessible by user without music folder access", func(t *testing.T) {
			isolatedArtist := thCreateArtist(t, db)
			isolatedFolder := thCreateMusicFolder(t, db)
			thAssociateMusicFolderArtist(t, db, isolatedArtist, isolatedFolder)
			_, err := repo.GetInfo(ctx, isolatedArtist, user)
			assert.ErrorIs(t, err, repos.ErrNotFound)
		})
	})

	t.Run("MigrateAnnotations", func(t *testing.T) {
		oldArtist := thCreateArtist(t, db)
		newArtist := thCreateArtist(t, db)

		require.NoError(t, repo.Star(ctx, user, oldArtist))
		require.NoError(t, repo.SetRating(ctx, user, oldArtist, 3))

		err := repo.MigrateAnnotations(ctx, oldArtist, newArtist)
		require.NoErrorf(t, err, "migrate annotations: %v", err)

		assert.True(t, thExists(t, db, "artist_stars", map[string]any{"artist_id": newArtist, "user_name": user}))
		assert.True(t, thExists(t, db, "artist_ratings", map[string]any{"artist_id": newArtist, "user_name": user, "rating": 3}))
	})

	t.Run("GetAlbums", func(t *testing.T) {
		artistID, folderID := thCreateArtistInMusicFolder(t, db, user)
		albumID := thCreateAlbum(t, db, folderID)
		_, err := db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0)", albumID, artistID)
		require.NoError(t, err)

		t.Run("returns albums for artist", func(t *testing.T) {
			albums, err := repo.GetAlbums(ctx, artistID, []int{folderID}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get albums: %v", err)
			ids := util.Map(albums, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, albumID)
		})

		t.Run("does not return albums outside music folder", func(t *testing.T) {
			albums, err := repo.GetAlbums(ctx, artistID, []int{thCreateMusicFolder(t, db, user)}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get albums: %v", err)
			assert.Empty(t, albums)
		})
	})

	t.Run("GetAppearsOnAlbums", func(t *testing.T) {
		folderID := thCreateMusicFolder(t, db, user)
		albumArtist := thCreateArtist(t, db)
		songArtist := thCreateArtist(t, db)
		thAssociateMusicFolderArtist(t, db, albumArtist, folderID)
		thAssociateMusicFolderArtist(t, db, songArtist, folderID)
		albumID := thCreateAlbum(t, db, folderID)
		songID := thCreateSong(t, db, &albumID, folderID)
		_, err := db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0)", albumID, albumArtist)
		require.NoError(t, err)
		_, err = db.db.ExecContext(ctx, "INSERT INTO song_artist (song_id, artist_id, index) VALUES ($1, $2, 0)", songID, songArtist)
		require.NoError(t, err)

		t.Run("returns album where artist appears on a song but is not the album artist", func(t *testing.T) {
			albums, err := repo.GetAppearsOnAlbums(ctx, songArtist, []int{folderID}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get appears on albums: %v", err)
			ids := util.Map(albums, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.Contains(t, ids, albumID)
		})

		t.Run("does not return album where artist is the album artist", func(t *testing.T) {
			albums, err := repo.GetAppearsOnAlbums(ctx, albumArtist, []int{folderID}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get appears on albums: %v", err)
			ids := util.Map(albums, func(a *repos.CompleteAlbum) string { return a.ID })
			assert.NotContains(t, ids, albumID)
		})

		t.Run("does not return albums outside music folder", func(t *testing.T) {
			albums, err := repo.GetAppearsOnAlbums(ctx, songArtist, []int{thCreateMusicFolder(t, db, user)}, repos.IncludeAlbumInfoBare())
			require.NoErrorf(t, err, "get appears on albums: %v", err)
			assert.Empty(t, albums)
		})
	})

	t.Run("FindArtistIDsToMigrate", func(t *testing.T) {
		t.Run("finds old artist to migrate to newer artist with same mbid", func(t *testing.T) {
			mbid := "artist-migrate-mbid-" + thCreateArtist(t, db)
			oldArtist, err := repo.Create(ctx, repos.CreateArtistParams{
				Name:          "OldMigrateArtist",
				MusicBrainzID: &mbid,
			})
			require.NoError(t, err)

			scanStartTime := time.Now()

			newArtist, err := repo.Create(ctx, repos.CreateArtistParams{
				Name:          "NewMigrateArtist",
				MusicBrainzID: &mbid,
			})
			require.NoError(t, err)

			results, err := repo.FindArtistIDsToMigrate(ctx, scanStartTime)
			require.NoErrorf(t, err, "find artist ids to migrate: %v", err)
			var found bool
			for _, r := range results {
				if r.OldID == oldArtist && r.NewID == newArtist {
					found = true
					break
				}
			}
			assert.True(t, found, "expected migration pair (%s -> %s) in results", oldArtist, newArtist)
		})

		t.Run("does not return artist that has songs", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			mbid := "artist-has-songs-mbid-" + thCreateArtist(t, db)
			artistWithSongs, err := repo.Create(ctx, repos.CreateArtistParams{
				Name:          "ArtistWithSongs",
				MusicBrainzID: &mbid,
			})
			require.NoError(t, err)
			songID := thCreateSong(t, db, nil, folderID)
			_, err = db.db.ExecContext(ctx, "INSERT INTO song_artist (song_id, artist_id, index) VALUES ($1, $2, 0)", songID, artistWithSongs)
			require.NoError(t, err)

			scanStartTime := time.Now()

			_, err = repo.Create(ctx, repos.CreateArtistParams{
				Name:          "ArtistWithSongsDuplicate",
				MusicBrainzID: &mbid,
			})
			require.NoError(t, err)

			results, err := repo.FindArtistIDsToMigrate(ctx, scanStartTime)
			require.NoErrorf(t, err, "find artist ids to migrate: %v", err)
			oldIDs := util.Map(results, func(r repos.FindArtistIDsToMigrateResult) string { return r.OldID })
			assert.NotContains(t, oldIDs, artistWithSongs)
		})

		t.Run("does not return artist that has albums", func(t *testing.T) {
			folderID := thCreateMusicFolder(t, db, user)
			mbid := "artist-has-albums-mbid-" + thCreateArtist(t, db)
			artistWithAlbums, err := repo.Create(ctx, repos.CreateArtistParams{
				Name:          "ArtistWithAlbums",
				MusicBrainzID: &mbid,
			})
			require.NoError(t, err)
			albumID := thCreateAlbum(t, db, folderID)
			_, err = db.db.ExecContext(ctx, "INSERT INTO album_artist (album_id, artist_id, index) VALUES ($1, $2, 0)", albumID, artistWithAlbums)
			require.NoError(t, err)

			scanStartTime := time.Now()

			_, err = repo.Create(ctx, repos.CreateArtistParams{
				Name:          "ArtistWithAlbumsDuplicate",
				MusicBrainzID: &mbid,
			})
			require.NoError(t, err)

			results, err := repo.FindArtistIDsToMigrate(ctx, scanStartTime)
			require.NoErrorf(t, err, "find artist ids to migrate: %v", err)
			oldIDs := util.Map(results, func(r repos.FindArtistIDsToMigrateResult) string { return r.OldID })
			assert.NotContains(t, oldIDs, artistWithAlbums)
		})

		t.Run("does not return artist without music brainz id", func(t *testing.T) {
			artistNoMBID := thCreateArtist(t, db)

			scanStartTime := time.Now()

			_, err := repo.Create(ctx, repos.CreateArtistParams{Name: "NoMBIDDuplicate"})
			require.NoError(t, err)

			results, err := repo.FindArtistIDsToMigrate(ctx, scanStartTime)
			require.NoErrorf(t, err, "find artist ids to migrate: %v", err)
			oldIDs := util.Map(results, func(r repos.FindArtistIDsToMigrateResult) string { return r.OldID })
			assert.NotContains(t, oldIDs, artistNoMBID)
		})
	})
}

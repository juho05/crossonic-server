package postgres

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var thMusicFolderIDCounter int64

func thNextMusicFolderID() int {
	return int(atomic.AddInt64(&thMusicFolderIDCounter, 1))
}

func thCreateMusicFolder(t *testing.T, db *DB, users ...string) int {
	t.Helper()
	ctx := context.Background()
	id := thNextMusicFolderID()
	err := db.MusicFolder().CreateOrUpdate(ctx, []repos.CreateMusicFolderParams{
		{ID: id, Name: "Test Folder " + strconv.Itoa(id), Path: "/test/" + strconv.Itoa(id)},
	})
	require.NoErrorf(t, err, "create music folder: %v", err)
	for _, user := range users {
		err = db.MusicFolder().CreateUserAssociations(ctx, id, []string{user})
		require.NoErrorf(t, err, "create music folder user association: %v", err)
	}
	return id
}

// thCreateArtist creates an artist but does NOT set up music folder associations.
// Use thCreateArtistInMusicFolder if you need the artist to be accessible by a user.
func thCreateArtist(t *testing.T, db *DB) string {
	t.Helper()
	artistID, err := db.Artist().Create(context.Background(), repos.CreateArtistParams{
		Name: "Test Artist " + uuid.NewString(),
	})
	require.NoErrorf(t, err, "create test artist: %v", err)
	return artistID
}

func thAssociateMusicFolderArtist(t *testing.T, db *DB, artistID string, musicFolderID int) {
	t.Helper()
	err := db.MusicFolder().CreateArtistAssociations(context.Background(), []repos.ArtistMusicFolderAssociation{
		{MusicFolderID: musicFolderID, ArtistID: artistID},
	})
	require.NoErrorf(t, err, "associate artist with music folder: %v", err)
}

// thCreateArtistInMusicFolder creates an artist accessible by the given user via a new music folder.
func thCreateArtistInMusicFolder(t *testing.T, db *DB, user string) (artistID string, musicFolderID int) {
	t.Helper()
	musicFolderID = thCreateMusicFolder(t, db, user)
	artistID = thCreateArtist(t, db)
	thAssociateMusicFolderArtist(t, db, artistID, musicFolderID)
	return artistID, musicFolderID
}

func thCreateAlbum(t *testing.T, db *DB, musicFolderID int) string {
	t.Helper()
	albumID, err := db.Album().Create(context.Background(), repos.CreateAlbumParams{
		Name:          "Test Album " + uuid.NewString(),
		MusicFolderID: musicFolderID,
	})
	require.NoErrorf(t, err, "create test album: %v", err)
	return albumID
}

func thCreateSong(t *testing.T, db *DB, albumID *string, musicFolderID int) string {
	t.Helper()
	id := crossonic.GenIDSong()
	err := db.Song().CreateAll(context.Background(), []repos.CreateSongParams{
		{
			ID:            &id,
			Path:          "/test/song-" + uuid.NewString() + ".mp3",
			AlbumID:       albumID,
			Title:         "Test Song " + uuid.NewString(),
			Size:          1000,
			ContentType:   "audio/mpeg",
			Duration:      repos.NewDurationMS(180000),
			BitRate:       320,
			SamplingRate:  44100,
			ChannelCount:  2,
			MusicFolderID: musicFolderID,
		},
	})
	require.NoErrorf(t, err, "create test song: %v", err)
	return id
}

func thCreateGenre(t *testing.T, db *DB, name string) {
	t.Helper()
	err := db.Genre().CreateIfNotExists(context.Background(), []string{name})
	require.NoErrorf(t, err, "create test genre: %v", err)
}

func thCreateSongWithMBID(t *testing.T, db *DB, musicFolderID int, mbid string) string {
	t.Helper()
	id := crossonic.GenIDSong()
	err := db.Song().CreateAll(context.Background(), []repos.CreateSongParams{
		{
			ID:            &id,
			Path:          "/test/song-" + uuid.NewString() + ".mp3",
			Title:         "Test Song " + uuid.NewString(),
			MusicBrainzID: &mbid,
			Size:          1000,
			ContentType:   "audio/mpeg",
			Duration:      repos.NewDurationMS(180000),
			BitRate:       320,
			SamplingRate:  44100,
			ChannelCount:  2,
			MusicFolderID: musicFolderID,
		},
	})
	require.NoErrorf(t, err, "create test song with mbid: %v", err)
	return id
}
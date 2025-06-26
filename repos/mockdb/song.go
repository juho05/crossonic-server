package mockdb

import (
	"context"
	"github.com/juho05/crossonic-server/repos"
	"time"
)

type SongRepository struct {
	FindByIDMock                                        func(ctx context.Context, id string, include repos.IncludeSongInfo) (*repos.CompleteSong, error)
	FindByIDsMock                                       func(ctx context.Context, ids []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindByMusicBrainzIDMock                             func(ctx context.Context, musicBrainzID string) ([]*repos.CompleteSong, error)
	FindByPathMock                                      func(ctx context.Context, path string, include repos.IncludeSongInfo) (*repos.CompleteSong, error)
	FindRandomMock                                      func(ctx context.Context, params repos.SongFindRandomParams, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindBySearchMock                                    func(ctx context.Context, params repos.SongFindBySearchParams, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindStarredMock                                     func(ctx context.Context, paginate repos.Paginate, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindByGenreMock                                     func(ctx context.Context, genre string, paginate repos.Paginate, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindByTitleMock                                     func(ctx context.Context, title string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindAllByPathOrMBIDMock                             func(ctx context.Context, paths []string, mbids []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindNonExistentIDsMock                              func(ctx context.Context, ids []string) ([]string, error)
	FindPathsMock                                       func(ctx context.Context, updatedBefore time.Time, paginate repos.Paginate) ([]string, error)
	DeleteByPathsMock                                   func(ctx context.Context, paths []string) error
	GetStreamInfoMock                                   func(ctx context.Context, id string) (*repos.SongStreamInfo, error)
	CreateAllMock                                       func(ctx context.Context, params []repos.CreateSongParams) error
	TryUpdateAllMock                                    func(ctx context.Context, params []repos.UpdateSongAllParams) (int, error)
	DeleteLastUpdatedBeforeMock                         func(ctx context.Context, before time.Time) error
	DeleteArtistConnectionsMock                         func(ctx context.Context, songIDs []string) error
	CreateArtistConnectionsMock                         func(ctx context.Context, connections []repos.SongArtistConnection) error
	DeleteGenreConnectionsMock                          func(ctx context.Context, songIDs []string) error
	CreateGenreConnectionsMock                          func(ctx context.Context, connections []repos.SongGenreConnection) error
	StarMock                                            func(ctx context.Context, user, songID string) error
	StarMultipleMock                                    func(ctx context.Context, user string, songIDs []string) (int, error)
	UnStarMock                                          func(ctx context.Context, user, songID string) error
	UnStarMultipleMock                                  func(ctx context.Context, user string, songIDs []string) (int, error)
	SetRatingMock                                       func(ctx context.Context, user string, songID string, rating int) error
	RemoveRatingMock                                    func(ctx context.Context, user string, songID string) error
	FindNotUploadedLBFeedbackMock                       func(ctx context.Context, user string, lbLovedMBIDs []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	FindLocalOutdatedFeedbackByLBMock                   func(ctx context.Context, user string, lbLovedMBIDs []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	SetLBFeedbackUploadedMock                           func(ctx context.Context, user string, params []repos.SongSetLBFeedbackUploadedParams, updateRemoteMBIDs bool) error
	SetLBFeedbackUploadedForAllMatchingStarredSongsMock func(ctx context.Context, user string, lbLovedMBIDs []string) error
	CountMock                                           func(ctx context.Context) (int, error)
	GetMedianReplayGainMock                             func(ctx context.Context) (float64, error)
}

func (s SongRepository) FindByID(ctx context.Context, id string, include repos.IncludeSongInfo) (*repos.CompleteSong, error) {
	if s.FindByIDMock != nil {
		return s.FindByIDMock(ctx, id, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindByIDs(ctx context.Context, ids []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindByIDsMock != nil {
		return s.FindByIDsMock(ctx, ids, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindByMusicBrainzID(ctx context.Context, mbid string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindByMusicBrainzIDMock != nil {
		return s.FindByMusicBrainzIDMock(ctx, mbid)
	}
	panic("not implemented")
}

func (s SongRepository) FindByPath(ctx context.Context, path string, include repos.IncludeSongInfo) (*repos.CompleteSong, error) {
	if s.FindByPathMock != nil {
		return s.FindByPathMock(ctx, path, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindRandom(ctx context.Context, params repos.SongFindRandomParams, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindRandomMock != nil {
		return s.FindRandomMock(ctx, params, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindBySearch(ctx context.Context, params repos.SongFindBySearchParams, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindBySearchMock != nil {
		return s.FindBySearchMock(ctx, params, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindStarred(ctx context.Context, paginate repos.Paginate, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindStarredMock != nil {
		return s.FindStarredMock(ctx, paginate, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindByGenre(ctx context.Context, genre string, paginate repos.Paginate, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindByGenreMock != nil {
		return s.FindByGenreMock(ctx, genre, paginate, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindByTitle(ctx context.Context, title string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindByTitleMock != nil {
		return s.FindByTitle(ctx, title, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindAllByPathOrMBID(ctx context.Context, paths []string, mbids []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindAllByPathOrMBIDMock != nil {
		return s.FindAllByPathOrMBIDMock(ctx, paths, mbids, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindNonExistentIDs(ctx context.Context, ids []string) ([]string, error) {
	if s.FindNonExistentIDsMock != nil {
		return s.FindNonExistentIDsMock(ctx, ids)
	}
	panic("not implemented")
}

func (s SongRepository) FindPaths(ctx context.Context, updatedBefore time.Time, paginate repos.Paginate) ([]string, error) {
	if s.FindByPathMock != nil {
		return s.FindPathsMock(ctx, updatedBefore, paginate)
	}
	panic("not implemented")
}

func (s SongRepository) DeleteByPaths(ctx context.Context, paths []string) error {
	if s.DeleteByPathsMock != nil {
		return s.DeleteByPathsMock(ctx, paths)
	}
	panic("not implemented")
}

func (s SongRepository) GetStreamInfo(ctx context.Context, id string) (*repos.SongStreamInfo, error) {
	if s.GetStreamInfoMock != nil {
		return s.GetStreamInfoMock(ctx, id)
	}
	panic("not implemented")
}

func (s SongRepository) CreateAll(ctx context.Context, params []repos.CreateSongParams) error {
	if s.CreateAllMock != nil {
		return s.CreateAllMock(ctx, params)
	}
	panic("not implemented")
}

func (s SongRepository) TryUpdateAll(ctx context.Context, params []repos.UpdateSongAllParams) (int, error) {
	if s.TryUpdateAllMock != nil {
		return s.TryUpdateAllMock(ctx, params)
	}
	panic("not implemented")
}

func (s SongRepository) DeleteLastUpdatedBefore(ctx context.Context, before time.Time) error {
	if s.DeleteLastUpdatedBeforeMock != nil {
		return s.DeleteLastUpdatedBeforeMock(ctx, before)
	}
	panic("not implemented")
}

func (s SongRepository) DeleteArtistConnections(ctx context.Context, songIDs []string) error {
	if s.DeleteArtistConnectionsMock != nil {
		return s.DeleteArtistConnectionsMock(ctx, songIDs)
	}
	panic("not implemented")
}

func (s SongRepository) CreateArtistConnections(ctx context.Context, connections []repos.SongArtistConnection) error {
	if s.CreateArtistConnectionsMock != nil {
		return s.CreateArtistConnectionsMock(ctx, connections)
	}
	panic("not implemented")
}

func (s SongRepository) DeleteGenreConnections(ctx context.Context, songIDs []string) error {
	if s.DeleteGenreConnectionsMock != nil {
		return s.DeleteGenreConnectionsMock(ctx, songIDs)
	}
	panic("not implemented")
}

func (s SongRepository) CreateGenreConnections(ctx context.Context, connections []repos.SongGenreConnection) error {
	if s.CreateGenreConnectionsMock != nil {
		return s.CreateGenreConnectionsMock(ctx, connections)
	}
	panic("not implemented")
}

func (s SongRepository) Star(ctx context.Context, user, songID string) error {
	if s.StarMock != nil {
		return s.StarMock(ctx, user, songID)
	}
	panic("not implemented")
}

func (s SongRepository) StarMultiple(ctx context.Context, user string, songIDs []string) (int, error) {
	if s.StarMultipleMock != nil {
		return s.StarMultipleMock(ctx, user, songIDs)
	}
	panic("not implemented")
}

func (s SongRepository) UnStar(ctx context.Context, user, songID string) error {
	if s.UnStarMock != nil {
		return s.UnStarMock(ctx, user, songID)
	}
	panic("not implemented")
}

func (s SongRepository) UnStarMultiple(ctx context.Context, user string, songID []string) (int, error) {
	if s.UnStarMultipleMock != nil {
		return s.UnStarMultipleMock(ctx, user, songID)
	}
	panic("not implemented")
}

func (s SongRepository) SetRating(ctx context.Context, user, songID string, rating int) error {
	if s.SetRatingMock != nil {
		return s.SetRatingMock(ctx, user, songID, rating)
	}
	panic("not implemented")
}

func (s SongRepository) RemoveRating(ctx context.Context, user, songID string) error {
	if s.RemoveRatingMock != nil {
		return s.RemoveRatingMock(ctx, user, songID)
	}
	panic("not implemented")
}

func (s SongRepository) FindNotUploadedLBFeedback(ctx context.Context, user string, lbLovedMBIDs []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindNotUploadedLBFeedbackMock != nil {
		return s.FindNotUploadedLBFeedbackMock(ctx, user, lbLovedMBIDs, include)
	}
	panic("not implemented")
}

func (s SongRepository) FindLocalOutdatedFeedbackByLB(ctx context.Context, user string, lbLovedMBIDs []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if s.FindLocalOutdatedFeedbackByLBMock != nil {
		return s.FindLocalOutdatedFeedbackByLBMock(ctx, user, lbLovedMBIDs, include)
	}
	panic("not implemented")
}

func (s SongRepository) SetLBFeedbackUploaded(ctx context.Context, user string, params []repos.SongSetLBFeedbackUploadedParams, updateRemoteMBIDs bool) error {
	if s.SetLBFeedbackUploadedMock != nil {
		return s.SetLBFeedbackUploadedMock(ctx, user, params, updateRemoteMBIDs)
	}
	panic("not implemented")
}

func (s SongRepository) SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx context.Context, user string, lbLovedMBIDs []string) error {
	if s.SetLBFeedbackUploadedForAllMatchingStarredSongsMock != nil {
		return s.SetLBFeedbackUploadedForAllMatchingStarredSongsMock(ctx, user, lbLovedMBIDs)
	}
	panic("not implemented")
}

func (s SongRepository) Count(ctx context.Context) (int, error) {
	if s.CountMock != nil {
		return s.CountMock(ctx)
	}
	panic("not implemented")
}

func (s SongRepository) GetMedianReplayGain(ctx context.Context) (float64, error) {
	if s.GetMedianReplayGainMock != nil {
		return s.GetMedianReplayGainMock(ctx)
	}
	panic("not implemented")
}

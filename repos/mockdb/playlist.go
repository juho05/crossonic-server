package mockdb

import (
	"context"
	"github.com/juho05/crossonic-server/repos"
)

type PlaylistRepository struct {
	CreateMock          func(ctx context.Context, params repos.CreatePlaylistParams) (*repos.Playlist, error)
	UpdateMock          func(ctx context.Context, user, id string, params repos.UpdatePlaylistParams) error
	FindByIDMock        func(ctx context.Context, user, id string, include repos.IncludePlaylistInfo) (*repos.CompletePlaylist, error)
	FindAllMock         func(ctx context.Context, user string, include repos.IncludePlaylistInfo) ([]*repos.CompletePlaylist, error)
	GetTracksMock       func(ctx context.Context, id string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	AddTracksMock       func(ctx context.Context, id string, songIDs []string) error
	RemoveTracksMock    func(ctx context.Context, id string, trackNumbers []int) error
	ClearTracksMock     func(ctx context.Context, id string) error
	SetTracksMock       func(ctx context.Context, id string, songIDs []string) error
	FixTrackNumbersMock func(ctx context.Context) error
	DeleteMock          func(ctx context.Context, user, id string) error
}

func (p PlaylistRepository) Create(ctx context.Context, params repos.CreatePlaylistParams) (*repos.Playlist, error) {
	if p.CreateMock != nil {
		return p.CreateMock(ctx, params)
	}
	panic("not implemented")
}

func (p PlaylistRepository) Update(ctx context.Context, user, id string, params repos.UpdatePlaylistParams) error {
	if p.UpdateMock != nil {
		return p.UpdateMock(ctx, user, id, params)
	}
	panic("not implemented")
}

func (p PlaylistRepository) FindByID(ctx context.Context, user, id string, include repos.IncludePlaylistInfo) (*repos.CompletePlaylist, error) {
	if p.FindByIDMock != nil {
		return p.FindByIDMock(ctx, user, id, include)
	}
	panic("not implemented")
}

func (p PlaylistRepository) FindAll(ctx context.Context, user string, include repos.IncludePlaylistInfo) ([]*repos.CompletePlaylist, error) {
	if p.FindAllMock != nil {
		return p.FindAllMock(ctx, user, include)
	}
	panic("not implemented")
}

func (p PlaylistRepository) GetTracks(ctx context.Context, id string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if p.GetTracksMock != nil {
		return p.GetTracksMock(ctx, id, include)
	}
	panic("not implemented")
}

func (p PlaylistRepository) AddTracks(ctx context.Context, id string, songIDs []string) error {
	if p.AddTracksMock != nil {
		return p.AddTracksMock(ctx, id, songIDs)
	}
	panic("not implemented")
}

func (p PlaylistRepository) RemoveTracks(ctx context.Context, id string, trackNumbers []int) error {
	if p.RemoveTracksMock != nil {
		return p.RemoveTracksMock(ctx, id, trackNumbers)
	}
	panic("not implemented")
}

func (p PlaylistRepository) ClearTracks(ctx context.Context, id string) error {
	if p.ClearTracksMock != nil {
		return p.ClearTracksMock(ctx, id)
	}
	panic("not implemented")
}

func (p PlaylistRepository) SetTracks(ctx context.Context, id string, songIDs []string) error {
	if p.SetTracksMock != nil {
		return p.SetTracksMock(ctx, id, songIDs)
	}
	panic("not implemented")
}

func (p PlaylistRepository) FixTrackNumbers(ctx context.Context) error {
	if p.FixTrackNumbersMock != nil {
		return p.FixTrackNumbersMock(ctx)
	}
	panic("not implemented")
}

func (p PlaylistRepository) Delete(ctx context.Context, user, id string) error {
	if p.DeleteMock != nil {
		return p.DeleteMock(ctx, user, id)
	}
	panic("not implemented")
}

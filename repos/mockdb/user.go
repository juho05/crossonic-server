package mockdb

import (
	"context"

	"github.com/juho05/crossonic-server/repos"
)

type UserRepository struct {
	CreateMock                       func(ctx context.Context, name, password string) error
	UpdateListenBrainzConnectionMock func(ctx context.Context, name string) error
	UpdateListenBrainzSettingsMock   func(ctx context.Context, name string, params repos.UpdateListenBrainzSettingsParams) error
	FindAllMock                      func(ctx context.Context) ([]*repos.User, error)
	FindByNameMock                   func(ctx context.Context, name string) (*repos.User, error)
	DeleteByNameMock                 func(ctx context.Context, name string) error
}

func (u UserRepository) Create(ctx context.Context, name, password string) error {
	if u.CreateMock != nil {
		return u.CreateMock(ctx, name, password)
	}
	panic("not implemented")
}

func (u UserRepository) UpdateListenBrainzConnection(ctx context.Context, user string, lbUsername, lbToken *string) error {
	if u.UpdateListenBrainzConnectionMock != nil {
		return u.UpdateListenBrainzConnectionMock(ctx, user)
	}
	panic("not implemented")
}

func (u UserRepository) UpdateListenBrainzSettings(ctx context.Context, user string, params repos.UpdateListenBrainzSettingsParams) error {
	if u.UpdateListenBrainzSettingsMock != nil {
		return u.UpdateListenBrainzSettingsMock(ctx, user, params)
	}
	panic("not implemented")
}

func (u UserRepository) FindAll(ctx context.Context) ([]*repos.User, error) {
	if u.FindAllMock != nil {
		return u.FindAllMock(ctx)
	}
	panic("not implemented")
}

func (u UserRepository) FindByName(ctx context.Context, name string) (*repos.User, error) {
	if u.FindByNameMock != nil {
		return u.FindByNameMock(ctx, name)
	}
	panic("not implemented")
}

func (u UserRepository) DeleteByName(ctx context.Context, name string) error {
	if u.DeleteByNameMock != nil {
		return u.DeleteByNameMock(ctx, name)
	}
	panic("not implemented")
}

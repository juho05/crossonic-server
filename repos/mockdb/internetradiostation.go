package mockdb

import (
	"context"
	"github.com/juho05/crossonic-server/repos"
)

type InternetRadioStationRepository struct {
	FindAllMock func(ctx context.Context, user string) ([]*repos.InternetRadioStation, error)
	CreateMock  func(ctx context.Context, user string, params repos.CreateInternetRadioStationParams) (*repos.InternetRadioStation, error)
	UpdateMock  func(ctx context.Context, user, id string, params repos.UpdateInternetRadioStationParams) error
	DeleteMock  func(ctx context.Context, user, id string) error
}

func (i InternetRadioStationRepository) FindAll(ctx context.Context, user string) ([]*repos.InternetRadioStation, error) {
	if i.FindAllMock != nil {
		return i.FindAllMock(ctx, user)
	}
	panic("not implemented")
}

func (i InternetRadioStationRepository) Create(ctx context.Context, user string, params repos.CreateInternetRadioStationParams) (*repos.InternetRadioStation, error) {
	if i.CreateMock != nil {
		return i.CreateMock(ctx, user, params)
	}
	panic("not implemented")
}

func (i InternetRadioStationRepository) Update(ctx context.Context, user, id string, params repos.UpdateInternetRadioStationParams) error {
	if i.UpdateMock != nil {
		return i.UpdateMock(ctx, user, id, params)
	}
	panic("not implemented")
}

func (i InternetRadioStationRepository) Delete(ctx context.Context, user, id string) error {
	if i.DeleteMock != nil {
		return i.DeleteMock(ctx, user, id)
	}
	panic("not implemented")
}

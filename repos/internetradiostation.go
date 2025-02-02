package repos

import (
	"context"
	"time"
)

// models

type InternetRadioStation struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	StreamURL   string    `db:"stream_url"`
	Created     time.Time `db:"created"`
	Updated     time.Time `db:"updated"`
	User        string    `db:"user_name"`
	HomepageURL *string   `db:"homepage_url"`
}

// params

type CreateInternetRadioStationParams struct {
	Name        string
	StreamURL   string
	HomepageURL *string
}

type UpdateInternetRadioStationParams struct {
	Name        Optional[string]
	StreamURL   Optional[string]
	HomepageURL Optional[*string]
}

type InternetRadioStationRepository interface {
	FindAll(ctx context.Context, user string) ([]*InternetRadioStation, error)
	Create(ctx context.Context, user string, params CreateInternetRadioStationParams) (*InternetRadioStation, error)
	Update(ctx context.Context, user, id string, params UpdateInternetRadioStationParams) error
	Delete(ctx context.Context, user, id string) error
}

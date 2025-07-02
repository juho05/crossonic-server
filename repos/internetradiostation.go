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
	// Create creates a new internet radio station for the user.
	Create(ctx context.Context, user string, params CreateInternetRadioStationParams) (*InternetRadioStation, error)
	// FindAll returns all internet radio stations created by the user.
	FindAll(ctx context.Context, user string) ([]*InternetRadioStation, error)
	// Update updates an existing internet radio station of the user.
	// If no internet radio station with the provided id is found for the user, ErrNotFound will be returned.
	Update(ctx context.Context, user, id string, params UpdateInternetRadioStationParams) error
	// Delete removes an existing internet radio stations of the user.
	// If no internet radio station with the provided id is found for the user, ErrNotFound will be returned.
	Delete(ctx context.Context, user, id string) error
}

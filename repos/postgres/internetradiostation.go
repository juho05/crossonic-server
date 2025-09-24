package postgres

import (
	"context"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/nullism/bqb"
)

type internetRadioStationRepository struct {
	db executer
}

func (i internetRadioStationRepository) FindAll(ctx context.Context, user string) ([]*repos.InternetRadioStation, error) {
	q := bqb.New("SELECT * FROM internet_radio_stations WHERE user_name = ?", user)
	return selectQuery[*repos.InternetRadioStation](ctx, i.db, q)
}

func (i internetRadioStationRepository) Create(ctx context.Context, user string, params repos.CreateInternetRadioStationParams) (*repos.InternetRadioStation, error) {
	q := bqb.New(`INSERT INTO internet_radio_stations (id,name,stream_url,created,updated,user_name,homepage_url)
	VALUES (?,?,?,NOW(),NOW(),?,?) RETURNING *`, crossonic.GenIDInternetRadioStation(), params.Name, params.StreamURL, user, params.HomepageURL)
	return getQuery[*repos.InternetRadioStation](ctx, i.db, q)
}

func (i internetRadioStationRepository) Update(ctx context.Context, user, id string, params repos.UpdateInternetRadioStationParams) error {
	updateList, empty := genUpdateList(map[string]repos.OptionalGetter{
		"name":         params.Name,
		"stream_url":   params.StreamURL,
		"homepage_url": params.HomepageURL,
	}, true)
	if empty {
		return nil
	}
	q := bqb.New("UPDATE internet_radio_stations SET ? WHERE user_name = ? AND id = ?", updateList, user, id)
	return executeQueryExpectAffectedRows(ctx, i.db, q)
}

func (i internetRadioStationRepository) Delete(ctx context.Context, user, id string) error {
	q := bqb.New("DELETE FROM internet_radio_stations WHERE user_name = ? AND id = ?", user, id)
	return executeQueryExpectAffectedRows(ctx, i.db, q)
}

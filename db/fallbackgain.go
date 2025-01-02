package db

import (
	"context"

	"github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/log"
)

var fallbackGain *float64

func GetFallbackGain(ctx context.Context, store sqlc.Store) float64 {
	if fallbackGain == nil {
		gain, err := store.GetMedianReplayGain(ctx)
		if err != nil {
			log.Errorf("get fallback gain: %s, returning default -8", err)
			gain = float64(-8)
		}
		fGain := gain.(float64)
		if fGain == 0 {
			log.Warnf("get fallback gain: median gain is exactly 0 probably missing metadata, returning default -8")
			fGain = -8
		}
		fallbackGain = &fGain
	}
	return *fallbackGain
}

func InvalidateFallbackGain() {
	fallbackGain = nil
}

package repos

import (
	"sync"

	"github.com/juho05/log"
)

var fallbackGain *float64
var fallbackGainLock sync.RWMutex

const defaultFallbackGain = -8

func FallbackGain() float64 {
	fallbackGainLock.RLock()
	defer fallbackGainLock.RUnlock()
	if fallbackGain == nil {
		return defaultFallbackGain
	}
	return *fallbackGain
}

func SetFallbackGain(gain float64) {
	fallbackGainLock.Lock()
	defer fallbackGainLock.Unlock()
	if gain >= 0 {
		log.Warnf("trying to set fallbackGain >= 0: %d. Setting to nil instead", gain)
		fallbackGain = nil
		return
	}
	fallbackGain = &gain
}

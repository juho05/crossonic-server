package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newRateLimitTestHandler() *Handler {
	return &Handler{
		authFailures: make(map[string][]time.Time),
	}
}

func TestAuthRateLimit(t *testing.T) {
	t.Run("blocks after max fails", func(t *testing.T) {
		h := newRateLimitTestHandler()

		for i := 0; i < authRateMaxFails; i++ {
			limited, _ := h.isAuthRateLimited("alice")
			assert.False(t, limited, "should not be limited before reaching max (i=%d)", i)
			h.recordAuthFailure("alice")
		}

		limited, _ := h.isAuthRateLimited("alice")
		assert.True(t, limited, "should be limited after %d failures", authRateMaxFails)
	})

	t.Run("is per user", func(t *testing.T) {
		h := newRateLimitTestHandler()

		for i := 0; i < authRateMaxFails; i++ {
			h.recordAuthFailure("alice")
		}

		aliceLimited, _ := h.isAuthRateLimited("alice")
		bobLimited, _ := h.isAuthRateLimited("bob")
		assert.True(t, aliceLimited)
		assert.False(t, bobLimited, "limiting alice must not affect bob")
	})

	t.Run("expires after window", func(t *testing.T) {
		h := newRateLimitTestHandler()

		old := time.Now().Add(-authRateWindow - time.Second)
		failures := make([]time.Time, authRateMaxFails)
		for i := range failures {
			failures[i] = old
		}
		h.authFailures["alice"] = failures

		limited, _ := h.isAuthRateLimited("alice")
		assert.False(t, limited, "stale failures outside the window must not count")
	})

	t.Run("reports retry-after until oldest failure expires", func(t *testing.T) {
		h := newRateLimitTestHandler()

		// The oldest failure happened 2s ago, so the limit should lift in ~3s
		// (authRateWindow - 2s), since with exactly maxFails failures the count
		// drops below the threshold once that oldest one ages out.
		now := time.Now()
		h.authFailures["alice"] = []time.Time{
			now.Add(-2 * time.Second),
			now.Add(-1 * time.Second),
			now,
		}

		limited, retryAfter := h.isAuthRateLimited("alice")
		assert.True(t, limited)
		expected := authRateWindow - 2*time.Second
		assert.InDelta(t, expected.Seconds(), retryAfter.Seconds(), 0.5,
			"retry-after should be roughly the time until the oldest failure ages out")
	})

	t.Run("record prunes stale entries", func(t *testing.T) {
		h := newRateLimitTestHandler()

		old := time.Now().Add(-authRateWindow - time.Second)
		h.authFailures["alice"] = []time.Time{old, old, old, old, old}

		h.recordAuthFailure("alice")

		h.authFailuresLock.RLock()
		defer h.authFailuresLock.RUnlock()
		assert.Len(t, h.authFailures["alice"], 1, "stale entries should be pruned, leaving only the new one")
	})
}
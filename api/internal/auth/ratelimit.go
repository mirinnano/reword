package auth

import (
	"context"
	"time"

	"github.com/mirinnano/reword/api/internal/platform/store"
)

// Limiter is a fixed-window counter stored in MariaDB, so limits hold across
// API restarts and multiple instances.
type Limiter struct {
	Q   *store.Queries
	Now func() time.Time
}

type Limit struct {
	Name   string
	Max    uint32
	Window time.Duration
}

var (
	LimitLoginIP       = Limit{"login:ip", 30, 15 * time.Minute}
	LimitLoginEmail    = Limit{"login:email", 10, 15 * time.Minute}
	LimitSignupIP      = Limit{"signup:ip", 10, time.Hour}
	LimitSignupEmail   = Limit{"signup:email", 3, time.Hour}
	LimitResetIP       = Limit{"reset:ip", 10, time.Hour}
	LimitResetEmail    = Limit{"reset:email", 3, time.Hour}
	LimitTokenRedeemIP = Limit{"redeem:ip", 30, 15 * time.Minute}
)

// Allow records a hit for key under l and reports whether it is within the limit.
func (lm *Limiter) Allow(ctx context.Context, l Limit, key string) (bool, error) {
	bucket := l.Name + ":" + key
	if len(bucket) > 191 {
		bucket = bucket[:191]
	}
	start := lm.Now().UTC().Truncate(l.Window)
	if err := lm.Q.HitRateLimit(ctx, store.HitRateLimitParams{Bucket: bucket, WindowStart: start}); err != nil {
		return false, err
	}
	hits, err := lm.Q.GetRateLimitHits(ctx, store.GetRateLimitHitsParams{Bucket: bucket, WindowStart: start})
	if err != nil {
		return false, err
	}
	return hits <= l.Max, nil
}

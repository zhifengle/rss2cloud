package cloudfs

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type Limiter interface {
	Wait(ctx context.Context) error
}

type CooldownLimiter struct {
	min  time.Duration
	max  time.Duration
	rand *rand.Rand

	mu   sync.Mutex
	next time.Time
}

func NewCooldownLimiter(min, max time.Duration) *CooldownLimiter {
	if min < 0 {
		min = 0
	}
	if max <= 0 {
		max = min
	}
	if max < min {
		max = min
	}
	return &CooldownLimiter{
		min:  min,
		max:  max,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (l *CooldownLimiter) Wait(ctx context.Context) error {
	if l == nil || l.max <= 0 {
		return nil
	}

	l.mu.Lock()
	readyAt := l.next
	now := time.Now()
	if readyAt.Before(now) {
		readyAt = now
	}
	l.next = readyAt.Add(l.nextDelay())
	l.mu.Unlock()

	waitDuration := time.Until(readyAt)
	if waitDuration <= 0 {
		return nil
	}

	timer := time.NewTimer(waitDuration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (l *CooldownLimiter) nextDelay() time.Duration {
	if l == nil || l.max <= 0 {
		return 0
	}
	if l.max == l.min {
		return l.min
	}
	delta := l.max - l.min
	return time.Duration(l.rand.Int63n(int64(delta))) + l.min
}

type RateLimitedDriver struct {
	next    Driver
	limiter Limiter
}

func NewRateLimitedDriver(next Driver, limiter Limiter) Driver {
	if next == nil || limiter == nil {
		return next
	}
	return &RateLimitedDriver{
		next:    next,
		limiter: limiter,
	}
}

func (d *RateLimitedDriver) Provider() string {
	return d.next.Provider()
}

func (d *RateLimitedDriver) Root(ctx context.Context) (Entry, error) {
	return withLimit(ctx, d.limiter, func() (Entry, error) {
		return d.next.Root(ctx)
	})
}

func (d *RateLimitedDriver) Stat(ctx context.Context, entryID string) (Entry, error) {
	return withLimit(ctx, d.limiter, func() (Entry, error) {
		return d.next.Stat(ctx, entryID)
	})
}

func (d *RateLimitedDriver) List(ctx context.Context, dirID string) ([]Entry, error) {
	return withLimit(ctx, d.limiter, func() ([]Entry, error) {
		return d.next.List(ctx, dirID)
	})
}

func (d *RateLimitedDriver) Lookup(ctx context.Context, parentID, name string) (Entry, error) {
	return withLimit(ctx, d.limiter, func() (Entry, error) {
		return d.next.Lookup(ctx, parentID, name)
	})
}

func (d *RateLimitedDriver) Mkdir(ctx context.Context, parentID, name string) (Entry, error) {
	return withLimit(ctx, d.limiter, func() (Entry, error) {
		return d.next.Mkdir(ctx, parentID, name)
	})
}

func (d *RateLimitedDriver) Rename(ctx context.Context, entryID, newName string) (Entry, error) {
	return withLimit(ctx, d.limiter, func() (Entry, error) {
		return d.next.Rename(ctx, entryID, newName)
	})
}

func (d *RateLimitedDriver) Move(ctx context.Context, targetDirID, entryID string) (Entry, error) {
	return withLimit(ctx, d.limiter, func() (Entry, error) {
		return d.next.Move(ctx, targetDirID, entryID)
	})
}

func (d *RateLimitedDriver) Copy(ctx context.Context, targetDirID, entryID string) error {
	return waitLimit(ctx, d.limiter, func() error {
		return d.next.Copy(ctx, targetDirID, entryID)
	})
}

func (d *RateLimitedDriver) Delete(ctx context.Context, entryID string) error {
	return waitLimit(ctx, d.limiter, func() error {
		return d.next.Delete(ctx, entryID)
	})
}

func withLimit[T any](ctx context.Context, limiter Limiter, fn func() (T, error)) (T, error) {
	var zero T
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return zero, err
		}
	}
	return fn()
}

func waitLimit(ctx context.Context, limiter Limiter, fn func() error) error {
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return fn()
}

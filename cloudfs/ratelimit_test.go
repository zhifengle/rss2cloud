package cloudfs

import (
	"context"
	"errors"
	"testing"
	"time"
)

// countingLimiter counts how many times Wait is called.
type countingLimiter struct {
	count int
}

func (l *countingLimiter) Wait(_ context.Context) error {
	l.count++
	return nil
}

// minimalDriver is a no-op Driver for wrapping tests.
type minimalDriver struct {
	provider string
}

func (d *minimalDriver) Provider() string { return d.provider }
func (d *minimalDriver) Root(_ context.Context) (Entry, error) {
	return Entry{ID: "0", Type: EntryTypeDirectory}, nil
}
func (d *minimalDriver) Stat(_ context.Context, _ string) (Entry, error) {
	return Entry{ID: "1"}, nil
}
func (d *minimalDriver) List(_ context.Context, _ string) ([]Entry, error) {
	return []Entry{}, nil
}
func (d *minimalDriver) Lookup(_ context.Context, _, _ string) (Entry, error) {
	return Entry{ID: "1"}, nil
}
func (d *minimalDriver) Mkdir(_ context.Context, _, _ string) (Entry, error) {
	return Entry{ID: "2", Type: EntryTypeDirectory}, nil
}
func (d *minimalDriver) Rename(_ context.Context, _, _ string) (Entry, error) {
	return Entry{ID: "1", Name: "new"}, nil
}
func (d *minimalDriver) Move(_ context.Context, _, _ string) (Entry, error) {
	return Entry{ID: "1"}, nil
}
func (d *minimalDriver) Copy(_ context.Context, _, _ string) error  { return nil }
func (d *minimalDriver) Delete(_ context.Context, _ string) error   { return nil }

func TestRateLimitedDriverCallsLimiter(t *testing.T) {
	ctx := context.Background()
	limiter := &countingLimiter{}
	d := NewRateLimitedDriver(&minimalDriver{provider: "test"}, limiter)

	d.Root(ctx)
	d.Stat(ctx, "1")
	d.List(ctx, "0")
	d.Lookup(ctx, "0", "x")
	d.Mkdir(ctx, "0", "dir")
	d.Rename(ctx, "1", "new")
	d.Move(ctx, "0", "1")
	d.Copy(ctx, "0", "1")
	d.Delete(ctx, "1")

	if limiter.count != 9 {
		t.Fatalf("expected limiter called 9 times, got %d", limiter.count)
	}
}

func TestRateLimitedDriverContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	limiter := NewCooldownLimiter(100*time.Millisecond, 200*time.Millisecond)
	// Force next to be in the future so Wait actually blocks
	limiter.mu.Lock()
	limiter.next = time.Now().Add(10 * time.Second)
	limiter.mu.Unlock()

	d := NewRateLimitedDriver(&minimalDriver{}, limiter)

	_, err := d.Rename(ctx, "1", "new")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	err = d.Delete(ctx, "1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled on Delete, got %v", err)
	}
}

func TestNewRateLimitedDriverNilPassthrough(t *testing.T) {
	inner := &minimalDriver{provider: "x"}
	// nil limiter should return inner directly
	d := NewRateLimitedDriver(inner, nil)
	if d != Driver(inner) {
		t.Fatal("expected inner driver returned when limiter is nil")
	}
}

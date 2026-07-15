package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type XSyncSuite struct {
	test.Suite
}

func (s *XSyncSuite) TestErrGroupWithContext() {
	g, ctx := concurrency.ErrGroupWithContext(s.Ctx)
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
			return nil
		}
	})
	g.Go(func() error {
		return errors.New("boom")
	})
	err := g.Wait()
	s.Error(err)
	s.Equal("boom", err.Error())
}

func (s *XSyncSuite) TestNewWeighted() {
	sem := concurrency.NewWeighted(1)
	s.True(sem.TryAcquire(1))
	s.False(sem.TryAcquire(1))
	sem.Release(1)
	ctx, cancel := context.WithTimeout(s.Ctx, time.Second)
	defer cancel()
	s.NoError(sem.Acquire(ctx, 1))
	sem.Release(1)
}

func TestXSyncSuite(t *testing.T) {
	test.Run(t, new(XSyncSuite))
}

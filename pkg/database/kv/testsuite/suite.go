package testsuite

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// KVSuite is a reusable conformance suite for kv.KV implementations.
type KVSuite struct {
	*test.Suite
	Store kv.KV
	// Optional cleanup after each test.
	Cleanup func()
}

func (s *KVSuite) TearDownTest() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

func (s *KVSuite) TestGetSetDeleteExists() {
	ctx := context.Background()
	key := "conformance-key"
	value := []byte("hello-kv")

	err := s.Store.Set(ctx, key, value, 0)
	s.NoError(err)

	got, err := s.Store.Get(ctx, key)
	s.NoError(err)
	s.Equal(value, got)

	exists, err := s.Store.Exists(ctx, key)
	s.NoError(err)
	s.True(exists)

	err = s.Store.Delete(ctx, key)
	s.NoError(err)

	exists, err = s.Store.Exists(ctx, key)
	s.NoError(err)
	s.False(exists)

	_, err = s.Store.Get(ctx, key)
	s.Error(err)
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		s.Equal(errors.CodeNotFound, appErr.Code)
	} else {
		s.Fail("expected AppError for missing key")
	}
}

func (s *KVSuite) TestSetOverwrite() {
	ctx := context.Background()
	key := "overwrite-key"

	s.NoError(s.Store.Set(ctx, key, []byte("first"), 0))
	s.NoError(s.Store.Set(ctx, key, []byte("second"), 0))

	got, err := s.Store.Get(ctx, key)
	s.NoError(err)
	s.Equal([]byte("second"), got)
}

func (s *KVSuite) TestDeleteMissingKey() {
	ctx := context.Background()
	err := s.Store.Delete(ctx, "missing-key-never-set")
	s.NoError(err)
}

func (s *KVSuite) TestExistsMissingKey() {
	ctx := context.Background()
	exists, err := s.Store.Exists(ctx, "missing-key-never-set")
	s.NoError(err)
	s.False(exists)
}

func (s *KVSuite) TestTTLExpiration() {
	ctx := context.Background()
	key := "ttl-key"
	value := []byte("ephemeral")

	err := s.Store.Set(ctx, key, value, 50*time.Millisecond)
	s.NoError(err)

	got, err := s.Store.Get(ctx, key)
	s.NoError(err)
	s.Equal(value, got)

	exists, err := s.Store.Exists(ctx, key)
	s.NoError(err)
	s.True(exists)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		exists, err = s.Store.Exists(ctx, key)
		s.NoError(err)
		if !exists {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	exists, err = s.Store.Exists(ctx, key)
	s.NoError(err)
	s.False(exists)

	_, err = s.Store.Get(ctx, key)
	s.Error(err)
}

func (s *KVSuite) TestClose() {
	s.NoError(s.Store.Close())
	// Avoid double-close in TearDown when Cleanup is set.
	s.Cleanup = nil
}

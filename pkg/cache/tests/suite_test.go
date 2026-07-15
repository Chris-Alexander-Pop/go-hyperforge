package cache

import (
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type InvalidatePrefixSuite struct {
	test.Suite
	cache cache.Cache
}

func (s *InvalidatePrefixSuite) SetupTest() {
	s.Suite.SetupTest()
	s.cache = memory.New()
}

func (s *InvalidatePrefixSuite) TearDownTest() {
	_ = s.cache.Close()
}

func (s *InvalidatePrefixSuite) TestInvalidatePrefix() {
	_ = s.cache.Set(s.Ctx, "user:1", "a", time.Minute)
	_ = s.cache.Set(s.Ctx, "user:2", "b", time.Minute)
	_ = s.cache.Set(s.Ctx, "other:1", "c", time.Minute)

	n, err := cache.InvalidatePrefix(s.Ctx, s.cache, "user:")
	s.NoError(err)
	s.Equal(int64(2), n)

	ok, _ := s.cache.Exists(s.Ctx, "user:1")
	s.False(ok)
	ok, _ = s.cache.Exists(s.Ctx, "other:1")
	s.True(ok)
}

func (s *InvalidatePrefixSuite) TestInvalidatePrefixNil() {
	_, err := cache.InvalidatePrefix(s.Ctx, nil, "x")
	s.Error(err)
	s.True(strings.Contains(err.Error(), "nil"))
}

func TestInvalidatePrefixSuite(t *testing.T) {
	test.Run(t, new(InvalidatePrefixSuite))
}

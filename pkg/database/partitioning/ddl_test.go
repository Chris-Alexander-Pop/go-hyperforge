package partitioning

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type captureLogger struct {
	lastQuery string
}

func (c *captureLogger) LogMode(level logger.LogLevel) logger.Interface           { return c }
func (c *captureLogger) Info(ctx context.Context, s string, args ...interface{})  {}
func (c *captureLogger) Warn(ctx context.Context, s string, args ...interface{})  {}
func (c *captureLogger) Error(ctx context.Context, s string, args ...interface{}) {}
func (c *captureLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	c.lastQuery = sql
}

func TestCreateRangePartition_SQLInjection(t *testing.T) {
	capturer := &captureLogger{}

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		DryRun: true,
		Logger: capturer,
	})
	assert.NoError(t, err)

	table := "orders"
	// Malicious input trying to inject SQL
	start := "2023-01-01'); DROP TABLE orders; --"
	end := "2023-02-01"

	_ = CreateRangePartition(db, table, "created_at", start, end)

	// If vulnerable, the query contains: ... VALUES FROM ('2023-01-01'); DROP TABLE orders; --') ...
	// If secure, it should contain: ... VALUES FROM ('2023-01-01''); DROP TABLE orders; --') ...

	// Check if the single quote was escaped (doubled)
	assert.Contains(t, capturer.lastQuery, "2023-01-01''); DROP TABLE", "Expected escaped single quote, but found raw injection. Query: %s", capturer.lastQuery)
}

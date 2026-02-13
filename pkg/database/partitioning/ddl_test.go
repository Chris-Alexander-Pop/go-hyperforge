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
	CapturedSQL string
}

func (l *captureLogger) LogMode(level logger.LogLevel) logger.Interface { return l }
func (l *captureLogger) Info(ctx context.Context, s string, args ...interface{}) {}
func (l *captureLogger) Warn(ctx context.Context, s string, args ...interface{}) {}
func (l *captureLogger) Error(ctx context.Context, s string, args ...interface{}) {}
func (l *captureLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	l.CapturedSQL = sql
}

func TestCreateRangePartition_SQLInjection(t *testing.T) {
	l := &captureLogger{}

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DryRun: true,
		Logger: l,
	})
	assert.NoError(t, err)

	maliciousStart := "2023-01-01'); DROP TABLE users; --"
	maliciousEnd := "2023-02-01"

	err = CreateRangePartition(db, "orders", "created_at", maliciousStart, maliciousEnd)
	// CreateRangePartition might execute SQL. In DryRun, it prepares it.

	// The injection payload
	// Input: 2023-01-01'); DROP TABLE users; --
	//
	// If VULNERABLE, SQL contains: ... FROM ('2023-01-01'); DROP TABLE users; --') ...
	// If SECURE, SQL contains:     ... FROM ('2023-01-01''); DROP TABLE users; --') ...

	// Verify that the single quote was escaped (doubled)
	expectedEscaped := "2023-01-01''); DROP TABLE users; --"
	assert.Contains(t, l.CapturedSQL, expectedEscaped, "SQL should contain escaped payload")

	// Verify that the unescaped payload (which would close the string literal) is NOT present
	unexpectedUnescaped := "'2023-01-01');"
	assert.NotContains(t, l.CapturedSQL, unexpectedUnescaped, "SQL should NOT contain unescaped payload")
}

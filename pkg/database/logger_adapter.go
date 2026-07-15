package database

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	gormlogger "gorm.io/gorm/logger"
)

// GORMLogger adapts our structured logger to GORM's interface
type GORMLogger struct {
	LogLevel gormlogger.LogLevel
}

func NewGORMLogger() *GORMLogger {
	return &GORMLogger{LogLevel: gormlogger.Info}
}

func (l *GORMLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *GORMLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		logger.L().InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *GORMLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		logger.L().WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *GORMLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		logger.L().ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *GORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && l.LogLevel >= gormlogger.Error {
		logger.L().ErrorContext(ctx, "sql query executed", "sql", sql, "rows", rows, "elapsed", elapsed, "error", err)
		return
	}

	if l.LogLevel >= gormlogger.Info {
		logger.L().InfoContext(ctx, "sql query executed", "sql", sql, "rows", rows, "elapsed", elapsed)
	}
}

// Package scheduler provides distributed job scheduling.
//
// Features:
//   - Cron-based scheduling via robfig/cron (5-field, @hourly/@daily/@weekly, @every)
//   - One-time delayed jobs
//   - Optional distributed locking via pkg/concurrency/distlock for single execution
//   - Optional job persistence via Store (MemoryStore for tests/single-node)
//
// Usage:
//
//	store := scheduler.NewMemoryStore()
//	locker := distlockmemory.New() // or nil for single-node
//	sched := scheduler.New(store, locker)
//	sched.Schedule("daily-report", "0 0 * * *", generateReportJob)
//	sched.Start(ctx)
package scheduler

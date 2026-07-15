// Package billing provides subscription and invoicing logic.
//
// The memory adapter ships a small plan catalog and supports UpgradeSubscription
// (immediate amount swap; proration is a stub) and MarkPastDue for dunning.
// Amounts use commerce.Money (int64 minor units). StatusPastDue is set via
// MarkPastDue after failed invoice collection.
package billing

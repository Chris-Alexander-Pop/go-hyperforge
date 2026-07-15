// Package billing provides subscription and invoicing logic.
//
// The memory adapter ships a small plan catalog, mid-cycle Prorate on
// UpgradeSubscription (issues a net proration invoice when owed), MarkPastDue,
// and ProcessDunning (open invoices → past_due + subscription past_due).
// Amounts use commerce.Money (int64 minor units).
package billing

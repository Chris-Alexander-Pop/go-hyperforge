package transfer

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors" // Import our error package
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConflictStrategy defines how to handle data conflicts
type ConflictStrategy string

const (
	ConflictStrategyNone      ConflictStrategy = ""           // Standard Insert (Fail on conflict)
	ConflictStrategyDoNothing ConflictStrategy = "do_nothing" // INSERT IGNORE / ON CONFLICT DO NOTHING
	ConflictStrategyUpdateAll ConflictStrategy = "update_all" // ON CONFLICT UPDATE SET ...
)

// TransferOptions configures the CopyTable operation
type TransferOptions struct {
	BatchSize  int
	OnConflict ConflictStrategy
}

// CopyTable transfers data from source to destination for a given model.
// It uses batch processing to avoid memory overflow.
func CopyTable(ctx context.Context, src *gorm.DB, dst *gorm.DB, model interface{}, opts TransferOptions) error {
	// 1. Ensure Table Exists in Dst (AutoMigrate)
	if err := dst.AutoMigrate(model); err != nil {
		return errors.Wrap(err, "failed to migrate destination table")
	}

	// 2. Read and Write in Batches
	var rows []map[string]interface{}
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	result := src.WithContext(ctx).Model(model).FindInBatches(&rows, batchSize, func(tx *gorm.DB, batch int) error {
		txDst := dst.WithContext(ctx).Model(model)

		// Apply Conflict Strategy
		switch opts.OnConflict {
		case ConflictStrategyDoNothing:
			txDst = txDst.Clauses(clause.OnConflict{DoNothing: true})
		case ConflictStrategyUpdateAll:
			txDst = txDst.Clauses(clause.OnConflict{UpdateAll: true})
		}

		if err := txDst.Create(&rows).Error; err != nil {
			return errors.Wrap(err, "failed to insert batch")
		}
		return nil
	})

	return result.Error
}

// MigrateSchema copies the schema structure from source model to destination.
func MigrateSchema(ctx context.Context, dst *gorm.DB, models ...interface{}) error {
	return dst.WithContext(ctx).AutoMigrate(models...)
}

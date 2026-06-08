package cosmosdb

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCosmosDBInjection(t *testing.T) {
	query := map[string]interface{}{
		"1=1; DROP TABLE c; --": "test",
	}

	_, _, err := buildQuery(query)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid query key")
}

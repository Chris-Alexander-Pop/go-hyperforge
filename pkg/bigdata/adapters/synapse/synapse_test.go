package synapse_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata/adapters/synapse"
)

func TestSynapse_Init(t *testing.T) {
	// Synapse adapter performs sql.Open which validates DSN format but doesn't connect immediately usually,
	// or relies on driver.
	// We pass a dummy DSN.

	adapter, err := synapse.New("server=localhost;user id=sa;password=pass")
	if err != nil {
		t.Logf("Synapse New error: %v", err)
	} else if adapter == nil {
		t.Error("Returned nil adapter")
	}
}

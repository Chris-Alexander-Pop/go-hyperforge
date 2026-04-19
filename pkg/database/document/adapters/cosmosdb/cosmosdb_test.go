package cosmosdb

import (
	"testing"
)

func TestBuildQuery_KeyValidation(t *testing.T) {
	tests := []struct {
		name    string
		query   map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid keys",
			query: map[string]interface{}{
				"id":    "123",
				"name":  "test",
				"a.b_c": "value",
			},
			wantErr: false,
		},
		{
			name: "invalid key with space",
			query: map[string]interface{}{
				"invalid key": "value",
			},
			wantErr: true,
		},
		{
			name: "invalid key with sql injection attempt",
			query: map[string]interface{}{
				"id = '1' OR 1=1 --": "value",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := buildQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

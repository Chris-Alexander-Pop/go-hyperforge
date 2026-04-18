package cosmosdb

import (
	"testing"
)

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     map[string]interface{}
		wantError bool
	}{
		{
			name: "valid query keys",
			query: map[string]interface{}{
				"id":     "123",
				"status": "active",
				"age":    30,
			},
			wantError: false,
		},
		{
			name: "invalid query key with injection attempt",
			query: map[string]interface{}{
				"id OR 1=1": "123",
			},
			wantError: true,
		},
		{
			name: "invalid query key with comment",
			query: map[string]interface{}{
				"id --": "123",
			},
			wantError: true,
		},
		{
			name: "invalid query key with semicolon",
			query: map[string]interface{}{
				"id; DROP TABLE c;": "123",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := buildQuery(tt.query)
			if (err != nil) != tt.wantError {
				t.Errorf("buildQuery() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

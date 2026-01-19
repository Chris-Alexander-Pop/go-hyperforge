package avro_test

import (
	"bytes"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata/formats/avro"
)

type User struct {
	ID   int    `avro:"id"`
	Name string `avro:"name"`
}

func TestAvroReadWrite(t *testing.T) {
	schema := `{
		"type": "record",
		"name": "user",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "name", "type": "string"}
		]
	}`

	buf := &bytes.Buffer{}

	// Write
	writer, err := avro.NewWriter(buf, schema)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	for _, u := range users {
		if err := writer.Write(u); err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
	}
	writer.Close()

	// Read
	reader, err := avro.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	var readUsers []User
	var u User
	for reader.Read(&u) == nil {
		readUsers = append(readUsers, u)
		u = User{} // reset
	}

	if len(readUsers) != 2 {
		t.Errorf("Expected 2 users, got %d", len(readUsers))
	}
	if readUsers[0].Name != "Alice" {
		t.Errorf("Expected Alice, got %s", readUsers[0].Name)
	}
}

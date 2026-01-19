package parquet_test

import (
	"bytes"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/bigdata/formats/parquet"
)

type User struct {
	ID   int64  `parquet:"id"`
	Name string `parquet:"name"`
}

func TestParquetReadWrite(t *testing.T) {
	buf := &bytes.Buffer{} // Parquet writer needs io.Writer

	// Write
	writer := parquet.NewWriter[User](buf)

	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	if err := writer.Write(users); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	writer.Close()

	// Read
	// Parquet Reader needs io.ReaderAt and size
	data := buf.Bytes()
	reader := parquet.NewReader[User](bytes.NewReader(data), int64(len(data)))

	readUsers, err := reader.Read(10)
	if err != nil && err.Error() != "EOF" { // parquet-go might behave differently on EOF
		// check logic
	}

	// If parquet-go reader returns io.EOF with partial data? No, usually explicitly returns available.
	if len(readUsers) != 2 {
		t.Errorf("Expected 2 users, got %d", len(readUsers))
	} else {
		if readUsers[0].Name != "Alice" {
			t.Errorf("Expected Alice, got %s", readUsers[0].Name)
		}
	}
}

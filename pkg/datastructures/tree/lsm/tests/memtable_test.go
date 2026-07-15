package lsm_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/tree/lsm"
)

import "testing"

func TestMemTable(t *testing.T) {
	mt := lsm.New(1024)

	if !mt.Put("key1", []byte("value1")) {
		t.Error("Failed to put key1")
	}

	if val, found := mt.Get("key1"); !found || string(val) != "value1" {
		t.Error("Get key1 failed")
	}

	mt.Delete("key1")
	if _, found := mt.Get("key1"); found {
		t.Error("Expected key1 to be deleted")
	}

	// Test full
	mtSmall := lsm.New(10) // Small cap
	if !mtSmall.Put("longkey", []byte("longvalue")) {
		// Should fail
	} else {
		// Or pass if exact fit? 7+9 = 16 > 10. Should fail.
		t.Error("Expected put to fail due to capacity")
	}
}

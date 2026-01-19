package bk

import (
	"sort"
	"testing"
)

func TestBKTree(t *testing.T) {
	tree := New()
	tree.Add("book")
	tree.Add("books")
	tree.Add("cake")
	tree.Add("boo")

	// 'boos' distance to 'book' is 1 (s)
	// 'boos' distance to 'books' is 1 (k->s or del s) -> 'books' len 5, 'boos' len 4. 'book' -> 'boos' (k->s).
	// Levenshtein:
	// book, boos -> 1 sub (k->s)
	// books, boos -> 1 sub (k->s) ? no. books -> boos (del k).

	matches := tree.Search("boos", 1)
	sort.Strings(matches)

	// Expect matches: book (1), books (1), boo (1 add s).
	// cake (dist large).

	// Check coverage
	found := make(map[string]bool)
	for _, m := range matches {
		found[m] = true
	}

	if !found["book"] {
		t.Error("Expected to find book")
	}
	if !found["books"] { // books -> boos: del k. dist=1.
		t.Error("Expected to find books")
	}
	if !found["boo"] { // boo -> boos: add s. dist=1.
		t.Error("Expected to find boo")
	}
}

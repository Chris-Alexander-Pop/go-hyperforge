package sql

import (
	"fmt"
	"strings"
)

// rewrite converts ? placeholders to $1, $2, ... for PostgreSQL.
func rewrite(dialect Dialect, query string) string {
	if dialect != DialectPostgres {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

package snowflake

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/bigdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	sql.Register("snowflake-test", &fakeDriver{})
}

type fakeDriver struct{}

func (d *fakeDriver) Open(name string) (driver.Conn, error) {
	return &fakeConn{}, nil
}

type fakeConn struct{}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) { return &fakeStmt{query: query}, nil }
func (c *fakeConn) Close() error                              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)                 { return nil, driver.ErrSkip }

type fakeStmt struct {
	query string
}

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return 0 }
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &fakeRows{
		cols: []string{"ID", "NAME"},
		data: [][]driver.Value{{int64(1), "alpha"}},
	}, nil
}

type fakeRows struct {
	cols []string
	data [][]driver.Value
	i    int
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}

func TestSQLAdapter(t *testing.T) {
	a, err := NewSQLWithDriver("snowflake-test", "test-dsn")
	require.NoError(t, err)
	defer a.Close()

	res, err := a.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, int64(1), res.Rows[0]["ID"])
	assert.Equal(t, "alpha", res.Rows[0]["NAME"])
	assert.Equal(t, "snowflake-sql", res.Metadata["source"])
}

func TestNewFromDB(t *testing.T) {
	db, err := sql.Open("snowflake-test", "x")
	require.NoError(t, err)
	a, err := NewFromDB(db)
	require.NoError(t, err)
	require.NoError(t, a.Close()) // does not close owned=false
	require.NoError(t, db.Close())
}

func TestHTTPAdapter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/statements", r.URL.Path)
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"statementHandle": "abc",
			"resultSetMetaData": map[string]interface{}{
				"numRows": 1,
				"rowType": []map[string]string{{"name": "C"}},
			},
			"data": [][]interface{}{{"hello"}},
		})
	}))
	defer srv.Close()

	a, err := NewHTTP(Config{
		AccountURL: srv.URL,
		Token:      "tok",
		HTTPClient: srv.Client(),
	})
	require.NoError(t, err)
	res, err := a.Query(context.Background(), "SELECT 'hello'")
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "hello", res.Rows[0]["C"])
	assert.Equal(t, "snowflake-http", res.Metadata["source"])
}

func TestClosed(t *testing.T) {
	a, err := NewSQLWithDriver("snowflake-test", "x")
	require.NoError(t, err)
	require.NoError(t, a.Close())
	_, err = a.Query(context.Background(), "SELECT 1")
	assert.True(t, bigdata.IsClosed(err))
}

func TestHTTPRequiresConfig(t *testing.T) {
	_, err := NewHTTP(Config{})
	require.Error(t, err)
	_, err = NewHTTP(Config{AccountURL: "https://x"})
	require.Error(t, err)
}

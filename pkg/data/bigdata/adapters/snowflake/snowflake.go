// Package snowflake provides a thin Snowflake warehouse adapter implementing bigdata.Client.
//
// Two backends are supported:
//   - SQL: database/sql with a registered Snowflake driver (blank-import snowflakedb/gosnowflake
//     in your main package). Use NewSQL / NewFromDB.
//   - HTTP: Snowflake SQL REST API (POST /api/v2/statements). Use NewHTTP for tests or
//     environments without the native driver.
package snowflake

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/bigdata"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Ensure Adapter implements bigdata.Client.
var _ bigdata.Client = (*Adapter)(nil)

// Config configures the Snowflake adapter.
type Config struct {
	// DSN is the database/sql DSN (for SQL mode).
	DSN string

	// Driver is the sql driver name (default "snowflake").
	Driver string

	// AccountURL is the Snowflake account base URL for HTTP SQL API
	// (e.g. https://xy12345.us-east-1.snowflakecomputing.com).
	AccountURL string

	// Token is a bearer JWT / session token for the HTTP SQL API.
	Token string

	// Database / Schema / Warehouse / Role are optional session context for HTTP.
	Database  string
	Schema    string
	Warehouse string
	Role      string

	// HTTPClient overrides the HTTP client (tests).
	HTTPClient *http.Client
}

// Adapter is a thin Snowflake bigdata.Client.
type Adapter struct {
	db     *sql.DB
	owned  bool // close db on Close when opened by NewSQL
	http   *http.Client
	cfg    Config
	closed bool
}

// NewSQL opens a Snowflake connection via database/sql.
// The snowflake driver must be registered by the caller (blank import).
func NewSQL(dsn string) (*Adapter, error) {
	return NewSQLWithDriver("snowflake", dsn)
}

// NewSQLWithDriver opens database/sql with an explicit driver name.
func NewSQLWithDriver(driver, dsn string) (*Adapter, error) {
	if dsn == "" {
		return nil, errors.InvalidArgument("snowflake DSN is required", nil)
	}
	if driver == "" {
		driver = "snowflake"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, bigdata.ErrConnectionFailed("snowflake", err)
	}
	return &Adapter{db: db, owned: true, cfg: Config{DSN: dsn, Driver: driver}}, nil
}

// NewFromDB wraps an existing *sql.DB (caller owns Close).
func NewFromDB(db *sql.DB) (*Adapter, error) {
	if db == nil {
		return nil, errors.InvalidArgument("sql.DB is nil", nil)
	}
	return &Adapter{db: db, owned: false}, nil
}

// NewHTTP creates an HTTP SQL API client (no native driver required).
func NewHTTP(cfg Config) (*Adapter, error) {
	if cfg.AccountURL == "" {
		return nil, errors.InvalidArgument("AccountURL is required for HTTP mode", nil)
	}
	if cfg.Token == "" {
		return nil, errors.InvalidArgument("Token is required for HTTP mode", nil)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Adapter{http: client, cfg: cfg}, nil
}

// Query executes a SQL statement and returns rows as maps.
func (a *Adapter) Query(ctx context.Context, query string, args ...interface{}) (*bigdata.Result, error) {
	if a == nil || a.closed {
		return nil, bigdata.ErrClosed
	}
	if a.db != nil {
		return a.querySQL(ctx, query, args...)
	}
	if a.http != nil {
		if len(args) > 0 {
			return nil, bigdata.ErrInvalidQuery("HTTP mode does not support positional args; interpolate or use SQL mode", nil)
		}
		return a.queryHTTP(ctx, query)
	}
	return nil, bigdata.ErrConnectionFailed("snowflake", errors.Internal("no backend configured", nil))
}

func (a *Adapter) querySQL(ctx context.Context, query string, args ...interface{}) (*bigdata.Result, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, bigdata.ErrQueryFailed(query, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, bigdata.ErrQueryFailed(query, err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, bigdata.ErrQueryFailed(query, err)
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, bigdata.ErrQueryFailed(query, err)
	}
	return &bigdata.Result{
		Rows:     results,
		Metadata: map[string]interface{}{"source": "snowflake-sql"},
	}, nil
}

type httpStatementRequest struct {
	Statement  string                 `json:"statement"`
	Timeout    int                    `json:"timeout,omitempty"`
	Database   string                 `json:"database,omitempty"`
	Schema     string                 `json:"schema,omitempty"`
	Warehouse  string                 `json:"warehouse,omitempty"`
	Role       string                 `json:"role,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type httpStatementResponse struct {
	StatementHandle   string `json:"statementHandle"`
	ResultSetMetaData struct {
		NumRows int `json:"numRows"`
		RowType []struct {
			Name string `json:"name"`
		} `json:"rowType"`
	} `json:"resultSetMetaData"`
	Data    [][]interface{} `json:"data"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
}

func (a *Adapter) queryHTTP(ctx context.Context, query string) (*bigdata.Result, error) {
	body := httpStatementRequest{
		Statement: query,
		Timeout:   60,
		Database:  a.cfg.Database,
		Schema:    a.cfg.Schema,
		Warehouse: a.cfg.Warehouse,
		Role:      a.cfg.Role,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, bigdata.ErrInvalidQuery("marshal", err)
	}
	url := strings.TrimRight(a.cfg.AccountURL, "/") + "/api/v2/statements"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, bigdata.ErrConnectionFailed("snowflake", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.Token)

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, bigdata.ErrConnectionFailed("snowflake", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, bigdata.ErrQueryFailed(query, err)
	}
	if resp.StatusCode >= 400 {
		return nil, bigdata.ErrQueryFailed(query, fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody)))
	}

	var parsed httpStatementResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, bigdata.ErrQueryFailed(query, err)
	}
	if parsed.Message != "" && parsed.Code != "" && parsed.Code != "090001" && len(parsed.Data) == 0 && len(parsed.ResultSetMetaData.RowType) == 0 {
		// Some error payloads include code/message without rows.
		if resp.StatusCode >= 300 {
			return nil, bigdata.ErrQueryFailed(query, errors.Internal(parsed.Message, nil))
		}
	}

	cols := make([]string, len(parsed.ResultSetMetaData.RowType))
	for i, rt := range parsed.ResultSetMetaData.RowType {
		cols[i] = rt.Name
	}
	rows := make([]map[string]interface{}, 0, len(parsed.Data))
	for _, dataRow := range parsed.Data {
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			if i < len(dataRow) {
				row[col] = dataRow[i]
			}
		}
		rows = append(rows, row)
	}
	return &bigdata.Result{
		Rows: rows,
		Metadata: map[string]interface{}{
			"source":          "snowflake-http",
			"statementHandle": parsed.StatementHandle,
		},
	}, nil
}

// Close closes the owned SQL connection (if any). HTTP mode is a no-op.
func (a *Adapter) Close() error {
	if a == nil || a.closed {
		return nil
	}
	a.closed = true
	if a.owned && a.db != nil {
		return a.db.Close()
	}
	return nil
}

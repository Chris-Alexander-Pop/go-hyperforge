// Package postgres provides a ControlPlane backed by database/sql.
//
// Hosts and instances are stored as JSON rows. Call Migrate before use.
// Tests typically use modernc.org/sqlite with DialectSQLite; production uses
// DialectPostgres with lib/pq or pgx stdlib.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Dialect selects SQL placeholder style.
type Dialect int

const (
	// DialectSQLite uses ? placeholders.
	DialectSQLite Dialect = iota
	// DialectPostgres uses $1, $2, ... placeholders.
	DialectPostgres
)

// Config configures the postgres/sql control-plane adapter.
type Config struct {
	Dialect Dialect
}

// ControlPlane persists host/instance inventory via database/sql.
type ControlPlane struct {
	db      *sql.DB
	dialect Dialect
}

var _ controlplane.ControlPlane = (*ControlPlane)(nil)

// New wraps an existing *sql.DB. Call Migrate before use.
func New(db *sql.DB, cfg Config) (*ControlPlane, error) {
	if db == nil {
		return nil, pkgerrors.InvalidArgument("db is required", nil)
	}
	return &ControlPlane{db: db, dialect: cfg.Dialect}, nil
}

func (c *ControlPlane) rewrite(query string) string {
	if c.dialect != DialectPostgres {
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

// Migrate creates inventory tables if missing.
func (c *ControlPlane) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS cp_hosts (
			id TEXT PRIMARY KEY,
			payload TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cp_instances (
			id TEXT PRIMARY KEY,
			host_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			payload TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cp_instances_host ON cp_instances(host_id)`,
	}
	for _, stmt := range stmts {
		if _, err := c.db.ExecContext(ctx, stmt); err != nil {
			return pkgerrors.Internal("controlplane migrate failed", err)
		}
	}
	return nil
}

func (c *ControlPlane) saveHost(ctx context.Context, h cloud.Host) error {
	raw, err := json.Marshal(h)
	if err != nil {
		return pkgerrors.Internal("failed to encode host", err)
	}
	_, err = c.db.ExecContext(ctx, c.rewrite(`
INSERT INTO cp_hosts (id, payload) VALUES (?, ?)
ON CONFLICT(id) DO UPDATE SET payload = excluded.payload
`), h.ID, string(raw))
	if err != nil {
		return pkgerrors.Internal("failed to save host", err)
	}
	return nil
}

func (c *ControlPlane) loadHost(ctx context.Context, hostID string) (cloud.Host, error) {
	var payload string
	err := c.db.QueryRowContext(ctx, c.rewrite(`SELECT payload FROM cp_hosts WHERE id = ?`), hostID).Scan(&payload)
	if err == sql.ErrNoRows {
		return cloud.Host{}, controlplane.ErrHostNotFound
	}
	if err != nil {
		return cloud.Host{}, pkgerrors.Internal("failed to load host", err)
	}
	var h cloud.Host
	if err := json.Unmarshal([]byte(payload), &h); err != nil {
		return cloud.Host{}, pkgerrors.Internal("failed to decode host", err)
	}
	return h, nil
}

func (c *ControlPlane) saveInstance(ctx context.Context, inst controlplane.Instance) error {
	raw, err := json.Marshal(inst)
	if err != nil {
		return pkgerrors.Internal("failed to encode instance", err)
	}
	_, err = c.db.ExecContext(ctx, c.rewrite(`
INSERT INTO cp_instances (id, host_id, status, payload) VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET host_id = excluded.host_id, status = excluded.status, payload = excluded.payload
`), inst.ID, inst.HostID, string(inst.Status), string(raw))
	if err != nil {
		return pkgerrors.Internal("failed to save instance", err)
	}
	return nil
}

func (c *ControlPlane) loadInstance(ctx context.Context, id string) (controlplane.Instance, error) {
	var payload string
	err := c.db.QueryRowContext(ctx, c.rewrite(`SELECT payload FROM cp_instances WHERE id = ?`), id).Scan(&payload)
	if err == sql.ErrNoRows {
		return controlplane.Instance{}, controlplane.ErrInstanceNotFound
	}
	if err != nil {
		return controlplane.Instance{}, pkgerrors.Internal("failed to load instance", err)
	}
	var inst controlplane.Instance
	if err := json.Unmarshal([]byte(payload), &inst); err != nil {
		return controlplane.Instance{}, pkgerrors.Internal("failed to decode instance", err)
	}
	return inst, nil
}

// RegisterHost adds a host to inventory.
func (c *ControlPlane) RegisterHost(ctx context.Context, host cloud.Host) error {
	if host.ID == "" {
		return pkgerrors.InvalidArgument("host id is required", nil)
	}
	if _, err := c.loadHost(ctx, host.ID); err == nil {
		return controlplane.ErrHostAlreadyRegistered
	} else if err != controlplane.ErrHostNotFound {
		return err
	}
	if host.Available.VCPUs == 0 && host.Available.MemoryMB == 0 && host.Available.DiskGB == 0 {
		host.Available = host.Capacity
	}
	return c.saveHost(ctx, host)
}

// DeregisterHost removes a host when it has no bound instances.
func (c *ControlPlane) DeregisterHost(ctx context.Context, hostID string) error {
	if _, err := c.loadHost(ctx, hostID); err != nil {
		return err
	}
	insts, err := c.ListInstances(ctx, controlplane.ListInstancesOptions{HostID: hostID})
	if err != nil {
		return err
	}
	if len(insts) > 0 {
		return controlplane.ErrHostHasInstances
	}
	_, err = c.db.ExecContext(ctx, c.rewrite(`DELETE FROM cp_hosts WHERE id = ?`), hostID)
	if err != nil {
		return pkgerrors.Internal("failed to delete host", err)
	}
	return nil
}

// UpdateHostStatus updates a registered host's status.
func (c *ControlPlane) UpdateHostStatus(ctx context.Context, hostID string, status cloud.HostStatus) error {
	h, err := c.loadHost(ctx, hostID)
	if err != nil {
		return err
	}
	h.Status = status
	return c.saveHost(ctx, h)
}

// GetHost retrieves a host by ID.
func (c *ControlPlane) GetHost(ctx context.Context, hostID string) (*cloud.Host, error) {
	h, err := c.loadHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// ListHosts returns all registered hosts.
func (c *ControlPlane) ListHosts(ctx context.Context) ([]cloud.Host, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT payload FROM cp_hosts`)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list hosts", err)
	}
	defer rows.Close()
	out := make([]cloud.Host, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, pkgerrors.Internal("failed to scan host", err)
		}
		var h cloud.Host
		if err := json.Unmarshal([]byte(payload), &h); err != nil {
			return nil, pkgerrors.Internal("failed to decode host", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// CreateInstance registers an instance, optionally binding to a host.
func (c *ControlPlane) CreateInstance(ctx context.Context, req controlplane.CreateInstanceRequest) (*controlplane.Instance, error) {
	if req.Name == "" {
		return nil, pkgerrors.InvalidArgument("instance name is required", nil)
	}
	inst := controlplane.Instance{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Status:    cloud.InstanceStatusPending,
		Resources: req.Resources,
		Image:     req.Image,
		Tags:      req.Tags,
		CreatedAt: time.Now().UTC(),
	}
	if req.HostID != "" {
		if err := c.bind(ctx, &inst, req.HostID); err != nil {
			return nil, err
		}
		inst.Status = cloud.InstanceStatusProvisioning
	}
	if err := c.saveInstance(ctx, inst); err != nil {
		return nil, err
	}
	out := inst
	return &out, nil
}

// BindInstance assigns an unbound instance to a host.
func (c *ControlPlane) BindInstance(ctx context.Context, instanceID, hostID string) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.HostID != "" {
		return controlplane.ErrInstanceAlreadyBound
	}
	if err := c.bind(ctx, &inst, hostID); err != nil {
		return err
	}
	if inst.Status == cloud.InstanceStatusPending {
		inst.Status = cloud.InstanceStatusProvisioning
	}
	return c.saveInstance(ctx, inst)
}

// UnbindInstance detaches an instance from its host.
func (c *ControlPlane) UnbindInstance(ctx context.Context, instanceID string) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.HostID == "" {
		return controlplane.ErrInstanceNotBound
	}
	if err := c.release(ctx, inst); err != nil {
		return err
	}
	inst.HostID = ""
	return c.saveInstance(ctx, inst)
}

// GetInstance retrieves an instance by ID.
func (c *ControlPlane) GetInstance(ctx context.Context, instanceID string) (*controlplane.Instance, error) {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// ListInstances returns instances filtered by options.
func (c *ControlPlane) ListInstances(ctx context.Context, opts controlplane.ListInstancesOptions) ([]controlplane.Instance, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT payload FROM cp_instances`)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list instances", err)
	}
	defer rows.Close()
	out := make([]controlplane.Instance, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, pkgerrors.Internal("failed to scan instance", err)
		}
		var inst controlplane.Instance
		if err := json.Unmarshal([]byte(payload), &inst); err != nil {
			return nil, pkgerrors.Internal("failed to decode instance", err)
		}
		if opts.HostID != "" && inst.HostID != opts.HostID {
			continue
		}
		if opts.Status != "" && inst.Status != opts.Status {
			continue
		}
		out = append(out, inst)
	}
	return out, rows.Err()
}

// DeleteInstance removes an instance and releases capacity.
func (c *ControlPlane) DeleteInstance(ctx context.Context, instanceID string) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.HostID != "" {
		if err := c.release(ctx, inst); err != nil {
			return err
		}
	}
	_, err = c.db.ExecContext(ctx, c.rewrite(`DELETE FROM cp_instances WHERE id = ?`), instanceID)
	if err != nil {
		return pkgerrors.Internal("failed to delete instance", err)
	}
	return nil
}

// UpdateInstanceStatus updates instance lifecycle status.
func (c *ControlPlane) UpdateInstanceStatus(ctx context.Context, instanceID string, status cloud.InstanceStatus) error {
	inst, err := c.loadInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	inst.Status = status
	return c.saveInstance(ctx, inst)
}

func (c *ControlPlane) bind(ctx context.Context, inst *controlplane.Instance, hostID string) error {
	host, err := c.loadHost(ctx, hostID)
	if err != nil {
		return err
	}
	if host.Status != cloud.HostStatusReady && host.Status != cloud.HostStatusBusy {
		return controlplane.ErrHostNotReady
	}
	res := inst.Resources
	if host.Available.VCPUs < res.VCPUs || host.Available.MemoryMB < res.MemoryMB || host.Available.DiskGB < res.DiskGB {
		return controlplane.ErrHostCapacityExhausted
	}
	host.Available.VCPUs -= res.VCPUs
	host.Available.MemoryMB -= res.MemoryMB
	host.Available.DiskGB -= res.DiskGB
	host.Available.GPUs -= res.GPUs
	if host.Available.VCPUs == 0 || host.Available.MemoryMB == 0 {
		host.Status = cloud.HostStatusBusy
	}
	if err := c.saveHost(ctx, host); err != nil {
		return err
	}
	inst.HostID = hostID
	return nil
}

func (c *ControlPlane) release(ctx context.Context, inst controlplane.Instance) error {
	host, err := c.loadHost(ctx, inst.HostID)
	if err != nil {
		if err == controlplane.ErrHostNotFound {
			return nil
		}
		return err
	}
	host.Available.VCPUs += inst.Resources.VCPUs
	host.Available.MemoryMB += inst.Resources.MemoryMB
	host.Available.DiskGB += inst.Resources.DiskGB
	host.Available.GPUs += inst.Resources.GPUs
	if host.Status == cloud.HostStatusBusy {
		host.Status = cloud.HostStatusReady
	}
	return c.saveHost(ctx, host)
}

package gce

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/vm"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// Config holds GCE adapter configuration.
type Config struct {
	ProjectID       string `env:"VM_GCP_PROJECT"`
	Zone            string `env:"VM_GCP_ZONE" env-default:"us-central1-a"`
	CredentialsFile string `env:"VM_GCP_CREDENTIALS"`
	DefaultType     string `env:"VM_DEFAULT_TYPE" env-default:"e2-medium"`
	HTTPClient      *http.Client
}

// InstancesAPI is the subset of the GCE Instances service used by Manager.
type InstancesAPI interface {
	Insert(ctx context.Context, project, zone string, inst *compute.Instance) (*compute.Operation, error)
	Get(ctx context.Context, project, zone, name string) (*compute.Instance, error)
	List(ctx context.Context, project, zone string) ([]*compute.Instance, error)
	Start(ctx context.Context, project, zone, name string) (*compute.Operation, error)
	Stop(ctx context.Context, project, zone, name string) (*compute.Operation, error)
	Reset(ctx context.Context, project, zone, name string) (*compute.Operation, error)
	Delete(ctx context.Context, project, zone, name string) (*compute.Operation, error)
	SetLabels(ctx context.Context, project, zone, name string, req *compute.InstancesSetLabelsRequest) (*compute.Operation, error)
	GetSerialPortOutput(ctx context.Context, project, zone, name string) (string, error)
}

// sdkInstances wraps *compute.Service.
type sdkInstances struct {
	svc *compute.Service
}

func (s *sdkInstances) Insert(ctx context.Context, project, zone string, inst *compute.Instance) (*compute.Operation, error) {
	return s.svc.Instances.Insert(project, zone, inst).Context(ctx).Do()
}
func (s *sdkInstances) Get(ctx context.Context, project, zone, name string) (*compute.Instance, error) {
	return s.svc.Instances.Get(project, zone, name).Context(ctx).Do()
}
func (s *sdkInstances) List(ctx context.Context, project, zone string) ([]*compute.Instance, error) {
	var out []*compute.Instance
	err := s.svc.Instances.List(project, zone).Pages(ctx, func(page *compute.InstanceList) error {
		out = append(out, page.Items...)
		return nil
	})
	return out, err
}
func (s *sdkInstances) Start(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return s.svc.Instances.Start(project, zone, name).Context(ctx).Do()
}
func (s *sdkInstances) Stop(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return s.svc.Instances.Stop(project, zone, name).Context(ctx).Do()
}
func (s *sdkInstances) Reset(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return s.svc.Instances.Reset(project, zone, name).Context(ctx).Do()
}
func (s *sdkInstances) Delete(ctx context.Context, project, zone, name string) (*compute.Operation, error) {
	return s.svc.Instances.Delete(project, zone, name).Context(ctx).Do()
}
func (s *sdkInstances) SetLabels(ctx context.Context, project, zone, name string, req *compute.InstancesSetLabelsRequest) (*compute.Operation, error) {
	return s.svc.Instances.SetLabels(project, zone, name, req).Context(ctx).Do()
}
func (s *sdkInstances) GetSerialPortOutput(ctx context.Context, project, zone, name string) (string, error) {
	out, err := s.svc.Instances.GetSerialPortOutput(project, zone, name).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return out.Contents, nil
}

// Manager implements vm.VMManager for GCE.
type Manager struct {
	api     InstancesAPI
	project string
	zone    string
	defType string
}

// New creates a GCE manager using Application Default Credentials.
func New(cfg Config) (*Manager, error) {
	if cfg.ProjectID == "" {
		return nil, pkgerrors.InvalidArgument("project_id is required", nil)
	}
	zone := cfg.Zone
	if zone == "" {
		zone = "us-central1-a"
	}

	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}
	if cfg.HTTPClient != nil {
		opts = append(opts, option.WithHTTPClient(cfg.HTTPClient))
	}

	svc, err := compute.NewService(context.Background(), opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create GCE client", err)
	}
	return NewWithAPI(&sdkInstances{svc: svc}, cfg.ProjectID, zone, cfg.DefaultType), nil
}

// NewWithAPI creates a Manager with an injected InstancesAPI.
func NewWithAPI(api InstancesAPI, project, zone, defaultType string) *Manager {
	if defaultType == "" {
		defaultType = "e2-medium"
	}
	return &Manager{api: api, project: project, zone: zone, defType: defaultType}
}

func (m *Manager) Create(ctx context.Context, opts vm.CreateOptions) (*vm.Instance, error) {
	if opts.ImageID == "" {
		return nil, pkgerrors.InvalidArgument("image_id is required", nil)
	}
	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("vm-%d", time.Now().UnixNano()%1_000_000)
	}
	machineType := opts.InstanceType
	if machineType == "" {
		machineType = m.defType
	}
	zone := opts.Zone
	if zone == "" {
		zone = m.zone
	}

	inst := &compute.Instance{
		Name:        name,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType),
		Disks: []*compute.AttachedDisk{{
			Boot:       true,
			AutoDelete: true,
			InitializeParams: &compute.AttachedDiskInitializeParams{
				SourceImage: opts.ImageID,
			},
		}},
		NetworkInterfaces: []*compute.NetworkInterface{{
			AccessConfigs: []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT", Name: "External NAT"}},
		}},
		Labels: sanitizeLabels(opts.Tags),
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{},
		},
	}
	if opts.UserData != "" {
		inst.Metadata.Items = append(inst.Metadata.Items, &compute.MetadataItems{
			Key:   "startup-script",
			Value: &opts.UserData,
		})
	}

	if _, err := m.api.Insert(ctx, m.project, zone, inst); err != nil {
		return nil, pkgerrors.Internal("failed to create GCE instance", err)
	}
	created, err := m.api.Get(ctx, m.project, zone, name)
	if err != nil {
		// Insert accepted; return optimistic pending instance.
		return &vm.Instance{
			ID: name, Name: name, State: vm.InstanceStatePending,
			InstanceType: machineType, ImageID: opts.ImageID, Zone: zone, Tags: opts.Tags,
			LaunchTime: time.Now(),
		}, nil
	}
	return mapGCEInstance(created, zone), nil
}

func (m *Manager) Get(ctx context.Context, instanceID string) (*vm.Instance, error) {
	inst, err := m.api.Get(ctx, m.project, m.zone, instanceID)
	if err != nil {
		if isNotFound(err) {
			return nil, vm.ErrInstanceNotFound
		}
		return nil, pkgerrors.Internal("failed to get GCE instance", err)
	}
	return mapGCEInstance(inst, m.zone), nil
}

func (m *Manager) List(ctx context.Context, opts vm.ListOptions) (*vm.ListResult, error) {
	items, err := m.api.List(ctx, m.project, m.zone)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list GCE instances", err)
	}
	result := &vm.ListResult{Instances: make([]*vm.Instance, 0, len(items))}
	for _, item := range items {
		mapped := mapGCEInstance(item, m.zone)
		if opts.State != "" && mapped.State != opts.State {
			continue
		}
		if len(opts.Tags) > 0 && !tagsMatch(mapped.Tags, opts.Tags) {
			continue
		}
		result.Instances = append(result.Instances, mapped)
	}
	if opts.Limit > 0 && len(result.Instances) > opts.Limit {
		result.Instances = result.Instances[:opts.Limit]
	}
	return result, nil
}

func (m *Manager) Start(ctx context.Context, instanceID string) error {
	_, err := m.api.Start(ctx, m.project, m.zone, instanceID)
	if err != nil {
		if isNotFound(err) {
			return vm.ErrInstanceNotFound
		}
		return pkgerrors.Internal("failed to start GCE instance", err)
	}
	return nil
}

func (m *Manager) Stop(ctx context.Context, instanceID string) error {
	_, err := m.api.Stop(ctx, m.project, m.zone, instanceID)
	if err != nil {
		if isNotFound(err) {
			return vm.ErrInstanceNotFound
		}
		return pkgerrors.Internal("failed to stop GCE instance", err)
	}
	return nil
}

func (m *Manager) Reboot(ctx context.Context, instanceID string) error {
	_, err := m.api.Reset(ctx, m.project, m.zone, instanceID)
	if err != nil {
		if isNotFound(err) {
			return vm.ErrInstanceNotFound
		}
		return pkgerrors.Internal("failed to reboot GCE instance", err)
	}
	return nil
}

func (m *Manager) Terminate(ctx context.Context, instanceID string) error {
	_, err := m.api.Delete(ctx, m.project, m.zone, instanceID)
	if err != nil {
		if isNotFound(err) {
			return vm.ErrInstanceNotFound
		}
		return pkgerrors.Internal("failed to terminate GCE instance", err)
	}
	return nil
}

func (m *Manager) UpdateTags(ctx context.Context, instanceID string, tags map[string]string) error {
	inst, err := m.api.Get(ctx, m.project, m.zone, instanceID)
	if err != nil {
		if isNotFound(err) {
			return vm.ErrInstanceNotFound
		}
		return pkgerrors.Internal("failed to get GCE instance for labels", err)
	}
	labels := sanitizeLabels(tags)
	if inst.Labels != nil {
		for k, v := range inst.Labels {
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}
	}
	_, err = m.api.SetLabels(ctx, m.project, m.zone, instanceID, &compute.InstancesSetLabelsRequest{
		Labels:           labels,
		LabelFingerprint: inst.LabelFingerprint,
	})
	if err != nil {
		return pkgerrors.Internal("failed to update GCE labels", err)
	}
	return nil
}

func (m *Manager) GetConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	out, err := m.api.GetSerialPortOutput(ctx, m.project, m.zone, instanceID)
	if err != nil {
		if isNotFound(err) {
			return "", vm.ErrInstanceNotFound
		}
		return "", pkgerrors.Internal("failed to get GCE serial output", err)
	}
	return out, nil
}

func mapGCEInstance(inst *compute.Instance, zone string) *vm.Instance {
	state := vm.InstanceStatePending
	switch strings.ToUpper(inst.Status) {
	case "PROVISIONING", "STAGING":
		state = vm.InstanceStatePending
	case "RUNNING":
		state = vm.InstanceStateRunning
	case "STOPPING":
		state = vm.InstanceStateStopping
	case "TERMINATED", "STOPPED":
		state = vm.InstanceStateStopped
	case "SUSPENDING", "SUSPENDED":
		state = vm.InstanceStateStopped
	}

	machineType := inst.MachineType
	if i := strings.LastIndex(machineType, "/"); i >= 0 {
		machineType = machineType[i+1:]
	}

	publicIP, privateIP := "", ""
	for _, ni := range inst.NetworkInterfaces {
		privateIP = ni.NetworkIP
		for _, ac := range ni.AccessConfigs {
			if ac.NatIP != "" {
				publicIP = ac.NatIP
			}
		}
	}

	image := ""
	if len(inst.Disks) > 0 && inst.Disks[0].Source != "" {
		image = inst.Disks[0].Source
	}

	launch := time.Time{}
	if inst.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, inst.CreationTimestamp); err == nil {
			launch = t
		}
	}

	return &vm.Instance{
		ID:           inst.Name,
		Name:         inst.Name,
		State:        state,
		InstanceType: machineType,
		ImageID:      image,
		PublicIP:     publicIP,
		PrivateIP:    privateIP,
		Zone:         zone,
		Tags:         inst.Labels,
		LaunchTime:   launch,
	}
}

func sanitizeLabels(tags map[string]string) map[string]string {
	if tags == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		nk := strings.ToLower(strings.ReplaceAll(k, "_", "-"))
		out[nk] = v
	}
	return out
}

func tagsMatch(have, want map[string]string) bool {
	for k, v := range want {
		nk := strings.ToLower(strings.ReplaceAll(k, "_", "-"))
		if have[nk] != v && have[k] != v {
			return false
		}
	}
	return true
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "404") || strings.Contains(msg, "notFound") || strings.Contains(msg, "not found")
}

var _ vm.VMManager = (*Manager)(nil)

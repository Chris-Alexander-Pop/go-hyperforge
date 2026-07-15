package ebs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/block"
)

const defaultPollInterval = 2 * time.Second

// EC2API is the AWS EC2 surface used by the real EBS SDK store.
type EC2API interface {
	CreateVolume(ctx context.Context, params *ec2.CreateVolumeInput, optFns ...func(*ec2.Options)) (*ec2.CreateVolumeOutput, error)
	DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
	DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error)
	ModifyVolume(ctx context.Context, params *ec2.ModifyVolumeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyVolumeOutput, error)
	AttachVolume(ctx context.Context, params *ec2.AttachVolumeInput, optFns ...func(*ec2.Options)) (*ec2.AttachVolumeOutput, error)
	DetachVolume(ctx context.Context, params *ec2.DetachVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DetachVolumeOutput, error)
	CreateSnapshot(ctx context.Context, params *ec2.CreateSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.CreateSnapshotOutput, error)
	DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error)
	DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// SDKConfig configures the real AWS EBS VolumeStore.
type SDKConfig struct {
	Region             string
	AvailabilityZone   string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	Endpoint           string
	// PollInterval for WaitUntil* helpers (default 2s; use a short value in tests).
	PollInterval time.Duration
}

// SDKStore implements block.VolumeStore via AWS EC2 volume APIs.
type SDKStore struct {
	client       EC2API
	cfg          SDKConfig
	pollInterval time.Duration
}

var _ block.VolumeStore = (*SDKStore)(nil)

// NewSDKFromAPI wraps an EC2API (SDK client or test double).
func NewSDKFromAPI(api EC2API, cfg SDKConfig) (*SDKStore, error) {
	if api == nil {
		return nil, errors.InvalidArgument("ec2 api is required", nil)
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.AvailabilityZone == "" {
		cfg.AvailabilityZone = cfg.Region + "a"
	}
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	return &SDKStore{client: api, cfg: cfg, pollInterval: interval}, nil
}

// NewSDK builds an SDKStore from AWS SDK config.
func NewSDK(ctx context.Context, cfg SDKConfig) (*SDKStore, error) {
	if cfg.Region == "" {
		return nil, errors.InvalidArgument("aws region is required", nil)
	}
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, "",
		)))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Unavailable("failed to load aws config", err)
	}
	client := ec2.NewFromConfig(awsCfg, func(o *ec2.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return NewSDKFromAPI(client, cfg)
}

// NewSDKWithBlockConfig maps block.Config into NewSDK.
func NewSDKWithBlockConfig(ctx context.Context, bc block.Config) (*SDKStore, error) {
	return NewSDK(ctx, SDKConfig{
		Region:             bc.Region,
		AvailabilityZone:   bc.AvailabilityZone,
		AWSAccessKeyID:     bc.AWSAccessKeyID,
		AWSSecretAccessKey: bc.AWSSecretAccessKey,
	})
}

func mapVolumeType(t block.VolumeType) types.VolumeType {
	switch t {
	case block.VolumeTypeStandard:
		return types.VolumeTypeStandard
	case block.VolumeTypeIOPS:
		return types.VolumeTypeIo2
	case block.VolumeTypeSSD:
		return types.VolumeTypeGp3
	default:
		return types.VolumeTypeGp3
	}
}

func fromVolumeType(t types.VolumeType) block.VolumeType {
	switch t {
	case types.VolumeTypeStandard, types.VolumeTypeGp2:
		return block.VolumeTypeStandard
	case types.VolumeTypeIo1, types.VolumeTypeIo2:
		return block.VolumeTypeIOPS
	default:
		return block.VolumeTypeSSD
	}
}

func mapVolumeState(s types.VolumeState) block.VolumeState {
	switch s {
	case types.VolumeStateCreating:
		return block.VolumeStateCreating
	case types.VolumeStateAvailable:
		return block.VolumeStateAvailable
	case types.VolumeStateInUse:
		return block.VolumeStateInUse
	case types.VolumeStateDeleting, types.VolumeStateDeleted:
		return block.VolumeStateDeleting
	case types.VolumeStateError:
		return block.VolumeStateError
	default:
		return block.VolumeStateAvailable
	}
}

func volumeFromAWS(v types.Volume) *block.Volume {
	vol := &block.Volume{
		ID:               aws.ToString(v.VolumeId),
		SizeGB:           int64(aws.ToInt32(v.Size)),
		State:            mapVolumeState(v.State),
		VolumeType:       fromVolumeType(v.VolumeType),
		AvailabilityZone: aws.ToString(v.AvailabilityZone),
		Encrypted:        aws.ToBool(v.Encrypted),
		IOPS:             int64(aws.ToInt32(v.Iops)),
		Throughput:       int64(aws.ToInt32(v.Throughput)),
		Attachments:      make([]block.Attachment, 0, len(v.Attachments)),
		Tags:             map[string]string{},
	}
	if v.CreateTime != nil {
		vol.CreatedAt = *v.CreateTime
	}
	for _, a := range v.Attachments {
		vol.Attachments = append(vol.Attachments, block.Attachment{
			InstanceID: aws.ToString(a.InstanceId),
			Device:     aws.ToString(a.Device),
			AttachedAt: aws.ToTime(a.AttachTime),
		})
	}
	for _, tag := range v.Tags {
		k, val := aws.ToString(tag.Key), aws.ToString(tag.Value)
		vol.Tags[k] = val
		if k == "Name" {
			vol.Name = val
		}
	}
	return vol
}

func (s *SDKStore) CreateVolume(ctx context.Context, opts block.CreateVolumeOptions) (*block.Volume, error) {
	if opts.SizeGB <= 0 && opts.SnapshotID == "" {
		return nil, block.ErrInvalidSize
	}
	az := opts.AvailabilityZone
	if az == "" {
		az = s.cfg.AvailabilityZone
	}
	input := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(az),
		Encrypted:        aws.Bool(opts.Encrypted),
		VolumeType:       mapVolumeType(opts.VolumeType),
	}
	if opts.SizeGB > 0 {
		input.Size = aws.Int32(int32(opts.SizeGB))
	}
	if opts.SnapshotID != "" {
		input.SnapshotId = aws.String(opts.SnapshotID)
	}
	if opts.IOPS > 0 {
		input.Iops = aws.Int32(int32(opts.IOPS))
	}
	if opts.Throughput > 0 {
		input.Throughput = aws.Int32(int32(opts.Throughput))
	}
	if opts.EncryptionKeyID != "" {
		input.KmsKeyId = aws.String(opts.EncryptionKeyID)
	}
	out, err := s.client.CreateVolume(ctx, input)
	if err != nil {
		return nil, errors.Unavailable("ec2 CreateVolume failed", err)
	}
	vol := volumeFromAWS(types.Volume{
		VolumeId:         out.VolumeId,
		Size:             out.Size,
		State:            out.State,
		VolumeType:       out.VolumeType,
		AvailabilityZone: out.AvailabilityZone,
		Encrypted:        out.Encrypted,
		Iops:             out.Iops,
		Throughput:       out.Throughput,
		CreateTime:       out.CreateTime,
		Attachments:      out.Attachments,
		Tags:             out.Tags,
	})
	if opts.Name != "" || len(opts.Tags) > 0 {
		tags := make([]types.Tag, 0, len(opts.Tags)+1)
		if opts.Name != "" {
			tags = append(tags, types.Tag{Key: aws.String("Name"), Value: aws.String(opts.Name)})
			vol.Name = opts.Name
		}
		for k, v := range opts.Tags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
			if vol.Tags == nil {
				vol.Tags = map[string]string{}
			}
			vol.Tags[k] = v
		}
		_, _ = s.client.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{vol.ID},
			Tags:      tags,
		})
	}
	return vol, nil
}

func (s *SDKStore) GetVolume(ctx context.Context, volumeID string) (*block.Volume, error) {
	out, err := s.client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, block.ErrVolumeNotFound
		}
		return nil, errors.Unavailable("ec2 DescribeVolumes failed", err)
	}
	if len(out.Volumes) == 0 {
		return nil, block.ErrVolumeNotFound
	}
	return volumeFromAWS(out.Volumes[0]), nil
}

func (s *SDKStore) ListVolumes(ctx context.Context, opts block.ListOptions) (*block.ListResult, error) {
	input := &ec2.DescribeVolumesInput{}
	if opts.NextToken != "" {
		input.NextToken = aws.String(opts.NextToken)
	}
	if opts.Limit > 0 {
		input.MaxResults = aws.Int32(int32(opts.Limit))
	}
	out, err := s.client.DescribeVolumes(ctx, input)
	if err != nil {
		return nil, errors.Unavailable("ec2 DescribeVolumes failed", err)
	}
	result := &block.ListResult{Volumes: make([]*block.Volume, 0, len(out.Volumes))}
	for _, v := range out.Volumes {
		vol := volumeFromAWS(v)
		if len(opts.Filters) > 0 {
			match := true
			for k, val := range opts.Filters {
				switch k {
				case "name":
					if vol.Name != val {
						match = false
					}
				case "state":
					if string(vol.State) != val {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}
		result.Volumes = append(result.Volumes, vol)
	}
	result.NextToken = aws.ToString(out.NextToken)
	return result, nil
}

func (s *SDKStore) DeleteVolume(ctx context.Context, volumeID string) error {
	_, err := s.client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{VolumeId: aws.String(volumeID)})
	if err != nil {
		return errors.Unavailable("ec2 DeleteVolume failed", err)
	}
	return nil
}

func (s *SDKStore) ResizeVolume(ctx context.Context, volumeID string, opts block.ResizeVolumeOptions) (*block.Volume, error) {
	input := &ec2.ModifyVolumeInput{VolumeId: aws.String(volumeID)}
	if opts.NewSizeGB > 0 {
		input.Size = aws.Int32(int32(opts.NewSizeGB))
	}
	if opts.NewVolumeType != "" {
		input.VolumeType = mapVolumeType(opts.NewVolumeType)
	}
	if opts.NewIOPS > 0 {
		input.Iops = aws.Int32(int32(opts.NewIOPS))
	}
	if opts.NewThroughput > 0 {
		input.Throughput = aws.Int32(int32(opts.NewThroughput))
	}
	if _, err := s.client.ModifyVolume(ctx, input); err != nil {
		return nil, errors.Unavailable("ec2 ModifyVolume failed", err)
	}
	return s.GetVolume(ctx, volumeID)
}

func (s *SDKStore) AttachVolume(ctx context.Context, opts block.AttachVolumeOptions) error {
	if opts.VolumeID == "" || opts.InstanceID == "" {
		return errors.InvalidArgument("volumeID and instanceID are required", nil)
	}
	device := opts.Device
	if device == "" {
		device = "/dev/sdf"
	}
	_, err := s.client.AttachVolume(ctx, &ec2.AttachVolumeInput{
		VolumeId:   aws.String(opts.VolumeID),
		InstanceId: aws.String(opts.InstanceID),
		Device:     aws.String(device),
	})
	if err != nil {
		return errors.Unavailable("ec2 AttachVolume failed", err)
	}
	return nil
}

func (s *SDKStore) DetachVolume(ctx context.Context, volumeID, instanceID string) error {
	input := &ec2.DetachVolumeInput{VolumeId: aws.String(volumeID)}
	if instanceID != "" {
		input.InstanceId = aws.String(instanceID)
	}
	_, err := s.client.DetachVolume(ctx, input)
	if err != nil {
		return errors.Unavailable("ec2 DetachVolume failed", err)
	}
	return nil
}

func (s *SDKStore) CreateSnapshot(ctx context.Context, opts block.CreateSnapshotOptions) (*block.Snapshot, error) {
	out, err := s.client.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{
		VolumeId:    aws.String(opts.VolumeID),
		Description: aws.String(opts.Description),
	})
	if err != nil {
		return nil, errors.Unavailable("ec2 CreateSnapshot failed", err)
	}
	snap := &block.Snapshot{
		ID:          aws.ToString(out.SnapshotId),
		VolumeID:    aws.ToString(out.VolumeId),
		SizeGB:      int64(aws.ToInt32(out.VolumeSize)),
		State:       string(out.State),
		Description: aws.ToString(out.Description),
		Tags:        opts.Tags,
	}
	if out.StartTime != nil {
		snap.CreatedAt = *out.StartTime
	} else {
		snap.CreatedAt = time.Now().UTC()
	}
	if snap.Tags == nil {
		snap.Tags = map[string]string{}
	}
	return snap, nil
}

func (s *SDKStore) GetSnapshot(ctx context.Context, snapshotID string) (*block.Snapshot, error) {
	out, err := s.client.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{
		SnapshotIds: []string{snapshotID},
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, block.ErrSnapshotNotFound
		}
		return nil, errors.Unavailable("ec2 DescribeSnapshots failed", err)
	}
	if len(out.Snapshots) == 0 {
		return nil, block.ErrSnapshotNotFound
	}
	sn := out.Snapshots[0]
	snap := &block.Snapshot{
		ID:          aws.ToString(sn.SnapshotId),
		VolumeID:    aws.ToString(sn.VolumeId),
		SizeGB:      int64(aws.ToInt32(sn.VolumeSize)),
		State:       string(sn.State),
		Description: aws.ToString(sn.Description),
		Tags:        map[string]string{},
	}
	if sn.StartTime != nil {
		snap.CreatedAt = *sn.StartTime
	}
	for _, tag := range sn.Tags {
		snap.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return snap, nil
}

func (s *SDKStore) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	_, err := s.client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{SnapshotId: aws.String(snapshotID)})
	if err != nil {
		return errors.Unavailable("ec2 DeleteSnapshot failed", err)
	}
	return nil
}

// WaitUntilVolumeAvailable polls DescribeVolumes until the volume is available or ctx is done.
func (s *SDKStore) WaitUntilVolumeAvailable(ctx context.Context, volumeID string) error {
	return s.waitVolume(ctx, volumeID, func(v *types.Volume, missing bool) (bool, error) {
		if missing {
			return false, block.ErrVolumeNotFound
		}
		return v.State == types.VolumeStateAvailable, nil
	})
}

// WaitUntilVolumeInUse polls DescribeVolumes until the volume is in-use or ctx is done.
func (s *SDKStore) WaitUntilVolumeInUse(ctx context.Context, volumeID string) error {
	return s.waitVolume(ctx, volumeID, func(v *types.Volume, missing bool) (bool, error) {
		if missing {
			return false, block.ErrVolumeNotFound
		}
		return v.State == types.VolumeStateInUse, nil
	})
}

// WaitUntilVolumeDeleted polls until the volume is gone. NotFound / empty Describe is success.
func (s *SDKStore) WaitUntilVolumeDeleted(ctx context.Context, volumeID string) error {
	return s.waitVolume(ctx, volumeID, func(v *types.Volume, missing bool) (bool, error) {
		if missing {
			return true, nil
		}
		if v.State == types.VolumeStateDeleted {
			return true, nil
		}
		return false, nil
	})
}

// WaitUntilSnapshotCompleted polls DescribeSnapshots until the snapshot is completed or ctx is done.
func (s *SDKStore) WaitUntilSnapshotCompleted(ctx context.Context, snapshotID string) error {
	interval := s.pollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, err := s.client.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{
			SnapshotIds: []string{snapshotID},
		})
		if err != nil {
			if isNotFoundErr(err) {
				return block.ErrSnapshotNotFound
			}
			return errors.Unavailable("ec2 DescribeSnapshots failed", err)
		}
		if len(out.Snapshots) == 0 {
			return block.ErrSnapshotNotFound
		}
		if out.Snapshots[0].State == types.SnapshotStateCompleted {
			return nil
		}
		if out.Snapshots[0].State == types.SnapshotStateError {
			return errors.Internal("snapshot entered error state", nil)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *SDKStore) waitVolume(ctx context.Context, volumeID string, done func(v *types.Volume, missing bool) (bool, error)) error {
	interval := s.pollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, err := s.client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			VolumeIds: []string{volumeID},
		})
		missing := false
		var vol *types.Volume
		if err != nil {
			if isNotFoundErr(err) {
				missing = true
			} else {
				return errors.Unavailable("ec2 DescribeVolumes failed", err)
			}
		} else if len(out.Volumes) == 0 {
			missing = true
		} else {
			vol = &out.Volumes[0]
		}
		ok, err := done(vol, missing)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, block.ErrVolumeNotFound) || errors.Is(err, block.ErrSnapshotNotFound) {
		return true
	}
	return errors.IsCode(err, errors.CodeNotFound)
}

// MemoryEC2API is an in-process EC2API for unit tests.
type MemoryEC2API struct {
	mu    sync.Mutex
	vols  map[string]types.Volume
	snaps map[string]types.Snapshot
	seq   int
}

// NewMemoryEC2API creates a test double.
func NewMemoryEC2API() *MemoryEC2API {
	return &MemoryEC2API{
		vols:  make(map[string]types.Volume),
		snaps: make(map[string]types.Snapshot),
	}
}

func (m *MemoryEC2API) nextID(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s-%04d", prefix, m.seq)
}

// SetVolumeState updates a volume's state (for waiter tests).
func (m *MemoryEC2API) SetVolumeState(volumeID string, state types.VolumeState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.vols[volumeID]
	if !ok {
		return errors.NotFound("volume not found", nil)
	}
	v.State = state
	m.vols[volumeID] = v
	return nil
}

// SetSnapshotState updates a snapshot's state (for waiter tests).
func (m *MemoryEC2API) SetSnapshotState(snapshotID string, state types.SnapshotState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sn, ok := m.snaps[snapshotID]
	if !ok {
		return errors.NotFound("snapshot not found", nil)
	}
	sn.State = state
	m.snaps[snapshotID] = sn
	return nil
}

func (m *MemoryEC2API) CreateVolume(_ context.Context, params *ec2.CreateVolumeInput, _ ...func(*ec2.Options)) (*ec2.CreateVolumeOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextID("vol")
	now := time.Now().UTC()
	size := params.Size
	if size == nil && params.SnapshotId != nil {
		if sn, ok := m.snaps[aws.ToString(params.SnapshotId)]; ok {
			size = sn.VolumeSize
		}
	}
	v := types.Volume{
		VolumeId:         aws.String(id),
		Size:             size,
		State:            types.VolumeStateAvailable,
		VolumeType:       params.VolumeType,
		AvailabilityZone: params.AvailabilityZone,
		Encrypted:        params.Encrypted,
		Iops:             params.Iops,
		Throughput:       params.Throughput,
		CreateTime:       &now,
		Attachments:      []types.VolumeAttachment{},
	}
	m.vols[id] = v
	return &ec2.CreateVolumeOutput{
		VolumeId:         v.VolumeId,
		Size:             v.Size,
		State:            v.State,
		VolumeType:       v.VolumeType,
		AvailabilityZone: v.AvailabilityZone,
		Encrypted:        v.Encrypted,
		Iops:             v.Iops,
		Throughput:       v.Throughput,
		CreateTime:       v.CreateTime,
		Attachments:      v.Attachments,
	}, nil
}

func (m *MemoryEC2API) DescribeVolumes(_ context.Context, params *ec2.DescribeVolumesInput, _ ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var vols []types.Volume
	if len(params.VolumeIds) > 0 {
		for _, id := range params.VolumeIds {
			v, ok := m.vols[id]
			if !ok {
				return nil, errors.NotFound("volume not found", nil)
			}
			vols = append(vols, v)
		}
	} else {
		for _, v := range m.vols {
			vols = append(vols, v)
		}
	}
	return &ec2.DescribeVolumesOutput{Volumes: vols}, nil
}

func (m *MemoryEC2API) DeleteVolume(_ context.Context, params *ec2.DeleteVolumeInput, _ ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.vols, aws.ToString(params.VolumeId))
	return &ec2.DeleteVolumeOutput{}, nil
}

func (m *MemoryEC2API) ModifyVolume(_ context.Context, params *ec2.ModifyVolumeInput, _ ...func(*ec2.Options)) (*ec2.ModifyVolumeOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := aws.ToString(params.VolumeId)
	v, ok := m.vols[id]
	if !ok {
		return nil, errors.NotFound("volume not found", nil)
	}
	if params.Size != nil {
		v.Size = params.Size
	}
	if params.VolumeType != "" {
		v.VolumeType = params.VolumeType
	}
	if params.Iops != nil {
		v.Iops = params.Iops
	}
	if params.Throughput != nil {
		v.Throughput = params.Throughput
	}
	m.vols[id] = v
	return &ec2.ModifyVolumeOutput{}, nil
}

func (m *MemoryEC2API) AttachVolume(_ context.Context, params *ec2.AttachVolumeInput, _ ...func(*ec2.Options)) (*ec2.AttachVolumeOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := aws.ToString(params.VolumeId)
	v, ok := m.vols[id]
	if !ok {
		return nil, errors.NotFound("volume not found", nil)
	}
	now := time.Now().UTC()
	v.Attachments = append(v.Attachments, types.VolumeAttachment{
		InstanceId: params.InstanceId,
		Device:     params.Device,
		AttachTime: &now,
	})
	v.State = types.VolumeStateInUse
	m.vols[id] = v
	return &ec2.AttachVolumeOutput{
		VolumeId:   params.VolumeId,
		InstanceId: params.InstanceId,
		Device:     params.Device,
	}, nil
}

func (m *MemoryEC2API) DetachVolume(_ context.Context, params *ec2.DetachVolumeInput, _ ...func(*ec2.Options)) (*ec2.DetachVolumeOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := aws.ToString(params.VolumeId)
	v, ok := m.vols[id]
	if !ok {
		return nil, errors.NotFound("volume not found", nil)
	}
	inst := aws.ToString(params.InstanceId)
	kept := v.Attachments[:0]
	for _, a := range v.Attachments {
		if inst != "" && aws.ToString(a.InstanceId) == inst {
			continue
		}
		if inst == "" {
			continue
		}
		kept = append(kept, a)
	}
	v.Attachments = kept
	if len(v.Attachments) == 0 {
		v.State = types.VolumeStateAvailable
	}
	m.vols[id] = v
	return &ec2.DetachVolumeOutput{}, nil
}

func (m *MemoryEC2API) CreateSnapshot(_ context.Context, params *ec2.CreateSnapshotInput, _ ...func(*ec2.Options)) (*ec2.CreateSnapshotOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	volID := aws.ToString(params.VolumeId)
	v, ok := m.vols[volID]
	if !ok {
		return nil, errors.NotFound("volume not found", nil)
	}
	id := m.nextID("snap")
	now := time.Now().UTC()
	sn := types.Snapshot{
		SnapshotId:  aws.String(id),
		VolumeId:    aws.String(volID),
		VolumeSize:  v.Size,
		State:       types.SnapshotStateCompleted,
		Description: params.Description,
		StartTime:   &now,
	}
	m.snaps[id] = sn
	return &ec2.CreateSnapshotOutput{
		SnapshotId:  sn.SnapshotId,
		VolumeId:    sn.VolumeId,
		VolumeSize:  sn.VolumeSize,
		State:       sn.State,
		Description: sn.Description,
		StartTime:   sn.StartTime,
	}, nil
}

func (m *MemoryEC2API) DescribeSnapshots(_ context.Context, params *ec2.DescribeSnapshotsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var snaps []types.Snapshot
	if len(params.SnapshotIds) > 0 {
		for _, id := range params.SnapshotIds {
			sn, ok := m.snaps[id]
			if !ok {
				return nil, errors.NotFound("snapshot not found", nil)
			}
			snaps = append(snaps, sn)
		}
	} else {
		for _, sn := range m.snaps {
			snaps = append(snaps, sn)
		}
	}
	return &ec2.DescribeSnapshotsOutput{Snapshots: snaps}, nil
}

func (m *MemoryEC2API) DeleteSnapshot(_ context.Context, params *ec2.DeleteSnapshotInput, _ ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.snaps, aws.ToString(params.SnapshotId))
	return &ec2.DeleteSnapshotOutput{}, nil
}

func (m *MemoryEC2API) CreateTags(_ context.Context, params *ec2.CreateTagsInput, _ ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range params.Resources {
		if v, ok := m.vols[id]; ok {
			v.Tags = append(v.Tags, params.Tags...)
			m.vols[id] = v
		}
	}
	return &ec2.CreateTagsOutput{}, nil
}

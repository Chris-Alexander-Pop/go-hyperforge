// Package registry defines an IoT device registry interface for identity and metadata.
package registry

import (
	"context"
	"time"
)

// Status is the lifecycle status of a registered device.
type Status string

const (
	StatusActive      Status = "active"
	StatusInactive    Status = "inactive"
	StatusProvisioned Status = "provisioned"
	StatusRevoked     Status = "revoked"
)

// Device is a registered IoT device record.
type Device struct {
	ID            string
	Name          string
	ThingType     string
	Status        Status
	Attributes    map[string]string
	CertificateID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSeenAt    time.Time
}

// RegisterOptions configures device registration.
type RegisterOptions struct {
	ID            string
	Name          string
	ThingType     string
	Attributes    map[string]string
	CertificateID string
	Status        Status
}

// UpdateOptions configures a partial device update.
type UpdateOptions struct {
	Name          *string
	ThingType     *string
	Status        *Status
	Attributes    map[string]string
	CertificateID *string
}

// ListOptions filters device listing.
type ListOptions struct {
	ThingType string
	Status    Status
	Limit     int
}

// DeviceRegistry stores and looks up IoT devices.
type DeviceRegistry interface {
	Register(ctx context.Context, opts RegisterOptions) (*Device, error)
	Get(ctx context.Context, deviceID string) (*Device, error)
	Update(ctx context.Context, deviceID string, opts UpdateOptions) (*Device, error)
	Deregister(ctx context.Context, deviceID string) error
	List(ctx context.Context, opts ListOptions) ([]*Device, error)
	Touch(ctx context.Context, deviceID string) error
	Close() error
}

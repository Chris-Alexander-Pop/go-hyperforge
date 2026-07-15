package iot

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedClient wraps a Client with logging and OpenTelemetry spans.
type InstrumentedClient struct {
	next   Client
	tracer trace.Tracer
}

// NewInstrumentedClient decorates client with logging and tracing.
func NewInstrumentedClient(next Client) *InstrumentedClient {
	return &InstrumentedClient{
		next:   next,
		tracer: otel.Tracer("pkg/iot"),
	}
}

// Connect logs and traces broker connection.
func (c *InstrumentedClient) Connect(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "iot.Client.Connect")
	defer span.End()

	logger.L().InfoContext(ctx, "connecting MQTT client")
	err := c.next.Connect(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "MQTT connect failed", "error", err)
		return err
	}
	return nil
}

// Disconnect logs client disconnect.
func (c *InstrumentedClient) Disconnect() {
	logger.L().Info("disconnecting MQTT client")
	c.next.Disconnect()
}

// IsConnected delegates to the underlying client.
func (c *InstrumentedClient) IsConnected() bool {
	return c.next.IsConnected()
}

// Publish logs and traces a publish.
func (c *InstrumentedClient) Publish(ctx context.Context, topic string, payload []byte) error {
	ctx, span := c.tracer.Start(ctx, "iot.Client.Publish", trace.WithAttributes(
		attribute.String("mqtt.topic", topic),
		attribute.Int("mqtt.payload_size", len(payload)),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "publishing MQTT message", "topic", topic, "size", len(payload))
	err := c.next.Publish(ctx, topic, payload)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "MQTT publish failed", "topic", topic, "error", err)
		return err
	}
	return nil
}

// PublishWithOptions logs and traces a publish with options.
func (c *InstrumentedClient) PublishWithOptions(ctx context.Context, topic string, payload []byte, qos QoS, retained bool) error {
	ctx, span := c.tracer.Start(ctx, "iot.Client.PublishWithOptions", trace.WithAttributes(
		attribute.String("mqtt.topic", topic),
		attribute.Int("mqtt.qos", int(qos)),
		attribute.Bool("mqtt.retained", retained),
	))
	defer span.End()

	err := c.next.PublishWithOptions(ctx, topic, payload, qos, retained)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "MQTT publish failed", "topic", topic, "error", err)
		return err
	}
	return nil
}

// Subscribe logs and traces a subscription.
func (c *InstrumentedClient) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	ctx, span := c.tracer.Start(ctx, "iot.Client.Subscribe", trace.WithAttributes(
		attribute.String("mqtt.topic", topic),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "subscribing to MQTT topic", "topic", topic)
	err := c.next.Subscribe(ctx, topic, handler)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "MQTT subscribe failed", "topic", topic, "error", err)
		return err
	}
	return nil
}

// SubscribeWithQoS logs and traces a QoS subscription.
func (c *InstrumentedClient) SubscribeWithQoS(ctx context.Context, topic string, qos QoS, handler MessageHandler) error {
	ctx, span := c.tracer.Start(ctx, "iot.Client.SubscribeWithQoS", trace.WithAttributes(
		attribute.String("mqtt.topic", topic),
		attribute.Int("mqtt.qos", int(qos)),
	))
	defer span.End()

	err := c.next.SubscribeWithQoS(ctx, topic, qos, handler)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "MQTT subscribe failed", "topic", topic, "error", err)
		return err
	}
	return nil
}

// Unsubscribe logs and traces an unsubscribe.
func (c *InstrumentedClient) Unsubscribe(ctx context.Context, topic string) error {
	ctx, span := c.tracer.Start(ctx, "iot.Client.Unsubscribe", trace.WithAttributes(
		attribute.String("mqtt.topic", topic),
	))
	defer span.End()

	err := c.next.Unsubscribe(ctx, topic)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "MQTT unsubscribe failed", "topic", topic, "error", err)
		return err
	}
	return nil
}

// InstrumentedUpdater wraps an Updater with logging and OpenTelemetry spans.
type InstrumentedUpdater struct {
	next   Updater
	tracer trace.Tracer
}

// NewInstrumentedUpdater decorates updater with logging and tracing.
func NewInstrumentedUpdater(next Updater) *InstrumentedUpdater {
	return &InstrumentedUpdater{
		next:   next,
		tracer: otel.Tracer("pkg/iot"),
	}
}

// CheckForUpdate logs and traces an update check.
func (u *InstrumentedUpdater) CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateManifest, bool, error) {
	ctx, span := u.tracer.Start(ctx, "iot.Updater.CheckForUpdate", trace.WithAttributes(
		attribute.String("ota.current_version", currentVersion),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "checking for OTA update", "current_version", currentVersion)
	manifest, available, err := u.next.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "OTA check failed", "error", err)
		return nil, false, err
	}
	if available && manifest != nil {
		span.SetAttributes(attribute.String("ota.available_version", manifest.Version))
		logger.L().InfoContext(ctx, "OTA update available", "version", manifest.Version)
	}
	return manifest, available, nil
}

// DownloadUpdate logs and traces a download.
func (u *InstrumentedUpdater) DownloadUpdate(ctx context.Context, manifest *UpdateManifest) (map[string][]byte, error) {
	version := ""
	files := 0
	if manifest != nil {
		version = manifest.Version
		files = len(manifest.Files)
	}
	ctx, span := u.tracer.Start(ctx, "iot.Updater.DownloadUpdate", trace.WithAttributes(
		attribute.String("ota.version", version),
		attribute.Int("ota.file_count", files),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "downloading OTA update", "version", version, "files", files)
	data, err := u.next.DownloadUpdate(ctx, manifest)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "OTA download failed", "version", version, "error", err)
		return nil, err
	}
	return data, nil
}

// ApplyUpdate logs and traces apply.
func (u *InstrumentedUpdater) ApplyUpdate(ctx context.Context, files map[string][]byte) error {
	ctx, span := u.tracer.Start(ctx, "iot.Updater.ApplyUpdate", trace.WithAttributes(
		attribute.Int("ota.file_count", len(files)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "applying OTA update", "files", len(files))
	err := u.next.ApplyUpdate(ctx, files)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "OTA apply failed", "error", err)
		return err
	}
	return nil
}

// CheckAndApply logs and traces check-and-apply.
func (u *InstrumentedUpdater) CheckAndApply(ctx context.Context, deviceID, currentVersion string) error {
	ctx, span := u.tracer.Start(ctx, "iot.Updater.CheckAndApply", trace.WithAttributes(
		attribute.String("ota.device_id", deviceID),
		attribute.String("ota.current_version", currentVersion),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "OTA check and apply", "device_id", deviceID, "current_version", currentVersion)
	err := u.next.CheckAndApply(ctx, deviceID, currentVersion)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "OTA check and apply failed", "device_id", deviceID, "error", err)
		return err
	}
	return nil
}

// GetState delegates to the underlying updater.
func (u *InstrumentedUpdater) GetState() UpdateState {
	return u.next.GetState()
}

// SetProgressCallback delegates to the underlying updater.
func (u *InstrumentedUpdater) SetProgressCallback(cb ProgressCallback) {
	u.next.SetProgressCallback(cb)
}

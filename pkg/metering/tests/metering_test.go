package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MeteringTestSuite struct {
	test.Suite
	meter metering.Meter
	rater metering.Rater
	raw   *memory.MemoryMetering
}

func (s *MeteringTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.raw = memory.New()
	s.meter = metering.NewInstrumentedMeter(s.raw)
	s.rater = metering.NewInstrumentedRater(s.raw)
}

func (s *MeteringTestSuite) TearDownTest() {
	_ = s.meter.Close()
}

func (s *MeteringTestSuite) validEvent(tenant, resource string, qty float64) metering.UsageEvent {
	return metering.UsageEvent{
		TenantID:     tenant,
		ResourceType: resource,
		ResourceID:   "res-1",
		Quantity:     qty,
		Timestamp:    time.Now().UTC(),
		Metadata:     map[string]string{"region": "us-east-1"},
	}
}

func (s *MeteringTestSuite) TestRecordAndGetUsage() {
	ev := s.validEvent("tenant-a", "compute.instance.small", 2.5)
	s.NoError(s.meter.RecordUsage(s.Ctx, ev))

	got, err := s.meter.GetUsage(s.Ctx, metering.UsageFilter{TenantID: "tenant-a"})
	s.NoError(err)
	s.Len(got, 1)
	s.Equal("tenant-a", got[0].TenantID)
	s.Equal("compute.instance.small", got[0].ResourceType)
	s.Equal(2.5, got[0].Quantity)
	s.NotEmpty(got[0].ID)
	s.Equal("us-east-1", got[0].Metadata["region"])
}

func (s *MeteringTestSuite) TestGetUsageFilters() {
	now := time.Now().UTC()
	s.NoError(s.meter.RecordUsage(s.Ctx, metering.UsageEvent{
		TenantID: "t1", ResourceType: "compute.instance.small", Quantity: 1, Timestamp: now.Add(-2 * time.Hour),
	}))
	s.NoError(s.meter.RecordUsage(s.Ctx, metering.UsageEvent{
		TenantID: "t1", ResourceType: "storage.standard", Quantity: 10, Timestamp: now.Add(-30 * time.Minute),
	}))
	s.NoError(s.meter.RecordUsage(s.Ctx, metering.UsageEvent{
		TenantID: "t2", ResourceType: "storage.standard", Quantity: 5, Timestamp: now,
	}))

	byTenant, err := s.meter.GetUsage(s.Ctx, metering.UsageFilter{TenantID: "t1"})
	s.NoError(err)
	s.Len(byTenant, 2)

	byType, err := s.meter.GetUsage(s.Ctx, metering.UsageFilter{ResourceType: "storage.standard"})
	s.NoError(err)
	s.Len(byType, 2)

	byWindow, err := s.meter.GetUsage(s.Ctx, metering.UsageFilter{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(time.Minute),
	})
	s.NoError(err)
	s.Len(byWindow, 2)
}

func (s *MeteringTestSuite) TestRecordUsageInvalid() {
	cases := []metering.UsageEvent{
		{ResourceType: "compute.instance.small", Quantity: 1},
		{TenantID: "t1", Quantity: 1},
		{TenantID: "t1", ResourceType: "compute.instance.small", Quantity: 0},
		{TenantID: "t1", ResourceType: "compute.instance.small", Quantity: -1},
	}
	for _, ev := range cases {
		err := s.meter.RecordUsage(s.Ctx, ev)
		s.True(errors.Is(err, metering.ErrInvalidUsage), "want ErrInvalidUsage, got %v for %+v", err, ev)
	}
}

func (s *MeteringTestSuite) TestGetRateAndCalculateCost() {
	rate, err := s.rater.GetRate(s.Ctx, "compute.instance.small")
	s.NoError(err)
	s.Equal(0.02, rate.PricePerUnit)
	s.Equal("USD", rate.Currency)

	cost, err := s.rater.CalculateCost(s.Ctx, s.validEvent("t1", "compute.instance.small", 10))
	s.NoError(err)
	s.InDelta(0.2, cost, 1e-9)
}

func (s *MeteringTestSuite) TestGetRateNotFound() {
	_, err := s.rater.GetRate(s.Ctx, "missing.resource")
	s.True(errors.Is(err, metering.ErrRateNotFound))
}

func (s *MeteringTestSuite) TestGetRateEmptyType() {
	_, err := s.rater.GetRate(s.Ctx, "")
	s.True(errors.Is(err, metering.ErrInvalidUsage))
}

func (s *MeteringTestSuite) TestSetRateAndListRates() {
	s.NoError(s.rater.SetRate(s.Ctx, metering.RateCard{
		ResourceType: "gpu.a100",
		PricePerUnit: 3.5,
		Currency:     "USD",
		Unit:         "hour",
	}))

	rate, err := s.rater.GetRate(s.Ctx, "gpu.a100")
	s.NoError(err)
	s.Equal(3.5, rate.PricePerUnit)

	rates, err := s.rater.ListRates(s.Ctx)
	s.NoError(err)
	s.GreaterOrEqual(len(rates), 3)

	s.NoError(s.rater.SetRate(s.Ctx, metering.RateCard{
		ResourceType: "gpu.a100",
		PricePerUnit: 4.0,
		Currency:     "USD",
		Unit:         "hour",
	}))
	rate, err = s.rater.GetRate(s.Ctx, "gpu.a100")
	s.NoError(err)
	s.Equal(4.0, rate.PricePerUnit)
}

func (s *MeteringTestSuite) TestSetRateInvalid() {
	cases := []metering.RateCard{
		{PricePerUnit: 1, Currency: "USD", Unit: "hour"},
		{ResourceType: "x", PricePerUnit: -1, Currency: "USD", Unit: "hour"},
		{ResourceType: "x", PricePerUnit: 1, Unit: "hour"},
		{ResourceType: "x", PricePerUnit: 1, Currency: "USD"},
	}
	for _, rate := range cases {
		err := s.rater.SetRate(s.Ctx, rate)
		s.True(errors.Is(err, metering.ErrInvalidUsage), "want ErrInvalidUsage, got %v for %+v", err, rate)
	}
}

func (s *MeteringTestSuite) TestCalculateCostMissingRate() {
	_, err := s.rater.CalculateCost(s.Ctx, s.validEvent("t1", "unknown.type", 1))
	s.True(errors.Is(err, metering.ErrRateNotFound))
}

func (s *MeteringTestSuite) TestCalculateCostInvalidUsage() {
	_, err := s.rater.CalculateCost(s.Ctx, metering.UsageEvent{
		TenantID: "t1", ResourceType: "compute.instance.small", Quantity: 0,
	})
	s.True(errors.Is(err, metering.ErrInvalidUsage))
}

func (s *MeteringTestSuite) TestCloseRejectsOps() {
	s.NoError(s.meter.Close())

	err := s.meter.RecordUsage(s.Ctx, s.validEvent("t1", "compute.instance.small", 1))
	s.Equal(metering.CodeClosed, errors.Code(err))

	_, err = s.meter.GetUsage(s.Ctx, metering.UsageFilter{})
	s.Equal(metering.CodeClosed, errors.Code(err))

	_, err = s.rater.GetRate(s.Ctx, "compute.instance.small")
	s.Equal(metering.CodeClosed, errors.Code(err))

	err = s.rater.SetRate(s.Ctx, metering.RateCard{
		ResourceType: "x", PricePerUnit: 1, Currency: "USD", Unit: "hour",
	})
	s.Equal(metering.CodeClosed, errors.Code(err))

	_, err = s.rater.ListRates(s.Ctx)
	s.Equal(metering.CodeClosed, errors.Code(err))

	_, err = s.rater.CalculateCost(s.Ctx, s.validEvent("t1", "compute.instance.small", 1))
	s.Equal(metering.CodeClosed, errors.Code(err))
}

func (s *MeteringTestSuite) TestValidateHelpers() {
	s.True(errors.Is(metering.ValidateUsageEvent(metering.UsageEvent{}), metering.ErrInvalidUsage))
	s.NoError(metering.ValidateUsageEvent(s.validEvent("t", "r", 1)))

	s.True(errors.Is(metering.ValidateRateCard(metering.RateCard{}), metering.ErrInvalidUsage))
	s.NoError(metering.ValidateRateCard(metering.RateCard{
		ResourceType: "r", PricePerUnit: 0, Currency: "USD", Unit: "hour",
	}))
}

func (s *MeteringTestSuite) TestEventedMeterPublishes() {
	bus := eventsmem.New(events.Config{})
	defer bus.Close()

	var got []events.Event
	var mu sync.Mutex
	_, err := bus.Subscribe(s.Ctx, metering.TopicUsage, func(ctx context.Context, e events.Event) error {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
		return nil
	})
	s.NoError(err)

	inner := memory.New()
	defer inner.Close()
	meter := metering.NewEventedMeter(inner, bus)

	ev := s.validEvent("tenant-evt", "compute.instance.small", 3)
	ev.ID = "evt-1"
	s.NoError(meter.RecordUsage(s.Ctx, ev))

	mu.Lock()
	defer mu.Unlock()
	s.Len(got, 1)
	s.Equal(metering.EventTypeUsageRecorded, got[0].Type)
	s.Equal("pkg/metering", got[0].Source)
	s.Equal("evt-1", got[0].ID)

	payload, ok := got[0].Payload.(metering.UsageRecordedPayload)
	s.True(ok)
	s.Equal("tenant-evt", payload.TenantID)
	s.Equal(3.0, payload.Quantity)

	stored, err := meter.GetUsage(s.Ctx, metering.UsageFilter{TenantID: "tenant-evt"})
	s.NoError(err)
	s.Len(stored, 1)
	s.Equal("evt-1", stored[0].ID)
}

func (s *MeteringTestSuite) TestEventedMeterNilBus() {
	inner := memory.New()
	defer inner.Close()
	meter := metering.NewEventedMeter(inner, nil)
	s.NoError(meter.RecordUsage(s.Ctx, s.validEvent("t1", "compute.instance.small", 1)))
}

func (s *MeteringTestSuite) TestEventedMeterSkipsPublishOnError() {
	bus := eventsmem.New(events.Config{})
	defer bus.Close()

	var count int
	_, err := bus.Subscribe(s.Ctx, metering.TopicUsage, func(ctx context.Context, e events.Event) error {
		count++
		return nil
	})
	s.NoError(err)

	meter := metering.NewEventedMeter(memory.New(), bus)
	err = meter.RecordUsage(s.Ctx, metering.UsageEvent{Quantity: 1})
	s.True(errors.Is(err, metering.ErrInvalidUsage))
	s.Equal(0, count)
}

func (s *MeteringTestSuite) TestRateCardCRUDAndHistory() {
	card := metering.RateCard{
		ResourceType: "gpu.a100",
		PricePerUnit: 3.5,
		Currency:     "USD",
		Unit:         "hour",
	}
	s.NoError(s.rater.SetRate(s.Ctx, card))

	err := s.rater.UpdateRate(s.Ctx, metering.RateCard{
		ResourceType: "missing.res",
		PricePerUnit: 1,
		Currency:     "USD",
		Unit:         "hour",
	})
	s.True(errors.Is(err, metering.ErrRateNotFound))

	s.NoError(s.rater.UpdateRate(s.Ctx, metering.RateCard{
		ResourceType: "gpu.a100",
		PricePerUnit: 4.0,
		Currency:     "USD",
		Unit:         "hour",
	}))
	rate, err := s.rater.GetRate(s.Ctx, "gpu.a100")
	s.NoError(err)
	s.Equal(4.0, rate.PricePerUnit)

	hist, err := s.rater.ListRateHistory(s.Ctx, "gpu.a100")
	s.NoError(err)
	s.GreaterOrEqual(len(hist), 2)
	s.Equal(metering.RateOpSet, hist[0].Op)
	s.Equal(metering.RateOpUpdate, hist[1].Op)

	s.NoError(s.rater.DeleteRate(s.Ctx, "gpu.a100"))
	_, err = s.rater.GetRate(s.Ctx, "gpu.a100")
	s.True(errors.Is(err, metering.ErrRateNotFound))

	hist, err = s.rater.ListRateHistory(s.Ctx, "gpu.a100")
	s.NoError(err)
	s.Equal(metering.RateOpDelete, hist[len(hist)-1].Op)

	err = s.rater.DeleteRate(s.Ctx, "gpu.a100")
	s.True(errors.Is(err, metering.ErrRateNotFound))
}

func (s *MeteringTestSuite) TestSummarizeAndPeriodAggregate() {
	base := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	events := []metering.UsageEvent{
		{TenantID: "t1", ResourceType: "compute.instance.small", ResourceID: "a", Quantity: 2, Timestamp: base},
		{TenantID: "t1", ResourceType: "compute.instance.small", ResourceID: "b", Quantity: 3, Timestamp: base.Add(30 * time.Minute)},
		{TenantID: "t1", ResourceType: "storage.standard", ResourceID: "c", Quantity: 10, Timestamp: base.Add(2 * time.Hour)},
	}
	for _, e := range events {
		s.NoError(s.meter.RecordUsage(s.Ctx, e))
	}

	sum, err := s.meter.SummarizeUsage(s.Ctx, metering.UsageFilter{TenantID: "t1"})
	s.NoError(err)
	s.Equal(15.0, sum.TotalQuantity)
	s.Equal(3, sum.EventCount)
	s.Equal(5.0, sum.ByResourceType["compute.instance.small"])
	s.Equal(10.0, sum.ByResourceType["storage.standard"])

	buckets, err := s.meter.PeriodAggregate(s.Ctx, metering.UsageFilter{TenantID: "t1"}, time.Hour)
	s.NoError(err)
	s.True(len(buckets) >= 2)

	_, err = s.meter.PeriodAggregate(s.Ctx, metering.UsageFilter{}, 0)
	s.True(errors.Is(err, metering.ErrInvalidUsage))
}

func (s *MeteringTestSuite) TestInstrumentedRaterClose() {
	s.NoError(s.rater.Close())
}

func TestMeteringSuite(t *testing.T) {
	test.Run(t, new(MeteringTestSuite))
}

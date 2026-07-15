package cqrs_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/cqrs"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

type createOrderCmd struct{ Name string }

func (c createOrderCmd) CommandName() string { return "CreateOrder" }

type getOrderQuery struct{ ID string }

func (q getOrderQuery) QueryName() string { return "GetOrder" }

type createOrderHandler struct {
	called atomic.Int32
	err    error
}

func (h *createOrderHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
	h.called.Add(1)
	return h.err
}

type getOrderHandler struct {
	result interface{}
	err    error
}

func (h *getOrderHandler) Handle(ctx context.Context, query cqrs.Query) (interface{}, error) {
	return h.result, h.err
}

func TestCommandBusDispatch(t *testing.T) {
	bus := cqrs.NewCommandBus()
	h := &createOrderHandler{}
	bus.Register("CreateOrder", h)

	if err := bus.Dispatch(context.Background(), createOrderCmd{Name: "x"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if h.called.Load() != 1 {
		t.Fatalf("handler not called")
	}
}

func TestCommandBusRegisterCommand(t *testing.T) {
	bus := cqrs.NewCommandBus()
	h := &createOrderHandler{}
	bus.RegisterCommand(createOrderCmd{}, h)

	if err := bus.Dispatch(context.Background(), createOrderCmd{}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
}

func TestCommandBusHandlerNotFound(t *testing.T) {
	bus := cqrs.NewCommandBus()
	err := bus.Dispatch(context.Background(), createOrderCmd{})
	if err == nil {
		t.Fatal("expected not found")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != cqrs.CodeHandlerNotFound {
		t.Fatalf("expected handler not found, got %v", err)
	}
}

func TestCommandBusNilCommand(t *testing.T) {
	bus := cqrs.NewCommandBus()
	err := bus.Dispatch(context.Background(), nil)
	if err == nil {
		t.Fatal("expected invalid command")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != cqrs.CodeInvalidCommand {
		t.Fatalf("expected invalid command, got %v", err)
	}
}

func TestQueryBusDispatch(t *testing.T) {
	bus := cqrs.NewQueryBus()
	bus.Register("GetOrder", &getOrderHandler{result: map[string]string{"id": "1"}})

	result, err := bus.Dispatch(context.Background(), getOrderQuery{ID: "1"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	m, ok := result.(map[string]string)
	if !ok || m["id"] != "1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestQueryBusHandlerNotFound(t *testing.T) {
	bus := cqrs.NewQueryBus()
	_, err := bus.Dispatch(context.Background(), getOrderQuery{})
	if err == nil {
		t.Fatal("expected not found")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != cqrs.CodeHandlerNotFound {
		t.Fatalf("expected handler not found, got %v", err)
	}
}

func TestQueryBusNilQuery(t *testing.T) {
	bus := cqrs.NewQueryBus()
	_, err := bus.Dispatch(context.Background(), nil)
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != cqrs.CodeInvalidQuery {
		t.Fatalf("expected invalid query, got %v", err)
	}
}

func TestCommandBusConcurrentRegisterAndDispatch(t *testing.T) {
	bus := cqrs.NewCommandBus()
	h := &createOrderHandler{}
	bus.Register("CreateOrder", h)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			bus.Register("CreateOrder", h)
		}()
		go func() {
			defer wg.Done()
			_ = bus.Dispatch(context.Background(), createOrderCmd{})
		}()
	}
	wg.Wait()
	if h.called.Load() == 0 {
		t.Fatal("expected some dispatches")
	}
}

// commandHandlerFunc adapts a function to CommandHandler for tests.
type commandHandlerFunc func(ctx context.Context, cmd cqrs.Command) error

func (f commandHandlerFunc) Handle(ctx context.Context, cmd cqrs.Command) error {
	return f(ctx, cmd)
}

func TestCommandMiddleware(t *testing.T) {
	var order []string
	base := &createOrderHandler{}

	mw1 := func(next cqrs.CommandHandler) cqrs.CommandHandler {
		return commandHandlerFunc(func(ctx context.Context, cmd cqrs.Command) error {
			order = append(order, "mw1")
			return next.Handle(ctx, cmd)
		})
	}
	mw2 := func(next cqrs.CommandHandler) cqrs.CommandHandler {
		return commandHandlerFunc(func(ctx context.Context, cmd cqrs.Command) error {
			order = append(order, "mw2")
			return next.Handle(ctx, cmd)
		})
	}

	handler := cqrs.WithCommandMiddleware(base, mw1, mw2)
	if err := handler.Handle(context.Background(), createOrderCmd{}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(order) != 2 || order[0] != "mw1" || order[1] != "mw2" {
		t.Fatalf("middleware order: %v", order)
	}
	if base.called.Load() != 1 {
		t.Fatal("base handler not called")
	}
}

func TestLoggingCommandMiddleware(t *testing.T) {
	var logs []string
	base := &createOrderHandler{}
	mw := cqrs.NewLoggingCommandMiddleware(func(format string, args ...interface{}) {
		logs = append(logs, format)
	})
	handler := mw(base)
	if err := handler.Handle(context.Background(), createOrderCmd{}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected log lines, got %v", logs)
	}
}

func TestInstrumentedBuses(t *testing.T) {
	cmdBus := cqrs.NewInstrumentedCommandBus(cqrs.NewCommandBus())
	h := &createOrderHandler{}
	cmdBus.Register("CreateOrder", h)
	if err := cmdBus.Dispatch(context.Background(), createOrderCmd{}); err != nil {
		t.Fatalf("command Dispatch: %v", err)
	}

	qBus := cqrs.NewInstrumentedQueryBus(cqrs.NewQueryBus())
	qBus.Register("GetOrder", &getOrderHandler{result: "ok"})
	result, err := qBus.Dispatch(context.Background(), getOrderQuery{})
	if err != nil || result != "ok" {
		t.Fatalf("query Dispatch: %v %v", result, err)
	}
}

func TestInstrumentedCommandBusNotFound(t *testing.T) {
	bus := cqrs.NewInstrumentedCommandBus(cqrs.NewCommandBus())
	err := bus.Dispatch(context.Background(), createOrderCmd{})
	if err == nil {
		t.Fatal("expected error")
	}
}

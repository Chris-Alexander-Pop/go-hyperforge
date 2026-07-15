// Package cqrs provides Command Query Responsibility Segregation patterns.
//
// Separates read and write operations with commands, queries, and handlers.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/cqrs"
//
//	bus := cqrs.NewCommandBus()
//	bus.Register("CreateOrder", &CreateOrderHandler{})
//	err := bus.Dispatch(ctx, CreateOrderCommand{...})
package cqrs

import (
	"context"
	"reflect"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Command represents a command that mutates state.
type Command interface {
	CommandName() string
}

// Query represents a query that reads state.
type Query interface {
	QueryName() string
}

// CommandHandler handles a specific command type.
type CommandHandler interface {
	Handle(ctx context.Context, cmd Command) error
}

// QueryHandler handles a specific query type.
type QueryHandler interface {
	Handle(ctx context.Context, query Query) (interface{}, error)
}

// CommandBus dispatches commands to handlers.
type CommandBus struct {
	handlers map[string]CommandHandler
	mu       *concurrency.SmartRWMutex
}

// NewCommandBus creates a new command bus.
func NewCommandBus() *CommandBus {
	return &CommandBus{
		handlers: make(map[string]CommandHandler),
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "cqrs-command-bus"}),
	}
}

// Register registers a handler for a command type.
func (b *CommandBus) Register(commandName string, handler CommandHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[commandName] = handler
}

// RegisterCommand registers a handler using the command's name.
func (b *CommandBus) RegisterCommand(cmd Command, handler CommandHandler) {
	b.Register(cmd.CommandName(), handler)
}

// Dispatch sends a command to its handler.
func (b *CommandBus) Dispatch(ctx context.Context, cmd Command) error {
	if cmd == nil {
		return ErrInvalidCommand("command is nil", nil)
	}

	b.mu.RLock()
	handler, ok := b.handlers[cmd.CommandName()]
	b.mu.RUnlock()

	if !ok {
		return ErrCommandHandlerNotFound(cmd.CommandName())
	}

	return handler.Handle(ctx, cmd)
}

// QueryBus dispatches queries to handlers.
type QueryBus struct {
	handlers map[string]QueryHandler
	mu       *concurrency.SmartRWMutex
}

// NewQueryBus creates a new query bus.
func NewQueryBus() *QueryBus {
	return &QueryBus{
		handlers: make(map[string]QueryHandler),
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "cqrs-query-bus"}),
	}
}

// Register registers a handler for a query type.
func (b *QueryBus) Register(queryName string, handler QueryHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[queryName] = handler
}

// RegisterQuery registers a handler using the query's name.
func (b *QueryBus) RegisterQuery(query Query, handler QueryHandler) {
	b.Register(query.QueryName(), handler)
}

// Dispatch sends a query to its handler and returns the result.
func (b *QueryBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
	if query == nil {
		return nil, ErrInvalidQuery("query is nil", nil)
	}

	b.mu.RLock()
	handler, ok := b.handlers[query.QueryName()]
	b.mu.RUnlock()

	if !ok {
		return nil, ErrQueryHandlerNotFound(query.QueryName())
	}

	return handler.Handle(ctx, query)
}

// BaseCommand provides default command name from struct type.
type BaseCommand struct{}

// CommandName returns the struct type name as command name.
func (c BaseCommand) CommandName() string {
	return reflect.TypeOf(c).Name()
}

// BaseQuery provides default query name from struct type.
type BaseQuery struct{}

// QueryName returns the struct type name as query name.
func (q BaseQuery) QueryName() string {
	return reflect.TypeOf(q).Name()
}

// CommandMiddleware wraps command handling.
type CommandMiddleware func(next CommandHandler) CommandHandler

// QueryMiddleware wraps query handling.
type QueryMiddleware func(next QueryHandler) QueryHandler

// WithCommandMiddleware wraps a handler with middleware.
func WithCommandMiddleware(handler CommandHandler, middlewares ...CommandMiddleware) CommandHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// WithQueryMiddleware wraps a handler with middleware.
func WithQueryMiddleware(handler QueryHandler, middlewares ...QueryMiddleware) QueryHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// LoggingCommandMiddleware logs command execution.
type LoggingCommandMiddleware struct {
	next   CommandHandler
	logger func(format string, args ...interface{})
}

// NewLoggingCommandMiddleware creates a logging middleware.
func NewLoggingCommandMiddleware(logger func(format string, args ...interface{})) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return &LoggingCommandMiddleware{next: next, logger: logger}
	}
}

// Handle logs and delegates to the next handler.
func (m *LoggingCommandMiddleware) Handle(ctx context.Context, cmd Command) error {
	m.logger("Executing command: %s", cmd.CommandName())
	err := m.next.Handle(ctx, cmd)
	if err != nil {
		m.logger("Command %s failed: %v", cmd.CommandName(), err)
	} else {
		m.logger("Command %s succeeded", cmd.CommandName())
	}
	return err
}

// ReadModel represents a denormalized read model.
type ReadModel interface {
	// ID returns the unique identifier.
	ID() string
}

// ReadModelRepository stores read models.
type ReadModelRepository interface {
	// Save persists a read model.
	Save(ctx context.Context, model ReadModel) error

	// FindByID retrieves a read model by ID.
	FindByID(ctx context.Context, id string) (ReadModel, error)

	// Delete removes a read model.
	Delete(ctx context.Context, id string) error
}

// Projector creates read models from domain events.
type Projector interface {
	// Project applies an event to update read models.
	Project(ctx context.Context, event interface{}) error

	// EventTypes returns the event types this projector handles.
	EventTypes() []string
}

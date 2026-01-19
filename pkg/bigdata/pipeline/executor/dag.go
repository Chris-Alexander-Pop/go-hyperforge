package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/events"
)

// Task is a unit of work.
type Task func(ctx context.Context) error

// Node in the DAG.
type Node struct {
	ID        string
	Task      Task
	DependsOn []string
}

// DAGExecutor runs tasks respecting dependencies.
type DAGExecutor struct {
	nodes map[string]*Node
	bus   events.Bus
}

// New creates a new DAGExecutor.
// bus is optional (can be nil).
func New(bus events.Bus) *DAGExecutor {
	return &DAGExecutor{
		nodes: make(map[string]*Node),
		bus:   bus,
	}
}

func (d *DAGExecutor) AddTask(id string, task Task, dependsOn ...string) {
	d.nodes[id] = &Node{
		ID:        id,
		Task:      task,
		DependsOn: dependsOn,
	}
}

func (d *DAGExecutor) Run(ctx context.Context) error {
	// Simple BFS/TopSort execution.
	// 1. Calculate in-degrees
	inDegree := make(map[string]int)
	graph := make(map[string][]string) // u -> [v]

	for id, node := range d.nodes {
		inDegree[id] = 0
		for _, dep := range node.DependsOn {
			graph[dep] = append(graph[dep], id)
			inDegree[id]++
		}
	}

	// 2. Queue zero in-degree nodes
	var queue []string
	for id := range d.nodes {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	// 3. Execute layer by layer (parallelizable)
	var mu sync.Mutex
	var errs []error

	active := 0
	completed := 0
	total := len(d.nodes)

	// Channel to signal completion of a task
	doneCh := make(chan string, total)

	// Start initial
	for _, id := range queue {
		d.startTask(ctx, id, doneCh, &mu, &errs)
		active++
	}

	for completed < total {
		if active == 0 && completed < total {
			return errors.Internal("cycle detected or deadlock", nil)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case id := <-doneCh:
			active--
			completed++

			// Decrement neighbors
			children := graph[id]
			for _, child := range children {
				inDegree[child]--
				if inDegree[child] == 0 {
					d.startTask(ctx, child, doneCh, &mu, &errs)
					active++
				}
			}
		}

		mu.Lock()
		if len(errs) > 0 {
			mu.Unlock()
			return errs[0] // Return first error
		}
		mu.Unlock()
	}

	return nil
}

func (d *DAGExecutor) startTask(ctx context.Context, id string, doneCh chan<- string, mu *sync.Mutex, errs *[]error) {
	node := d.nodes[id]
	go func() {
		// Emit Started
		if d.bus != nil {
			_ = d.bus.Publish(ctx, "bigdata.task.started", events.Event{
				ID:        fmt.Sprintf("evt-%s-start-%d", id, time.Now().UnixNano()),
				Type:      "bigdata.task.started",
				Source:    "bigdata-executor",
				Timestamp: time.Now(),
				Payload:   map[string]string{"task_id": id},
			})
		}

		err := node.Task(ctx)

		if err != nil {
			mu.Lock()
			*errs = append(*errs, err)
			mu.Unlock()

			// Emit Failed
			if d.bus != nil {
				_ = d.bus.Publish(ctx, "bigdata.task.failed", events.Event{
					ID:        fmt.Sprintf("evt-%s-fail-%d", id, time.Now().UnixNano()),
					Type:      "bigdata.task.failed",
					Source:    "bigdata-executor",
					Timestamp: time.Now(),
					Payload:   map[string]string{"task_id": id, "error": err.Error()},
				})
			}
		} else {
			// Emit Completed
			if d.bus != nil {
				_ = d.bus.Publish(ctx, "bigdata.task.completed", events.Event{
					ID:        fmt.Sprintf("evt-%s-comp-%d", id, time.Now().UnixNano()),
					Type:      "bigdata.task.completed",
					Source:    "bigdata-executor",
					Timestamp: time.Now(),
					Payload:   map[string]string{"task_id": id},
				})
			}
		}
		doneCh <- id
	}()
}

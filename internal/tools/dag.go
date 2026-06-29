package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/bimal009/Synapse/internal/models"
	"github.com/go-playground/validator/v10"
)

type dag struct {
	v *validator.Validate
}

type Dag interface {
	Validates(ctx context.Context, dag models.Dag) error
}

func NewDag() Dag {
	return &dag{v: validator.New()}
}

func (d *dag) Validates(ctx context.Context, dag models.Dag) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := d.v.StructCtx(ctx, dag); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			msgs := make([]string, 0, len(ve))
			for _, fe := range ve {
				msgs = append(msgs, fmt.Sprintf(
					"field %s failed %q (value: %v)",
					fe.Namespace(), fe.Tag(), fe.Value(),
				))
			}
			return fmt.Errorf("validation errors:\n  %s", strings.Join(msgs, "\n  "))
		}
		return err
	}

	idx := make(map[string]models.Task, len(dag.Tasks))
	for _, t := range dag.Tasks {
		if _, dup := idx[t.ID]; dup {
			return fmt.Errorf("duplicate task id: %q", t.ID)
		}
		idx[t.ID] = t
	}

	for _, t := range dag.Tasks {
		for _, dep := range t.Dependencies {
			if dep == t.ID {
				return fmt.Errorf("task %q depends on itself", t.ID)
			}
			if _, ok := idx[dep]; !ok {
				return fmt.Errorf("task %q has missing dependency %q", t.ID, dep)
			}
		}
	}

	order, err := topoSort(dag.Tasks)
	if err != nil {
		return err
	}

	writer := make(map[string]string)
	for _, t := range dag.Tasks {
		for _, out := range t.Outputs {
			if prev, ok := writer[out]; ok {
				return fmt.Errorf("output %q written by both %q and %q", out, prev, t.ID)
			}
			writer[out] = t.ID
		}
	}
	rank := make(map[string]int, len(order))
	for i, id := range order {
		rank[id] = i
	}
	for _, t := range dag.Tasks {
		for _, in := range t.Inputs {
			src, produced := writer[in]
			if !produced {
				if len(t.Dependencies) > 0 {
					return fmt.Errorf("task %q consumes %q but no task produces it", t.ID, in)
				}
				continue
			}
			if rank[src] >= rank[t.ID] {
				return fmt.Errorf("task %q needs %q but producer %q runs after", t.ID, in, src)
			}
		}
	}

	return nil
}

func topoSort(tasks []models.Task) ([]string, error) {
	indeg := make(map[string]int, len(tasks))
	children := make(map[string][]string, len(tasks))
	for _, t := range tasks {
		if _, ok := indeg[t.ID]; !ok {
			indeg[t.ID] = 0
		}
		for _, dep := range t.Dependencies {
			indeg[t.ID]++
			children[dep] = append(children[dep], t.ID)
		}
	}
	queue := make([]string, 0)
	for id, n := range indeg {
		if n == 0 {
			queue = append(queue, id)
		}
	}
	order := make([]string, 0, len(tasks))
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		order = append(order, n)
		for _, c := range children[n] {
			indeg[c]--
			if indeg[c] == 0 {
				queue = append(queue, c)
			}
		}
	}
	if len(order) != len(tasks) {
		stuck := []string{}
		for id, n := range indeg {
			if n > 0 {
				stuck = append(stuck, id)
			}
		}
		return nil, fmt.Errorf("cycle detected: %s", strings.Join(stuck, ", "))
	}
	return order, nil
}

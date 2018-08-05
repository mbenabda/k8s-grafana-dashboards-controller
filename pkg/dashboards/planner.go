package dashboards

import (
	"context"
	"fmt"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"strings"
)

type changesListPlanner struct {
}

type changesListPlan struct {
	Plan
	changes []change
}

type change interface {
	visit(context.Context, ApplyFuncs) error
}

type createAction struct {
	dashboard *grafana.Dashboard
}
type updateAction struct {
	dashboard *grafana.Dashboard
}
type deleteAction struct {
	slug string
}

func NewPlanner() Planner {
	return changesListPlanner{}
}

func (this changesListPlanner) Plan(ctx context.Context, current []*grafana.DashboardResult, desired []*grafana.Dashboard) Plan {
	plan := map[string]change{}

	currentSlugs := map[string]bool{}
	for _, dashboard := range current {
		slug, err := dashboard.Slug()
		if err != nil {
			continue
		}
		currentSlugs[slug] = true
		plan[slug] = deleteAction{
			slug: slug,
		}
	}

	for _, dashboard := range desired {
		slug, err := dashboard.Slug()
		if err != nil {
			continue
		}

		if currentSlugs[slug] {
			plan[slug] = updateAction{
				dashboard: dashboard,
			}
		} else {
			plan[slug] = createAction{
				dashboard: dashboard,
			}
		}
	}

	changes := []change{}
	for _, v := range plan {
		changes = append(changes, v)
	}

	return changesListPlan{changes: changes}
}

func (this changesListPlan) Apply(ctx context.Context, funcs ApplyFuncs) error {
	errors := map[change]error{}
	for _, change := range this.changes {
		err := change.visit(ctx, funcs)
		if err != nil {
			errors[change] = err
		}
	}

	if len(errors) > 0 {
		prettyErrors := []string{}

		for _, err := range errors {
			prettyErrors = append(prettyErrors, fmt.Sprintf("%v", err))
		}

		return fmt.Errorf(strings.Join(prettyErrors, "\n"))
	}

	return nil
}

func (this createAction) visit(ctx context.Context, visitor ApplyFuncs) error {
	return visitor.Create(ctx, this.dashboard)
}

func (this updateAction) visit(ctx context.Context, visitor ApplyFuncs) error {
	return visitor.Update(ctx, this.dashboard)
}

func (this deleteAction) visit(ctx context.Context, visitor ApplyFuncs) error {
	return visitor.Delete(ctx, this.slug)
}

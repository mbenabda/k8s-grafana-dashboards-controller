package dashboards

import (
	"context"
	log "github.com/sirupsen/logrus"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type Planner interface {
	Plan(ctx context.Context, current []*grafana.DashboardResult, desired []*grafana.Dashboard) Plan
}

type Plan interface {
	Apply(context.Context, DashboardChangesApplier) []error
}

type DashboardChangesApplier interface {
	Create(context.Context, *grafana.Dashboard) error
	Update(context.Context, *grafana.Dashboard) error
	Delete(context.Context, string) error
}

type changesListPlanner struct {
	logger *log.Entry
}

type changesListPlan struct {
	Plan
	changes []change
	logger  *log.Entry
}

type change interface {
	visit(context.Context, DashboardChangesApplier) error
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

func NewPlanner(logger *log.Entry) Planner {
	return changesListPlanner{
		logger: logger,
	}
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

	this.logger.Debugf("planed %d changes to current state", len(changes))

	return changesListPlan{
		logger:  this.logger,
		changes: changes,
	}
}

func (this changesListPlan) Apply(ctx context.Context, funcs DashboardChangesApplier) []error {
	var errors []error
	for _, change := range this.changes {
		err := change.visit(ctx, funcs)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func (this createAction) visit(ctx context.Context, visitor DashboardChangesApplier) error {
	return visitor.Create(ctx, this.dashboard)
}

func (this updateAction) visit(ctx context.Context, visitor DashboardChangesApplier) error {
	return visitor.Update(ctx, this.dashboard)
}

func (this deleteAction) visit(ctx context.Context, visitor DashboardChangesApplier) error {
	return visitor.Delete(ctx, this.slug)
}

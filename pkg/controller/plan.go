package controller

import (
	"context"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type World struct {
	Desired []*grafana.Dashboard
	Current []*grafana.DashboardResult
}

type Plan struct {
	Changes []change
}

type ApplyPlanFuncs struct {
	CreateFunc func(context.Context, *grafana.Dashboard) error
	UpdateFunc func(context.Context, *grafana.Dashboard) error
	DeleteFunc func(context.Context, string) error
}

type change interface {
	apply(context.Context, ApplyPlanFuncs) error
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

func (world *World) PlanChanges() Plan {
	plan := map[string]change{}

	currentSlugs := map[string]bool{}
	for _, dashboard := range world.Current {
		slug, err := dashboard.Slug()
		if err != nil {
			continue
		}
		currentSlugs[slug] = true
		plan[slug] = deleteAction{
			slug: slug,
		}
	}

	desiredSlugs := map[string]bool{}
	for _, dashboard := range world.Desired {
		slug, err := dashboard.Slug()
		if err != nil {
			continue
		}

		desiredSlugs[slug] = true
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

	return Plan{Changes: changes}
}

func (this Plan) Apply(ctx context.Context, funcs ApplyPlanFuncs) error {
	for _, change := range this.Changes {
		err := change.apply(ctx, funcs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this createAction) apply(ctx context.Context, funcs ApplyPlanFuncs) error {
	return funcs.CreateFunc(ctx, this.dashboard)
}

func (this updateAction) apply(ctx context.Context, funcs ApplyPlanFuncs) error {
	return funcs.UpdateFunc(ctx, this.dashboard)
}

func (this deleteAction) apply(ctx context.Context, funcs ApplyPlanFuncs) error {
	return funcs.DeleteFunc(ctx, this.slug)
}

package differ

import (
	"context"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type Funcs struct {
	ListDashboards func(ctx context.Context) ([]*grafana.DashboardResult, error)
	Create         func(context.Context, *grafana.Dashboard) error
	Update         func(context.Context, *grafana.Dashboard) error
	Delete         func(context.Context, string) error
}

func NewFuncsBased(funcs Funcs) Interface {
	return funcsBasedDiffer{funcs}
}

func (this funcsBasedDiffer) Apply(ctx context.Context, desired []*grafana.Dashboard) error {
	current, err := this.Funcs.ListDashboards(ctx)
	if err != nil {
		return err
	}

	for _, change := range plan(current, desired) {
		err := change.apply(ctx, this.Funcs)
		if err != nil {
			return err
		}
	}
	return nil
}

type funcsBasedDiffer struct {
	Funcs
}

type change interface {
	apply(context.Context, Funcs) error
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

func plan(current []*grafana.DashboardResult, desired []*grafana.Dashboard) []change {
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

	desiredSlugs := map[string]bool{}
	for _, dashboard := range desired {
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

	return changes
}

func (this createAction) apply(ctx context.Context, funcs Funcs) error {
	return funcs.Create(ctx, this.dashboard)
}

func (this updateAction) apply(ctx context.Context, funcs Funcs) error {
	return funcs.Update(ctx, this.dashboard)
}

func (this deleteAction) apply(ctx context.Context, funcs Funcs) error {
	return funcs.Delete(ctx, this.slug)
}

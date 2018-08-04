package differ

import (
	"context"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type simplePlanner struct {
}

func NewPlanner() DashboardsChangesPlanner {
	return simplePlanner{}
}

func (this simplePlanner) Plan(ctx context.Context, current []*grafana.DashboardResult, desired []*grafana.Dashboard) Plan {
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

	return Plan{changes: changes}
}

package differ

import (
	"context"
	"fmt"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"strings"
)

type DashboardsChangesPlanner interface {
	Plan(ctx context.Context, current []*grafana.DashboardResult, desired []*grafana.Dashboard) Plan
}

type Plan struct {
	changes []change
}

func (this Plan) Apply(ctx context.Context, funcs DashboardsChangesApplyFuncs) error {
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

type DashboardsChangesApplyFuncs struct {
	Create func(context.Context, *grafana.Dashboard) error
	Update func(context.Context, *grafana.Dashboard) error
	Delete func(context.Context, string) error
}

var NoOpPlanApplyFuncs = DashboardsChangesApplyFuncs{
	Create: func(ctx context.Context, dash *grafana.Dashboard) error {
		slug, _ := dash.Slug()
		log.Printf("created dashboard %v\n", slug)
		return nil
	},
	Update: func(ctx context.Context, dash *grafana.Dashboard) error {
		slug, _ := dash.Slug()
		log.Printf("updated dashboard %v\n", slug)
		return nil
	},
	Delete: func(ctx context.Context, slug string) error {
		log.Printf("deleted dashboard %v\n", slug)
		return nil
	},
}

func NewClientBasedPlanApplyFuncs(dashboards grafana.DashboardsInterface) DashboardsChangesApplyFuncs {
	return DashboardsChangesApplyFuncs{
		Create: func(ctx context.Context, dash *grafana.Dashboard) error {
			slug, err := dash.Slug()
			if err != nil {
				return fmt.Errorf("unable to get slug of dashboard %v", dash)
			}
			err = dashboards.Import(ctx, dash)
			if err != nil {
				return fmt.Errorf("unable to create dashboard %v: %v", slug, err)
			}
			return nil
		},
		Update: func(ctx context.Context, dash *grafana.Dashboard) error {
			slug, err := dash.Slug()
			if err != nil {
				return fmt.Errorf("unable to get slug of dashboard %v", dash)
			}
			err = dashboards.ImportAndOverwrite(ctx, dash)
			if err != nil {
				return fmt.Errorf("unable to update dashboard %v: %v", slug, err)
			}
			return nil
		},
		Delete: func(ctx context.Context, slug string) error {
			err := dashboards.Delete(ctx, slug)
			if err != nil {
				return fmt.Errorf("unable to delete dashboard %v: %v", slug, err)
			}
			return nil
		},
	}
}

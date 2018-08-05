package dashboards

import (
	"context"
	"fmt"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type Planner interface {
	Plan(ctx context.Context, current []*grafana.DashboardResult, desired []*grafana.Dashboard) Plan
}

type Plan interface {
	Apply(ctx context.Context, funcs ApplyFuncs) error
}

type ApplyFuncs struct {
	Create func(context.Context, *grafana.Dashboard) error
	Update func(context.Context, *grafana.Dashboard) error
	Delete func(context.Context, string) error
}

var NoOpPlanApplyFuncs = ApplyFuncs{
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

func NewClientBasedPlanApplyFuncs(dashboards grafana.DashboardsInterface) ApplyFuncs {
	return ApplyFuncs{
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

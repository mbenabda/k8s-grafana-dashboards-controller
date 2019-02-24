package dashboards

import (
	"context"
	"fmt"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type applier struct {
	dashboards grafana.DashboardsInterface
}

func NewApplier(dashboards grafana.DashboardsInterface) DashboardChangesApplier {
	return applier{dashboards: dashboards}
}

func (this applier) Create(ctx context.Context, dash *grafana.Dashboard) error {
	slug, err := dash.Slug()
	if err != nil {
		return fmt.Errorf("unable to get slug of dashboard %v", dash)
	}
	err = this.dashboards.Import(ctx, dash)
	if err != nil {
		return fmt.Errorf("unable to create dashboard %v: %v", slug, err)
	}
	return nil
}

func (this applier) Update(ctx context.Context, dash *grafana.Dashboard) error {
	slug, err := dash.Slug()
	if err != nil {
		return fmt.Errorf("unable to get slug of dashboard %v", dash)
	}
	err = this.dashboards.ImportAndOverwrite(ctx, dash)
	if err != nil {
		return fmt.Errorf("unable to update dashboard %v: %v", slug, err)
	}
	return nil
}

func (this applier) Delete(ctx context.Context, slug string) error {
	err := this.dashboards.Delete(ctx, slug)
	if err != nil {
		return fmt.Errorf("unable to delete dashboard %v: %v", slug, err)
	}
	return nil
}

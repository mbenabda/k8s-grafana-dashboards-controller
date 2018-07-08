package grafana

import (
	"context"
)

type Interface interface {
	Dashboards() DashboardsInterface
}

type DashboardsInterface interface {
	Import(ctx context.Context, dashboard Dashboard) error
	Delete(ctx context.Context, slug string) error
}

type GrafanaClient struct {
	dashboards DashboardsInterface
}

func (c GrafanaClient) Dashboards() DashboardsInterface {
	return c.dashboards
}

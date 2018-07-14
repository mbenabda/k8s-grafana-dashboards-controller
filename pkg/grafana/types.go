package grafana

import (
	"context"
)

type Interface interface {
	Dashboards() DashboardsInterface
}

type DashboardSearchQuery struct {
	Tags []string
}

type DashboardsInterface interface {
	Import(ctx context.Context, dashboard *Dashboard) error
	ImportAndOverwrite(ctx context.Context, dashboard *Dashboard) error
	Delete(ctx context.Context, slug string) error
	Search(ctx context.Context, query DashboardSearchQuery) ([]*DashboardResult, error)
}

type GrafanaClient struct {
	dashboards DashboardsInterface
}

func (c GrafanaClient) Dashboards() DashboardsInterface {
	return c.dashboards
}

package grafana

type Interface interface {
	Dashboards() DashboardsInterface
}

type DashboardsInterface interface {
	Import(dashboard Dashboard) error
	Delete(slug string) error
}

type GrafanaClient struct {
	dashboards DashboardsInterface
}

func (c GrafanaClient) Dashboards() DashboardsInterface {
	return c.dashboards
}

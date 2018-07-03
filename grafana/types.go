package grafana

type Dashboard map[string]struct{}

type Interface interface {
	Dashboards() DashboardsInterface
}

type DashboardsInterface interface {
	Import(dashboard Dashboard) error
	Delete(slug string) error
	Search(title string) ([]Dashboard, error)
}

type GrafanaClient struct {
	dashboards DashboardsInterface
}

func (c GrafanaClient) Dashboards() DashboardsInterface {
	return c.dashboards
}

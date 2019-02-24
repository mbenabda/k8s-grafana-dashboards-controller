package dashboards

import (
	"context"
	log "github.com/sirupsen/logrus"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type dryRun struct {
	logger *log.Logger
}

func NewDryRunApplier(logger *log.Logger) DashboardChangesApplier {
	return dryRun{logger}
}

func (this dryRun) Create(ctx context.Context, dash *grafana.Dashboard) error {
	slug, _ := dash.Slug()
	this.logger.Infof("created dashboard %v\n", slug)
	return nil
}

func (this dryRun) Update(ctx context.Context, dash *grafana.Dashboard) error {
	slug, _ := dash.Slug()
	this.logger.Infof("updated dashboard %v\n", slug)
	return nil
}

func (this dryRun) Delete(ctx context.Context, slug string) error {
	this.logger.Infof("deleted dashboard %v\n", slug)
	return nil
}

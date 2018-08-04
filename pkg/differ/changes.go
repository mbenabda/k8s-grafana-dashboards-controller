package differ

import (
	"context"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type change interface {
	visit(context.Context, DashboardsChangesApplyFuncs) error
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

func (this createAction) visit(ctx context.Context, visitor DashboardsChangesApplyFuncs) error {
	return visitor.Create(ctx, this.dashboard)
}

func (this updateAction) visit(ctx context.Context, visitor DashboardsChangesApplyFuncs) error {
	return visitor.Update(ctx, this.dashboard)
}

func (this deleteAction) visit(ctx context.Context, visitor DashboardsChangesApplyFuncs) error {
	return visitor.Delete(ctx, this.slug)
}

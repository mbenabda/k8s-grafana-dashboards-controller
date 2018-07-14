package differ

import (
	"context"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type Interface interface {
	Apply(ctx context.Context, desired []*grafana.Dashboard) error
}

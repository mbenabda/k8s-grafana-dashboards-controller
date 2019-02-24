package dashboards

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"time"
)

type dashboardsController struct {
	dashboards grafana.DashboardsInterface
	configmaps cache.SharedIndexInformer
	logger     *log.Logger
	markerTag  string
	planner    Planner
	applyFuncs DashboardChangesApplier
}

func NewController(logger *log.Logger, dashboards grafana.DashboardsInterface, configmapsInformer cache.SharedIndexInformer, markerTag string, planner Planner, applyFuncs DashboardChangesApplier) *dashboardsController {
	return &dashboardsController{
		dashboards: dashboards,
		configmaps: configmapsInformer,
		logger:     logger,
		markerTag:  markerTag,
		planner:    planner,
		applyFuncs: applyFuncs,
	}
}

func (c *dashboardsController) Run(ctx context.Context) {
	if !cache.WaitForCacheSync(ctx.Done(), c.configmaps.HasSynced) {
		return
	}

	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()

		processWorkItem := func(ctx context.Context) bool {
			c.logger.Infoln("reconciling")

			err := c.reconcile(ctx)
			if err != nil {
				c.logger.Errorf("Reconciliation failed: %v\n", err)
			} else {
				c.logger.Info("Reconciliation successful")
			}

			return true

		}

		for processWorkItem(ctx) {
			select {
			case <-t.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
}

func (c *dashboardsController) reconcile(ctx context.Context) error {
	currentDashboards, err := findManagedDashboards(ctx, c.dashboards, c.markerTag)
	if err != nil {
		return fmt.Errorf("failed to list dashboards declared in Grafana: %v", err)
	}
	c.logger.Debugf("%d managed dashboards found", len(currentDashboards))

	desiredDashboards, err := listDesiredDashboards(ctx, c.logger, c.configmaps, c.markerTag)
	if err != nil {
		return fmt.Errorf("failed to list dashboards declared in the ConfigMaps: %v", err)
	}
	c.logger.Debugf("%d desired dashboards found", len(desiredDashboards))

	errors := c.planner.
		Plan(ctx, currentDashboards, desiredDashboards).
		Apply(ctx, c.applyFuncs)

	if errors != nil {
		return fmt.Errorf("failed to apply plan : %v", errors)
	}

	return nil
}

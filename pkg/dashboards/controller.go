package dashboards

import (
	"context"
	"fmt"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"os"
	"reflect"
	"time"
)

type dashboardsController struct {
	dashboards  grafana.DashboardsInterface
	configmaps  cache.SharedIndexInformer
	errorLogger *log.Logger
	markerTag   string
	planner     Planner
	applyFuncs  ApplyFuncs
}

func NewController(dashboards grafana.DashboardsInterface, configmapsInformer cache.SharedIndexInformer, markerTag string, planner Planner, applyFuncs ApplyFuncs) *dashboardsController {
	return &dashboardsController{
		dashboards:  dashboards,
		configmaps:  configmapsInformer,
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
		markerTag:   markerTag,
		planner:     planner,
		applyFuncs:  applyFuncs,
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
			log.Println("reconciling")

			err := c.reconcile(ctx)
			if err != nil {
				c.errorLogger.Printf("%v\n", err)
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

	desiredDashboards, err := listDesiredDashboards(ctx, c.errorLogger, c.configmaps, c.markerTag)
	if err != nil {
		return fmt.Errorf("failed to list dashboards declared in the ConfigMaps: %v", err)
	}

	err = c.planner.
		Plan(ctx, currentDashboards, desiredDashboards).
		Apply(ctx, c.applyFuncs)

	if err != nil {
		return fmt.Errorf("failed to apply plan : %v", err)
	}

	log.Println("Grafana's dashboards have been updated")
	return nil
}

func findManagedDashboards(ctx context.Context, dashboards grafana.DashboardsInterface, markerTag string) ([]*grafana.DashboardResult, error) {
	tags := []string{}
	if markerTag != "" {
		tags = append(tags, markerTag)
	}

	return dashboards.Search(ctx, grafana.DashboardSearchQuery{
		Tags: tags,
	})
}

func listDesiredDashboards(ctx context.Context, errorLogger *log.Logger, configmaps cache.SharedIndexInformer, markerTag string) ([]*grafana.Dashboard, error) {
	desiredDashboards := []*grafana.Dashboard{}
	for _, cmObj := range configmaps.GetStore().List() {
		cm := cmObj.(*v1.ConfigMap)
		key, err := keyOf(cm)
		if err != nil {
			errorLogger.Printf("failed get key of ConfigMap %v: %v\n", cm, err)
			continue
		}

		desiredDashboard, err := asDashboard(cm)
		if err != nil {
			errorLogger.Printf("could not make a Dashboard out of ConfigMap %v: %v\n", key, err)
			continue
		}

		if err = desiredDashboard.AddTag(markerTag); err != nil {
			errorLogger.Printf("could not add marker tag to Dashboard of ConfigMap %v: %v\n", key, err)
			continue
		}

		desiredDashboards = append(desiredDashboards, desiredDashboard)
	}

	return desiredDashboards, nil
}

func keyOf(cm *v1.ConfigMap) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(cm)
}

func asDashboard(cm *v1.ConfigMap) (*grafana.Dashboard, error) {
	data := cm.Data
	keys := reflect.ValueOf(data).MapKeys()
	if len(keys) > 0 {
		firstKeyValue := data[keys[0].String()]
		return grafana.NewDashboard([]byte(firstKeyValue))
	}
	key, _ := keyOf(cm)
	return nil, fmt.Errorf("could not get first Data key of ConfigMap %v", key)
}

package controller

import (
	"context"
	"fmt"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/differ"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"os"
	"reflect"
	"time"
)

type DashboardsController struct {
	dashboards  grafana.DashboardsInterface
	configmaps  cache.SharedIndexInformer
	errorLogger *log.Logger
	markerTag   string
	reconciler  differ.Interface
}

func New(dashboards grafana.DashboardsInterface, configmaps cache.SharedIndexInformer, markerTag string, reconciler differ.Interface) *DashboardsController {
	return &DashboardsController{
		dashboards:  dashboards,
		configmaps:  configmaps,
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
		markerTag:   markerTag,
		reconciler:  reconciler,
	}
}

func (c *DashboardsController) Run(ctx context.Context) {
	if !cache.WaitForCacheSync(ctx.Done(), c.configmaps.HasSynced) {
		return
	}

	go func() {
		t := time.NewTicker(60 * time.Second)
		for c.processWorkItem(ctx) {
			select {
			case <-t.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
}

func (c *DashboardsController) processWorkItem(ctx context.Context) bool {
	log.Println("reconciling")

	currentDashboards, err := findManagedDashboards(ctx, c.dashboards, c.markerTag)
	if err != nil {
		c.errorLogger.Printf("failed to list dashboards declared in Grafana: %v\n", err)
		return true
	}

	desiredDashboards, err := listDesiredDashboards(ctx, c.errorLogger, c.dashboards, c.configmaps, c.markerTag)
	if err != nil {
		c.errorLogger.Printf("failed to list dashboards declared in the ConfigMaps: %v\n", err)
		return true
	}

	if err = c.reconciler.Apply(ctx, currentDashboards, desiredDashboards); err != nil {
		c.errorLogger.Printf("failed to apply plan : %v\n", err)
	} else {
		log.Println("reconciliation was successful")
	}

	return true
}

func findManagedDashboards(ctx context.Context, dashboards grafana.DashboardsInterface, markerTag string) ([]*grafana.DashboardResult, error) {
	var query grafana.DashboardSearchQuery

	if markerTag == "" {
		query = grafana.DashboardSearchQuery{}
	} else {
		query = grafana.DashboardSearchQuery{
			Tags: []string{markerTag},
		}
	}
	return dashboards.Search(ctx, query)
}

func listDesiredDashboards(ctx context.Context, errorLogger *log.Logger, dashboards grafana.DashboardsInterface, configmaps cache.SharedIndexInformer, markerTag string) ([]*grafana.Dashboard, error) {
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

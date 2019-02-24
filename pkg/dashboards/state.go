package dashboards

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"reflect"
)

func findManagedDashboards(ctx context.Context, dashboards grafana.DashboardsInterface, markerTag string) ([]*grafana.DashboardResult, error) {
	tags := []string{}
	if markerTag != "" {
		tags = append(tags, markerTag)
	}

	return dashboards.Search(ctx, grafana.DashboardSearchQuery{
		Tags: tags,
	})
}

func listDesiredDashboards(ctx context.Context, logger *log.Logger, configmaps cache.SharedIndexInformer, markerTag string) ([]*grafana.Dashboard, error) {
	desiredDashboards := []*grafana.Dashboard{}
	for _, cmObj := range configmaps.GetStore().List() {
		cm := cmObj.(*v1.ConfigMap)
		key, err := keyOf(cm)
		if err != nil {
			logger.Warnf("failed get key of ConfigMap %v: %v\n", cm, err)
			continue
		}

		desiredDashboard, err := asDashboard(cm)
		if err != nil {
			logger.Warnf("could not make a Dashboard out of ConfigMap %v: %v\n", key, err)
			continue
		}

		if err = desiredDashboard.AddTag(markerTag); err != nil {
			logger.Warnf("could not add marker tag to Dashboard of ConfigMap %v: %v\n", key, err)
			continue
		}

		desiredDashboards = append(desiredDashboards, desiredDashboard)
	}

	return desiredDashboards, nil
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

func keyOf(cm *v1.ConfigMap) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(cm)
}

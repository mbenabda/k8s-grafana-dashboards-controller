package controller

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"os"

	"reflect"
)

type DashboardsController struct {
	dashboards  grafana.DashboardsInterface
	clients     kubernetes.Interface
	configmaps  cache.SharedIndexInformer
	errorLogger *log.Logger
}

func New(dashboards grafana.DashboardsInterface, clients kubernetes.Interface, configmaps cache.SharedIndexInformer) *DashboardsController {
	return &DashboardsController{
		dashboards:  dashboards,
		clients:     clients,
		configmaps:  configmaps,
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func keyOf(cm *v1.ConfigMap) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(cm)
}

func asDashboard(cm *v1.ConfigMap) (grafana.Dashboard, error) {
	data := cm.Data
	keys := reflect.ValueOf(data).MapKeys()
	if len(keys) > 0 {
		firstKeyValue := data[keys[0].String()]
		return grafana.NewDashboard([]byte(firstKeyValue))
	}
	key, _ := keyOf(cm)
	return grafana.Dashboard{}, fmt.Errorf("could not get first Data key of ConfigMap %v", key)

}

func (c *DashboardsController) Run(ctx context.Context) {
	if !cache.WaitForCacheSync(ctx.Done(), c.configmaps.HasSynced) {
		return
	}
	c.configmaps.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cm := obj.(*v1.ConfigMap)

			key, _ := keyOf(cm)

			dashboard, err := asDashboard(cm)
			if err != nil {
				c.errorLogger.Printf("could not parse ConfigMap %v as a Dashboard : %v", key, err)
				return
			}

			if err := c.dashboards.Import(ctx, dashboard); err != nil {
				if err != nil {
					c.errorLogger.Printf("could not import the Dashboard from ConfigMap %v in grafana: %v", key, err)
					return
				}
			}

			log.Printf("The dashboard defined in ConfigMap %v has been added to Grafana", key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldObjCm := oldObj.(*v1.ConfigMap)

			oldObjKey, _ := keyOf(oldObjCm)

			oldObjDashboard, err := asDashboard(oldObjCm)
			if err != nil {
				c.errorLogger.Printf("could not parse ConfigMap %v as a Dashboard : %v", oldObjKey, err)
				return
			}
			oldDashboardSlug, err := oldObjDashboard.Slug()
			if err != nil {
				c.errorLogger.Printf("could not build slug of Dashboard defined in ConfigMap %v : %v", oldObjKey, err)
				return
			}

			cm := newObj.(*v1.ConfigMap)

			key, _ := keyOf(cm)

			dashboard, err := asDashboard(cm)
			if err != nil {
				c.errorLogger.Printf("could not parse ConfigMap %v as a Dashboard : %v", key, err)
				return
			}

			slug, err := dashboard.Slug()
			if err != nil {
				c.errorLogger.Printf("could not build slug of Dashboard defined in ConfigMap %v : %v", key, err)
				return
			}

			err = c.dashboards.Delete(ctx, oldDashboardSlug)
			if err != nil {
				c.errorLogger.Printf("could not delete Dashboard %v defined in ConfigMap %v : %v", oldDashboardSlug, oldObjKey, err)
				return
			}

			err = c.dashboards.Import(ctx, dashboard)
			if err != nil {
				c.errorLogger.Printf("could not import Dashboard defined in ConfigMap %v : %v", key, err)
				return
			}

			log.Printf("ConfigMap %s was modified. Dashboard %s updated accordingly", key, slug)
		},
		DeleteFunc: func(obj interface{}) {
			cm := obj.(*v1.ConfigMap)

			key, _ := keyOf(cm)

			dashboard, err := asDashboard(cm)
			if err != nil {
				c.errorLogger.Printf("could not parse ConfigMap %v as a Dashboard : %v", key, err)
				return
			}

			slug, err := dashboard.Slug()
			if err != nil {
				c.errorLogger.Printf("could not build slug of Dashboard defined in ConfigMap %v : %v", key, err)
				return
			}

			if err := c.dashboards.Delete(ctx, slug); err != nil {
				if err != nil {
					c.errorLogger.Printf("could not delet the Dashboard from ConfigMap %v in grafana: %v", key, err)
					return
				}
			}
			log.Printf("ConfigMap %s was deleted. Dashboard %s has been removed from Grafana", key, slug)
		},
	})

	<-ctx.Done()

}

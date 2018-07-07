package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"mbenabda.com/k8s-grafana-dashboards-controller/grafana"
)

type DashboardsController struct {
	dashboards    grafana.DashboardsInterface
	k8sConfig     *rest.Config
	namespace     string
	labelSelector labels.Selector
}

func New(dashboards grafana.DashboardsInterface, k8sConfig *rest.Config, namespace string, selector labels.Selector) *DashboardsController {
	return &DashboardsController{
		dashboards:    dashboards,
		k8sConfig:     k8sConfig,
		namespace:     namespace,
		labelSelector: selector,
	}
}

func (c *DashboardsController) Run(ctx context.Context) error {
	fmt.Println("we're running !")

	select {
	case <-ctx.Done():
		fmt.Println("controller done")
	}
	return nil
}

/*

    @Override
    public void eventReceived(Action action, ConfigMap configMap) {
        try {
            switch (action) {
                case ADDED: {
                    JsonNode dashboard = asDashboard(configMap);
                    grafana.importDashboard(dashboard);
                    LOGGER.info(format("ConfigMap %s was created. Dashboard %s has been added to Grafana", key(configMap), title(dashboard)));
                }
                break;

                case MODIFIED: {
                    JsonNode dashboard = asDashboard(configMap);
                    String title = title(dashboard);
                    String slug = grafana.slug(title);
                    grafana.deleteDashboard(slug);
                    grafana.importDashboard(dashboard);
                    LOGGER.info(format("ConfigMap %s was modified. Dashboard %s updated accordingly", key(configMap), title));
                }
                break;

                case DELETED: {
                    JsonNode dashboard = asDashboard(configMap);
                    String title = title(dashboard);
                    String slug = grafana.slug(title);
                    grafana.deleteDashboard(slug);
                    LOGGER.info(format("ConfigMap %s was deleted. Dashboard %s removed from Grafana", key(configMap), title));
                }
                break;
            }
        } catch (GrafanaException | IOException e) {
            LOGGER.warning(
                    format("Unable to handle %s event on ConfigMap %s because %s", action, key(configMap), e.getMessage())
            );
        }
    }

    private String key(ConfigMap configMap) {
        ObjectMeta metadata = configMap.getMetadata();

        return KEY_PARTS_JOINER.join(
                metadata.getNamespace(),
                metadata.getName()
        );
    }

    @Override
    public void onClose(KubernetesClientException e) {
        if (e != null) {
            LOGGER.log(Level.WARNING, e.getMessage(), e);
        }
    }

    private JsonNode asDashboard(ConfigMap configMap) throws IOException {
        return mapper.readTree(dashboardJson(configMap));
    }

    private String dashboardJson(ConfigMap configMap) {
        return valueOfFirstKey(configMap.getData());
    }

    private static <K, V> V valueOfFirstKey(Map<K, V> data) {
        return data.entrySet().iterator().next().getValue();
    }

    private static String title(JsonNode dashboard) {
        return dashboard.get("dashboard").get("title").asText();
    }
}

*/

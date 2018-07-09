Kubernetes controller that reconciles grafana configuration with dashboards defined as configmaps

- Access to the Kubernetes API is expected to be granted by the local kubeconfig file.
- You can use equality-based [labelSelector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#equality-based-requirement)s to select the configmaps to watch for dashboard descriptions.
- The controller requires access to the Grafana API, either using an `api key` or `basic auth`. Beware that if using HTTP, credentials will be transmited over the wire without any encryption.


Configuration
=======

Configuration can be specified in 2 ways:
- using command line arguments (use `--help` to see the complete usage instructions)
- using environment variables:

Env Variable | | Description | Default | Example
--- | --- | --- | --- | ---
`DRY_RUN` | `required` | prevents write operations against grafana when specified | `false` | ``
`CONFIGMAP_SELECTOR` | `optional` | kube-api compatible [labelSelector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) | `"" (everything)` | `"role=grafana-dashboard,app=awesome-app"`
`GRAFANA_URL` | `required` | Grafana's base URL | `""` | `https://grafana/`
`GRAFANA_API_KEY` | `required` unless using basic auth | Grafana API Key (get one at `<YOUR-GRAFANA-INSTANCE-URL>`/org/apikeys) | `""` | `"eyJrIjoiWlc4VjZaaFlZbWhwdzFiNVlHbXRn....."`
`GRAFANA_BASIC_AUTH_USERNAME` | `required` if using basic auth | Grafana username | `""` | `"mbenabda"`
`GRAFANA_BASIC_AUTH_PASSWORD` | if using basic auth | Grafana plain text password | `""` | `"1234"`
`MARKER_TAG` | `required` | value used to reconize dashboards managed by this instance | `""` | `"1234"`
`WATCH_NAMESPACE` | `optional` | namespae to watch for configmaps defining dashboards | `"" (all namespaces)` | `"1234"`

Run on Docker
======
```
docker build . -t k8s-grafana-dashboards-controller
docker run \
       -v $HOME/.kube/:/root/.kube:ro \
       -v /etc/ssl/certs:/etc/ssl/certs \
       k8s-grafana-dashboards-controller \
       --kubeconfig /root/.kube/config \
       --grafana-url https://grafana:3000 \
       --grafana-api-key s3cr3t \
       --selector role=grafana-dashboard \
       --marker-tag managed \
       --dry-run
```

Run on Kubernetes
========
Take a look at the [example deployment](./examples/k8s-deployment)


TODO
========
- add tests
- publish prometheus metrics
- update kubernetes deployment example
- setup CI
- setup [goreleaser](https://github.com/goreleaser/goreleaser)


Contributing
========
Contributions are welcome !

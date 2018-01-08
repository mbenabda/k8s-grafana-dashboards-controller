package com.mbenabda.kubernetes.controllers.grafana.dashboards;

import com.beust.jcommander.JCommander;
import com.beust.jcommander.ParameterException;
import com.mbenabda.grafana.client.GrafanaClientImpl;
import com.mbenabda.grafana.client.GrafanaConfiguration;
import com.mbenabda.kubernetes.controllers.grafana.dashboards.configuration.*;
import io.fabric8.kubernetes.client.Config;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;

import java.util.logging.Logger;

public class Main {
    private static final Logger LOGGER = Logger.getLogger(Main.class.getSimpleName());

    public static void main(String[] args) {
        try {
            K8sOptions k8sOptions = new K8sOptions();
            DashboardsWatchOptions configMapsFiler = new DashboardsWatchOptions();
            GrafanaOptions grafanaOptions = new GrafanaOptions();
            HelpParameter help = new HelpParameter();

            JCommander cli = JCommander.newBuilder()
                    .addObject(k8sOptions)
                    .addObject(configMapsFiler)
                    .addObject(grafanaOptions)
                    .addObject(help)
                    .allowParameterOverwriting(true)
                    .defaultProvider(new CompositeDefaultsProvider(
                            K8sOptions.defaultsProvider(),
                            DashboardsWatchOptions.defaultsProvider(),
                            GrafanaOptions.defaultsProvider()
                    ))
                    .build();

            cli.parse(args);

            if (help.requested()) {
                cli.usage();
                return;
            }

            Config k8sClientConfiguration = k8sOptions.asClientConfig();
            GrafanaConfiguration grafanaClientConfiguration = grafanaOptions.asClientConfig();

            new Controller(
                    new DefaultKubernetesClient(k8sClientConfiguration),
                    configMapsFiler,
                    new GrafanaClientImpl(grafanaClientConfiguration)
            ).run();

        } catch (ParameterException e) {
            LOGGER.warning(e.getMessage());
            System.exit(1);
        }
    }

}

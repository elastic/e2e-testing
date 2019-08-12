package services

// RunAPMServerService runs an APM Server service, connected to an elasticsearch and kibana services
func RunAPMServerService(
	version string, asDaemon bool, elasticsearchService Service, kibanaService Service) Service {

	esInspect, err := elasticsearchService.Inspect()
	if err != nil {
		return nil
	}

	esIP := esInspect.NetworkSettings.IPAddress

	kibanaInspect, err := kibanaService.Inspect()
	if err != nil {
		return nil
	}

	kibanaIP := kibanaInspect.NetworkSettings.IPAddress

	env := map[string]string{
		"apm-server.frontend.enabled":                      "true",
		"apm-server.frontend.rate_limit":                   "100000",
		"apm-server.host":                                  "0.0.0.0:8200",
		"apm-server.read_timeout":                          "1m",
		"apm-server.shutdown_timeout":                      "2m",
		"apm-server.write_timeout":                         "1m",
		"output.elasticsearch.enabled":                     "true",
		"setup.elasticsearch.host":                         "http://" + esIP + ":" + elasticsearchService.GetExposedPort(0),
		"setup.kibana.host":                                "http://" + kibanaIP + ":" + kibanaService.GetExposedPort(0),
		"setup.template.settings.index.number_of_replicas": "0",
		"xpack.monitoring.elasticsearch":                   "true",
	}

	serviceManager := NewServiceManager()

	apmServer := serviceManager.Build("apm-server", version, asDaemon)

	apmServer.SetEnv(env)

	return apmServer
}

module github.com/elastic/e2e-testing

go 1.14

replace github.com/containerd/containerd/pkg/cri/opts => github.com/containerd/containerd/pkg/cri/opts v1.6.1

replace github.com/containerd/containerd => github.com/containerd/containerd v1.5.13

require (
	github.com/Flaque/filet v0.0.0-20201012163910-45f684403088
	github.com/Jeffail/gabs/v2 v2.6.0
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/cucumber/godog v0.12.4
	github.com/docker/cli v20.10.21+incompatible
	github.com/docker/docker v20.10.21+incompatible
	github.com/elastic/elastic-package v0.77.0
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20210317102009-a9d74cec0186
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/gobuffalo/packr/v2 v2.8.3
	github.com/google/uuid v1.3.0
	github.com/joho/godotenv v1.4.0
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil/v3 v3.23.2
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.8.2
	github.com/testcontainers/testcontainers-go v0.13.0
	go.elastic.co/apm v1.13.0
	go.elastic.co/apm/module/apmelasticsearch v1.10.0
	go.elastic.co/apm/module/apmhttp v1.10.0
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/gotestsum v1.9.0
	howett.net/plist v0.0.0-20201203080718-1454fab16a06 // indirect
)

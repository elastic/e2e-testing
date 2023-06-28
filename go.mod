module github.com/elastic/e2e-testing

go 1.14

replace (
	github.com/cucumber/godog => github.com/laurazard/godog v0.0.0-20220922095256-4c4b17abdae7

	golang.org/x/oauth2 => golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783
)

require (
	github.com/Flaque/filet v0.0.0-20201012163910-45f684403088
	github.com/Jeffail/gabs/v2 v2.6.0
	github.com/cenkalti/backoff/v4 v4.2.0
	github.com/cucumber/godog v0.12.4
	github.com/docker/cli v23.0.5+incompatible
	github.com/docker/docker v23.0.5+incompatible
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
	github.com/spf13/cobra v1.7.0
	github.com/stretchr/testify v1.8.4
	github.com/testcontainers/testcontainers-go v0.21.0
	github.com/testcontainers/testcontainers-go/modules/compose v0.21.0
	go.elastic.co/apm v1.13.0
	go.elastic.co/apm/module/apmelasticsearch v1.10.0
	go.elastic.co/apm/module/apmhttp v1.10.0
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/gotestsum v1.10.0
	howett.net/plist v0.0.0-20201203080718-1454fab16a06 // indirect
)

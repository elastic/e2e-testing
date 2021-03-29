module github.com/elasticsearch/e2e-testing

go 1.14

require (
	github.com/Flaque/filet v0.0.0-20201012163910-45f684403088
	github.com/Jeffail/gabs/v2 v2.6.0
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/docker/docker v20.10.5+incompatible
	github.com/elastic/e2e-testing/cli v0.0.0-20210329121511-ae5b9fe35c35
	github.com/elastic/e2e-testing/e2e v0.0.0-20210329121511-ae5b9fe35c35
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20210317102009-a9d74cec0186
	github.com/gobuffalo/packr/v2 v2.8.1
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.10.0
	go.elastic.co/apm v1.11.0
	go.elastic.co/apm/module/apmelasticsearch v1.11.0
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/cucumber/godog => github.com/cucumber/godog v0.8.1

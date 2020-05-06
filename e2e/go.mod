module github.com/elastic/e2e-testing/e2e

go 1.12

require (
	github.com/Jeffail/gabs/v2 v2.5.0
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/cucumber/godog v0.9.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/elastic/e2e-testing/cli v1.0.0-rc1
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20190731061900-ea052088db25
	github.com/nsf/jsondiff v0.0.0-20190712045011-8443391ee9b6
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/sirupsen/logrus v1.4.2
)

replace github.com/elastic/e2e-testing/cli v1.0.0-rc1 => ../cli

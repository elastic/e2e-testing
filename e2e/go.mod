module github.com/elastic/e2e-testing/e2e

go 1.12

require (
	github.com/Jeffail/gabs/v2 v2.5.1
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/cucumber/godog v0.9.0
	github.com/elastic/e2e-testing/cli v1.0.0
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20190731061900-ea052088db25
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.6.1
)

replace github.com/elastic/e2e-testing/cli v1.0.0 => ../cli

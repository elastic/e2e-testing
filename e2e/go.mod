module github.com/elastic/e2e-testing/e2e

go 1.13

require (
	github.com/Jeffail/gabs/v2 v2.6.0
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/cucumber/godog v0.10.0
	github.com/elastic/e2e-testing/cli v0.0.0-20200717181709-15d2db53ded7
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20190731061900-ea052088db25
	github.com/sirupsen/logrus v1.4.2
)

replace github.com/elastic/e2e-testing/cli v0.0.0-20200717181709-15d2db53ded7 => ../cli

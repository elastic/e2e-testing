module github.com/elastic/e2e-testing/e2e

go 1.14

require (
	github.com/Jeffail/gabs/v2 v2.5.1
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/cucumber/godog v0.10.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/elastic/e2e-testing/cli v0.0.0-20200717181709-15d2db53ded7
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20190731061900-ea052088db25
	github.com/google/uuid v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
)

replace github.com/elastic/e2e-testing/cli v0.0.0-20200717181709-15d2db53ded7 => ../cli

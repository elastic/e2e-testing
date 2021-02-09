module github.com/elastic/e2e-testing/e2e

go 1.14

require (
	github.com/Jeffail/gabs/v2 v2.5.1
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/cucumber/godog v0.11.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/elastic/e2e-testing/cli v0.0.0-20200717181709-15d2db53ded7
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20190731061900-ea052088db25
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-memdb v1.3.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v1.1.1 // indirect
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.11.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace github.com/elastic/e2e-testing/cli v0.0.0-20200717181709-15d2db53ded7 => ../cli

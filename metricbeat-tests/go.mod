module github.com/elastic/metricbeat-tests-poc/metricbeat-tests

go 1.12

require (
	github.com/DATA-DOG/godog v0.7.13
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20190731061900-ea052088db25
	github.com/elastic/metricbeat-tests-poc/cli v0.1.0-rc4
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
)

replace github.com/testcontainers/testcontainers-go v0.0.8 => github.com/mdelapenya/testcontainers-go v0.0.9-compose

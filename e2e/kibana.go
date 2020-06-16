package e2e

import (
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

// WaitForKibana waits for kibana running in localhost:5601 to be healthy, returning false
// if kibana does not get healthy status in a defined number of minutes.
func WaitForKibana(maxTimeoutMinutes time.Duration) (bool, error) {
	return WaitForKibanaFromHostPort("localhost", 5601, maxTimeoutMinutes)
}

// WaitForKibanaFromHostPort waits for kibana running in a host:port to be healthy, returning false
// if kibana does not get healthy status in a defined number of minutes.
func WaitForKibanaFromHostPort(host string, port int, maxTimeoutMinutes time.Duration) (bool, error) {
	exp := getExponentialBackOff(maxTimeoutMinutes)

	retryCount := 1

	kibanaStatus := func() error {
		r := curl.HTTPRequest{
			BasicAuthUser:     "elastic",
			BasicAuthPassword: "changeme",
			URL:               fmt.Sprintf("http://%s:%d/status", host, port),
		}

		_, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error":          err,
				"retry":          retryCount,
				"statusEndpoint": r.URL,
				"elapsedTime":    exp.GetElapsedTime(),
			}).Warn("The Kibana instance is not healthy yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":        retryCount,
			"statusEndpoint": r.URL,
			"elapsedTime":    exp.GetElapsedTime(),
		}).Info("The Kibana instance is healthy")

		return nil
	}

	err := backoff.Retry(kibanaStatus, exp)
	if err != nil {
		return false, err
	}

	return true, nil
}

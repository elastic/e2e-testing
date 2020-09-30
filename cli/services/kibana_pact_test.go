package services

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
	"github.com/stretchr/testify/assert"
)

const packageNameRegex = `([a-zA-Z]+)`
const semverRegex = `(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`

var commonHeaders = dsl.MapMatcher{
	"Content-Type": term("application/json; charset=utf-8", `application\/json`),
}

var headersWithBasicAuth = dsl.MapMatcher{
	"Authorization": term("Basic ZWxhc3RpYzpjaGFuZ2VtZQ==", "Basic *"), // Base64('elastic:changeme') = 'ZWxhc3RpYzpjaGFuZ2VtZQ=='
	"Content-Type":  term("application/json; charset=utf-8", `application\/json`),
	"kbn-xsrf":      dsl.Like("e2e-tests"),
}

var client *KibanaClient

func TestMain(m *testing.M) {
	var exitCode int

	if os.Getenv("PACT_TEST") != "" {
		// Setup Pact and related test stuff
		setup()

		// Run all the tests
		exitCode = m.Run()

		// Shutdown the Mock Service and Write pact files to disk
		if err := pact.WritePact(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		pact.Teardown()
	} else {
		exitCode = m.Run()
	}

	os.Exit(exitCode)
}

func TestPactConsumer_GetIntegrations(t *testing.T) {
	type GetIntegrationsResponse struct {
		Response []struct {
			Title   string `json:"title"`
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"response"`
	}

	t.Run("there are integrations in the package registry", func(t *testing.T) {
		pact.
			AddInteraction().
			Given("There are integration in the package registry").
			UponReceiving("A request to get all integrations").
			WithRequest(request{
				Method:  "GET",
				Path:    dsl.Like(ingestManagerIntegrationsURL),
				Headers: headersWithBasicAuth,
			}).
			WillRespondWith(dsl.Response{
				Body:    dsl.Match(GetIntegrationsResponse{}),
				Status:  200,
				Headers: commonHeaders,
			})

		err := pact.Verify(func() error {
			body, err := client.GetIntegrations()
			if err != nil {
				return err
			}

			var r GetIntegrationsResponse
			err = json.Unmarshal([]byte(body), &r)
			if err != nil {
				return err
			}

			if len(r.Response) == 0 {
				return fmt.Errorf("Expected to retrieve integrations")
			}

			return nil
		})

		if err != nil {
			t.Fatalf("Error on Verify: %v", err)
		}
	})

	t.Run("the integration exists", func(t *testing.T) {
		packageName := "endpoint"
		version := "0.13.0"

		pact.
			AddInteraction().
			Given("Integration endpoint-0.13.0 exists").
			UponReceiving("A request to get the Elastic Endpoint integration on its 0.13.0 version").
			WithRequest(request{
				Method:  "GET",
				Path:    term(fmt.Sprintf(ingestManagerIntegrationURL, packageName, version), fmt.Sprintf(ingestManagerIntegrationURL, packageNameRegex, semverRegex)),
				Headers: headersWithBasicAuth,
			}).
			WillRespondWith(dsl.Response{
				Status:  200,
				Headers: commonHeaders,
			})

		err := pact.Verify(func() error {
			_, err := client.GetIntegration(packageName, version)
			if err != nil {
				return err
			}

			return err
		})

		assert.Nil(t, err)
	})

	t.Run("the integration does not exist", func(t *testing.T) {
		packageName := "foo"
		version := "1.0.0"
		expectedHTTPErrorCode := 500 // kibana returns a 500 error for a not found

		pact.
			AddInteraction().
			Given("Integration Foo-1.0.0 does not exist").
			UponReceiving("A request to get the Foo integration in its 1.0.0 version").
			WithRequest(request{
				Method:  "GET",
				Path:    term(fmt.Sprintf(ingestManagerIntegrationURL, packageName, version), fmt.Sprintf(ingestManagerIntegrationURL, packageNameRegex, semverRegex)),
				Headers: headersWithBasicAuth,
			}).
			WillRespondWith(dsl.Response{
				Status:  expectedHTTPErrorCode,
				Headers: commonHeaders,
			})

		err := pact.Verify(func() error {
			_, err := client.GetIntegration(packageName, version)
			if err != nil {
				return err
			}

			return err
		})

		assert.Equal(t, fmt.Errorf("GET request failed with %d", expectedHTTPErrorCode), err)
	})
}

// Common test data
var pact dsl.Pact

// Aliases
var term = dsl.Term

type request = dsl.Request

func setup() {
	pact = createPact()

	// Proactively start service to get access to the port
	pact.Setup(true)

	client = NewKibanaClient().withBaseURL(fmt.Sprintf("http://localhost:%d", pact.Server.Port))
}

func createPact() dsl.Pact {
	return dsl.Pact{
		Consumer:                 "E2E Testing framework",
		Provider:                 "Fleet",
		LogDir:                   os.Getenv("PACT_LOG_DIR"),
		PactDir:                  os.Getenv("PACT_DIR"),
		LogLevel:                 "INFO",
		DisableToolValidityCheck: true,
	}
}

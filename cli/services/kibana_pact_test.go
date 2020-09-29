package services

import (
	"fmt"
	"os"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

var commonHeaders = dsl.MapMatcher{
	"Content-Type":         term("application/json; charset=utf-8", `application\/json`),
	"X-Api-Correlation-Id": dsl.Like("100"),
}

var headersWithToken = dsl.MapMatcher{
	"Authorization": dsl.Like("Bearer 2019-01-01"),
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

func TestPact_GetIntegrations(t *testing.T) {
	t.Run("there are integrations in the package registry", func(t *testing.T) {
		pact.
			AddInteraction().
			Given("There are integration in the package registry").
			UponReceiving("A request to get all integrations").
			WithRequest(request{
				Method:  "GET",
				Path:    dsl.Like(ingestManagerIntegrationsURL),
				Headers: headersWithToken,
			}).
			WillRespondWith(dsl.Response{
				Body:    dsl.String(`{"response": []}`),
				Status:  200,
				Headers: commonHeaders,
			})

		err := pact.Verify(func() error {
			body, err := client.GetIntegrations()
			if body != `{"response": []}` {
				return fmt.Errorf("wanted an empty response, but was present")
			}

			return err
		})

		if err != nil {
			t.Fatalf("Error on Verify: %v", err)
		}
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
		LogDir:                   os.Getenv("LOG_DIR"),
		PactDir:                  os.Getenv("PACT_DIR"),
		LogLevel:                 "INFO",
		DisableToolValidityCheck: true,
	}
}

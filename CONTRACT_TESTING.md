# Contract tests for the E2E Testing project using PACT

> We are basically extracting contents from [Pact's docs](https://docs.pact.io)

## Introduction

### What is contract testing?
_**Contract testing** is a technique for testing an integration point by checking each application in isolation to ensure the messages it sends or receives conform to a shared understanding that is documented in a "contract"._

For applications that communicate via HTTP, these "messages" would be the HTTP request and response, and for an application that used queues, this would be the message that goes on the queue.

In practice, a common way of implementing contract tests (and the way Pact does it) is to check that all the calls to your test doubles [return the same results](https://martinfowler.com/bliki/ContractTest.html) as a call to the real application would.

### When would I use contract testing?
Contract testing is immediately applicable anywhere where you have two services that need to communicate - such as an API client and a web front-end. Although a single client and a single service is a common use case, contract testing really shines in an environment with many services (as is common for a microservice architecture). Having well-formed contract tests makes it easy for developers to avoid version hell. Contract testing is the killer app for microservice development and deployment.

### Contract testing terminology
In general, a contract is between a _consumer_ (for example, a client that wants to receive some data) and a _provider_ (for example, an API on a server that provides the data the client needs). In microservice architectures, the traditional terms _client_ and _server_ are not always appropriate -- for example, when communication is achieved through message queues. For this reason, we stick to _consumer_ and _provider_ in this documentation.

### Consumer Driven Contracts
Pact is a code-first [consumer-driven](http://martinfowler.com/articles/consumerDrivenContracts.html) contract testing tool, and is generally used by developers and testers who code. The contract is generated during the execution of the automated consumer tests. A major advantage of this pattern is that only parts of the communication that are actually used by the consumer(s) get tested. This in turn means that any provider behaviour not used by current consumers is free to change without breaking tests.

Unlike a schema or specification (eg. OAS), which is a static artefact that describes all possible states of a resource, a Pact contract is enforced by executing a collection of test cases, each of which describes a single concrete request/response pair - Pact is, in effect, "contract by example".

### Provider contract testing
The term "contract testing", or "provider contract testing", is sometimes used in other literature and documentation in the context of a standalone provider application (rather than in the context of an integration). When used in this context, "contract testing" means: a technique for ensuring a provider's actual behaviour conforms to its documented contract (for example, an Open API specification). This type of contract testing helps avoid integration failures by ensuring the provider code and documentation are in sync with each other. On its own, however, it does not provide any test based assurance that the consumers are calling the provider in the correct manner, or that the provider can meet all its consumers' expectations, and hence, it is not as effective in preventing integration bugs.

## Fleet contracts
We are already running E2E tests for Fleet, defining the scenarios and behaviours using Gherkin (see [here](./e2e/_suites/ingest-manager/features)). As explained above, we now want to verify that the contracts between Kibana Fleet and this project is not broken because of changes in the APIs the tests consume.

For that reason, we are going to use Pact to get notified whenever the contracts are not satisfied. The steps that we are folowing are:

1. Create unit tests mocking the HTTP requests/responses for Fleet's APIs. As we all know, this approach will always work, as we predefine the mocks, but on the other hand we won't get notified if/when the provider changes, so we must be reactively checking for updates.
1. Create a contract/pact for the consumer (the e2e tests), with the behaviours we expect from Fleet. The result will be a JSON file with the specs.
1. Store the contract in some place that is accesible by the provider. In this case, we are using the file system to store them, but Pact recommends using a [Broker](https://github.com/pact-foundation/pact_broker).
1. Verify the contract/pact from the provider side. The provider is able to check if a change breaks all its consumers by itself, as it has access to the contracts they provided (via file system or the afore mentioned Broker).

>Ideally, the provider project should be the one responsible of the verification but, for simplification, we are creating that verification in this project. Kibana should implement a way to verify contracts/pacts from its own build system, so any engineer is able to check if he/she breaks the consumers.

### Executing the tests
From root directory, use the following Make targets:

#### Generate the contracts ofr the consumer
```shell
$ make pact-consumer
```

It will execute Pact (using the [Go implementation](https://github.com/pact-foundation/pact-go/)) to generate the contract for the consumer (the E2E tests), generating a [JSON file](./pacts/e2e_testing_framework-fleet.json) representing the contract. This file will contain all the scenarios defined in the contract, highlighting the following sections:

- **consumer**: the name of the consumer
- **provider**: the name of the provider
- **interactions**: the scenarios executed in the contract, defining the format of the HTTP request, HTTP headers and HTTP response for each one. It will use regex to handle the response values, so that it's not coupled with the real response data returned by the provider.
- **meta**: Pact specification and version (2.0.0)


#### Verify the contracts from provider side
We want to verify that the provider satisfies the contracts. For that reason, we need the provider to be started first, and to achieve it, we are providing the following Make commands:

```shell
$ make prepare-pact-provider-deps # will run an Elasticsearch, Kibana andd Package Registry
$ make pact-provider              # will run the verification of the contracts by itself
$ make verify-provider            # will tear down the runtime dependencies
```

Each target depends on the one above it, so running `make verify-provider` will run the run the full life cycle.

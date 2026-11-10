---
status: draft
since: 2026-09-12
derived-from:
  - M-02-concept-of-operations
---

# ConOps Story: Running the Same Application in Three Clouds

An engineer is changing the rules for accepting customer events in a Go service that runs in AWS, Azure and Google Cloud.
She wants to check the rule on her laptop, then release it through the company's three existing deployment paths.
This story follows the reuse of application code through those steps, as described in [M-02 § Use the applications in production].

The setting is counterfactual Twilio Segment.
Segment's [Track specification] and [HTTP Tracking API] establish receiving customer events as a recognisable product concern.
The Go implementation, internal practices, multi-cloud arrangement and events below are imagined.

## The same setup in four places

The company receives customer events through a multi-tenant HTTP service running in serverless containers in all three clouds.
Customer commitments and the need to receive data near customers' chosen environments make those deployments part of its established operation.
The team already knows how to package, release and monitor the service in each environment.

The application accepts events such as `checkout.completed` and passes accepted events to the next part of the backend.
The proposed rule requires an order reference on those events when a validation option is enabled.
Otherwise valid events with an order reference should be accepted, and events missing it should be rejected, whichever cloud receives them.

The application code is shared, but its setup has accumulated in several places.
Each cloud entry point creates the application, supplies its dependencies and settings, and starts its HTTP server.
A local test program repeats part of that setup.
When an application setting changes, an engineer has to work through those paths to find which setup code also needs changing.
A check of the event-handling function alone does not show whether a running server received the setting.

## Put application setup with the application

The engineers put the code that creates the event handler, applies its settings and runs its server in a Go package with the application implementation.
The toolkit gives the hosts common ways to supply the application's configuration, construct an instance, request it to stop and wait for it to finish.
The application package implements the work and cleanup behind those operations.
The engineers adapt the local test host and the three cloud hosts to use those supported operations as they remove the duplicated setup.

Each cloud host still obtains the listening address, credentials and backing-service connections in the way its environment requires.
It supplies those values to the application and forwards the process's stop request to the application.
The application package defines what the validation option means and how it affects an event.
The hosts no longer need separate copies of that application setup.

Before changing the acceptance rule, the engineer runs the existing event checks through a local host that uses the package.
The host starts the actual HTTP server and supplies the dependencies needed for the test.
The engineer sends the existing accepted and rejected requests through that server and checks that they still produce the expected responses.
She stops the run and starts it again to check that it released its listener and can use the same address again.

## Check the new rule locally

The engineer adds the validation option to the application package and enables it in the local host's configuration.
She sends two otherwise valid `checkout.completed` events through the HTTP interface.
The event with an order reference is accepted; the event without one is rejected.
She repeats the checks with the option disabled and confirms the previous acceptance behaviour.

These checks use the construction and configuration code that the deployment hosts will call.
The engineer can follow that code in the debugger and see the setting reach the running application.
She can make another edit and repeat the requests without first building an image or waiting for a cloud deployment.
The team retains the checks in its Go test suite.

## Release through three hosts

After reviewing the application change, the engineers prepare the release builds for the three hosts with the same reviewed application package.
They enable the validation option in each deployment's configuration.
The cloud-specific work still includes packaging, supplying settings and dependencies, and choosing how to replace the running containers.
Each deployment follows the team's established release procedure.

The engineers check the running version and the supplied setting in each environment.
Through each cloud's public route, they repeat the accepted and rejected event requests used locally.
Those requests now pass through the cloud's routing and reach its running application.
The engineers compare the responses with the local results before accepting that deployment.
If one environment does not pass, its release can be held or rolled back while the team finds out why.
The three rollouts can finish at different times.

Replacement also exercises shutdown.
The application is configured to stop accepting new requests and give admitted requests time to finish before it exits.
The host passes the platform's stop request to that application code.
The platform sets the outer limit on how long the old container can remain alive.
The engineers check the old container's exit and watch for failed requests during replacement before continuing the rollout.
Those observations matter because the time allowed and the way traffic reaches a retiring container depend on the deployed arrangement.

## What the team keeps for the next change

The validation rule is now enabled in the three deployments.
The engineer checked its behaviour through one local host before the cloud releases, and each release supplied its own evidence about the running service.
The application package contains the rule, its configuration and the code for constructing and running the application.
The local tests and operational hosts reuse that package.

The team still maintains three cloud integrations and verifies them when the service changes.
The initial move into reusable Go code also took engineering work.
Whether that work pays off depends on the setup and maintenance it saves over later changes, and on the faults the retained checks reveal, as assessed in [M-01 § Mission and opportunity].

[HTTP Tracking API]: https://www.twilio.com/docs/segment/connections/sources/catalog/libraries/server/http-api
[M-01 § Mission and opportunity]: M-01-business-mission-analysis.md#mission-and-opportunity
[M-02 § Use the applications in production]: M-02-concept-of-operations.md#use-the-applications-in-production
[Track specification]: https://www.twilio.com/docs/segment/connections/spec/track

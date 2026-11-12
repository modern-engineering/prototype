---
status: draft
since: 2026-09-12
derived-from:
  - M-02-concept-of-operations
---

# ConOps Story: Running the Same Application in Three Clouds

An engineer wants a coding agent to help change the rules for accepting customer events in a Go service that runs in AWS, Azure and Google Cloud.
She needs a way for both of them to run the service and check the change before it reaches the company's three deployment paths.
This story follows their use of the toolkit's supplied host and testing support, then the release through custom cloud hosts, as described in [M-02 § Use the applications in production].

The setting is counterfactual Twilio Segment.
Segment's [Track specification] and [HTTP Tracking API] establish receiving customer events as a recognisable product concern.
The Go implementation, internal practices, multi-cloud arrangement and events below are imagined.

## The same setup in four places

The company receives customer events through a multi-tenant HTTP service running in serverless containers in all three clouds.
Customer commitments and the need to receive data near customers' chosen environments make those deployments part of its established operation.
The team already knows how to package, release and monitor the service in each environment.

The application accepts events such as `checkout.completed` and passes accepted events to the next part of the backend.
A customer uses those events to update order records and needs every checkout event to identify its order.
The proposed validation option requires an order reference on `checkout.completed` events when enabled.
Otherwise valid events with an order reference should be accepted and forwarded, and events missing it should be rejected, whichever cloud receives them.
When the option is disabled, otherwise valid events without an order reference must still be accepted and forwarded.

The application code is shared, but its setup has accumulated in several places.
Each cloud entry point creates the application, supplies its dependencies and settings, and starts its HTTP server.
A local test program repeats part of that setup.
Different maintainers have also kept their own code for handling process signals and waiting for shutdown.
When an application setting changes, an engineer has to work through those paths to find which setup code also needs changing.
A check of the event-handling function alone does not show whether a running server received the setting.

## Use the supplied host locally

The engineer separates application construction and execution from the cloud entry points.
She exposes the listen address, backend address and credentials as application inputs, with their meanings and defaults documented in the Go package.
The application reports failures and completion through the toolkit and responds to its cancellation request.
It keeps its existing HTTP handler and client libraries.

Making that change includes deciding how shutdown reaches the HTTP server and its client connections.
The application stops accepting new requests, gives admitted requests time to finish and releases the resources it owns before reporting completion.
The engineer implements that behaviour using the libraries' shutdown and cleanup operations.

The service can now run in the toolkit's supplied host.
The engineer writes a small Go program selecting the event service and a test receiver that also follows the toolkit's requirements for application inputs and execution.
The receiver records forwarded events and returns controlled replies in place of the production backend.
She supplies local addresses and test credentials, then invokes the supplied host to run both applications together on her laptop.
The host constructs them, coordinates failures and cancellation, and waits for them to finish.
She has no separate local host or process-signal loop to write.

Before changing the acceptance rule, she sends the existing accepted and rejected requests through the service's HTTP interface and checks the responses and recorded events.
She can pause the application code in her debugger and follow a request into the receiver.
When she interrupts the run, the supplied host requests cancellation and waits while both applications finish their work and cleanup.
She starts them again on the same addresses to check that the previous run released its listeners.

The engineer keeps this local program and uses the toolkit's testing support to run the same applications in Go tests.
The testing support coordinates cancellation and waits for completion at the end of each run.
The tests choose the requests and expected results.
This setup checks event acceptance and forwarding at the service's backend boundary using local test data.

## Give the agent a runnable starting point

The engineer gives the coding agent the customer's reason for the change and the expected behaviour with the option enabled and disabled.
The repository contains the application inputs, local program, existing tests and instructions for running them with the Go tools.
The agent runs the existing checks before editing the service.
For this setup, it needs neither cloud credentials nor a deployed environment.

The agent exposes the validation option as an application input and extends the HTTP tests.
With the option enabled, an otherwise valid event with an order reference must reach the receiver, and one without it must be rejected without being forwarded.
With the option disabled, those same events must both be accepted and forwarded.

The first implementation passes the handler's unit checks, but the HTTP test still accepts an event missing its order reference when validation is enabled.
The agent follows the setting from the application input into the handler and finds that the construction code still supplies the previous default.
It corrects that wiring and repeats the checks until the enabled and disabled cases pass.
Those checks exercise the construction and configuration code that the cloud hosts will use.

The engineer reviews the implementation and tests, including the failing result before the wiring correction and the passing result afterwards.
She can repeat the checks from the same checkout and use the local program to investigate anything the results leave unclear.
The agent has carried the change through implementation and local verification using the retained setup and supplied tooling.
The engineer did not have to reconstruct a cloud environment for each attempt.

## Reuse the execution support in three cloud hosts

The team adapts its existing cloud entry points to supply inputs to the same application package.
Each custom host obtains the listening address, credentials and backend connection settings in the way its environment requires.
It uses the execution packages behind the toolkit's supplied host to coordinate cancellation, failure and completion.
The cloud integration connects the platform's stop request to that support, while the application performs its shutdown and cleanup.
The team maintains the cloud-specific input handling and operating controls and reuses the toolkit's execution support across them.

After reviewing the change and the integrations, the engineers prepare the release builds for the three hosts with the same reviewed application package.
They enable the validation option in each deployment's configuration.
The cloud-specific work still includes packaging, supplying settings and dependencies, and choosing how to replace the running containers.
Each deployment follows the team's established release procedure.

The engineers check the running version and the supplied setting in each environment.
Through each cloud's public route, they repeat the accepted and rejected event requests used locally.
Those requests now pass through the cloud's routing and reach its running application.
The engineers also check the backend's records for those test events to confirm which were forwarded.
The engineers compare the responses with the local results before accepting that deployment.
If one environment does not pass, its release can be held or rolled back while the team finds out why.
The three rollouts can finish at different times.

Replacement also exercises shutdown.
The application uses the shutdown code exercised locally, with the time allowed for admitted requests set for that environment.
The platform sets the outer limit on how long the old container can remain alive.
The engineers check the old container's exit and watch for failed requests during replacement before continuing the rollout.
Those observations matter because the time allowed and the way traffic reaches a retiring container depend on the deployed arrangement.

## What the team keeps for the next change

The validation rule is now enabled in the three deployments.
The engineer and agent used the supplied host and testing support to check the change locally, and each release supplied its own evidence about the running service.
The application package contains the rule, its configuration and the code for constructing, running and stopping the application.
The local tests and cloud hosts reuse that package, and the custom hosts reuse the toolkit's execution support.

The next developer or coding agent can start with the retained local program, documented inputs and HTTP checks.
Because both applications expose their inputs and participate in cancellation and completion, the supplied host and testing support can run them.
The team continues to maintain the service, its shutdown behaviour, local program, test receiver and cloud integrations.

The initial adaptation also took engineering work.
Whether that work pays off depends on the setup and maintenance it saves over later changes, the faults the retained checks reveal and the human help agents need to complete their work, as assessed in [M-01 § Mission and opportunity].

[HTTP Tracking API]: https://www.twilio.com/docs/segment/connections/sources/catalog/libraries/server/http-api
[M-01 § Mission and opportunity]: M-01-business-mission-analysis.md#mission-and-opportunity
[M-02 § Use the applications in production]: M-02-concept-of-operations.md#use-the-applications-in-production
[Track specification]: https://www.twilio.com/docs/segment/connections/spec/track

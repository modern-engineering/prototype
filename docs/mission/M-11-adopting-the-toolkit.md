---
status: draft
since: 2026-11-10
derived-from:
  - M-02-concept-of-operations
---

# ConOps Story: Adopting the Toolkit

Before Ziv can debug the [GitHub Status gateway on her laptop][M-08-debugging-the-status-gateway], the team has to make its existing programs usable there.
This story imagines that earlier work, starting with one service and extending the migration as the team finds uses for it.

## A test environment on every push

The dashboard server, gateway, status backend and reporter already run in separate Docker containers.
Each container image packages one Go executable and what it needs to run.
The team's continuous integration (CI) pipeline builds these images.
A separate GitOps repository records which image versions and settings the deployment system should apply.

The team has also automated a way to try changes before releasing them.
Ziv pushes her branch, and a pipeline builds the images and deploys a private copy of the status system.
A few minutes later, she receives an address she can open in her browser.

Ziv wants to investigate how the gateway handles a request.
She needs the dashboard and gateway, with the gateway reading reports from staging.
The pipeline gives her copies of all four services, including a backend and reporter she does not need for this investigation.
Those unused copies consume compute capacity and add to the company's testing costs.
Nonetheless, pausing her gateway at a breakpoint or trying another backend address requires work outside the pipeline's normal workflow.
Every code edit sent through it also requires another build and deployment.

The team has a Docker Compose file for running the system locally too.
It describes the same set of containers and their connections, though it deviates from production manifests that reside in the GitOps source location.
To use it for Ziv's investigation, she would have to arrange a smaller selection and change the gateway's configuration to reach staging - not very convenient.
The file tells Docker which images to run and how to connect their containers.
It does not supply reusable Go code that a test can call to construct the applications with different inputs.
Keeping a test matrix of service combinations and settings in Compose files or overrides would also add configuration variations to maintain alongside the tests.

The automation has made a complete test environment easier to obtain, while leaving Ziv to work within the choices encoded in its deployment configuration.
She wants to choose which programs participate, supply their settings, pause their code and repeat an experiment from her editor.
The team tries the toolkit to bring that control into the Go code developers already work on.

## Start with the gateway

Ziv begins with the gateway because its request-handling code already lives in Go packages the team can reuse.
The remaining setup is in `func main`, the function Go calls when the executable starts.
It reads settings from environment variables, prepares the HTTP server and starts serving requests.
The executable also contains code that handles requests to stop the process and ends it when startup fails.

She separates the code that constructs and runs the gateway from the executable entry point so the toolkit can use it.
The gateway's application descriptor exposes the listen address, backend address, credentials and guest quota as inputs.
The application receives those values explicitly and keeps its existing HTTP handlers and client libraries.

Ziv uses the toolkit's supplied host to run the gateway with a local listen address, the staging backend address and her test credentials.
After the dashboard server is adapted in the same way, she writes a short Go program that selects both applications, supplies their inputs and invokes the toolkit's supplied host.
The two applications run in the same process on her laptop, where she can pause the gateway in her debugger.
Her browser can request a status report through them.
The status backend remains in staging.
Changing this local program changes what she runs and how it is configured; trying an edit does not require building a container image or allocating another complete environment.

The existing deployments still supply environment variables to each executable.
To preserve those deployment descriptors and their configuration, the team keeps a thin custom host at the original entry point as an adapter.
Its `main` translates the existing variables into the application descriptor's input fields, preserves the previous defaults and delegates execution to the toolkit.
This compatibility adapter is useful for the team's existing deployments; the local composition uses the supplied host directly.

## Hand execution to the toolkit

The gateway was written to run in a process of its own.
When it cannot open its network port, it calls `log.Fatal`, which writes an error message and immediately exits that process.
On Ziv's laptop, the gateway and dashboard now run in one process.
Exiting from inside the gateway would end the dashboard too, before either application had a chance to finish its work and release its resources.

Stopping the gateway needs a similar change.
Its existing shutdown code waits for an operating-system signal, such as the interrupt sent when someone presses Ctrl-C.
A Go test also needs to stop the gateway when its check finishes, while allowing the rest of the test process to continue.
The gateway must therefore be able to act on a cancellation request from the toolkit, without waiting for someone to interrupt the process.

Ziv changes the gateway to report its failure to the toolkit and to respond to the toolkit's cancellation request.
On cancellation, it stops accepting requests, finishes the work required for shutdown and releases the resources it owns.
It reports completion after that work finishes.
The application no longer decides when the process exits or installs its own process-signal handling.

The supplied host and the team's compatibility adapter both use the toolkit's cancellation and failure coordination.
For the local composition, a failure that prevents either server from running ends the run.
When Ziv deliberately occupies the gateway's port, the gateway reports the listen failure and the toolkit cancels the dashboard, waits for it to finish and reports why the run failed.
The adapter has no separate shutdown loop to maintain.

With the port available, both programs start normally.
Ziv interrupts the local run, and the toolkit requests cancellation and waits for both programs to finish.
She can start them again on the same addresses.
The compatibility adapter deals with the executable's inputs and final exit status; the toolkit coordinates execution, and the applications perform their own work and cleanup.

## Find the reporter's missing input

The first local setup is useful without migrating the rest of the backend.
Later, the team wants a repeatable check of how reports reach the gateway, so Ziv starts adapting the reporter and status backend for a Go test.
The reporter already accepts the addresses of the services it checks, but it still polls only once a minute.
That makes each attempt at the check unnecessarily slow.

The deployment configuration contains no polling interval.
A colleague reads the reporter's startup code and discovers that it looks for an environment variable named `STATUS_POLL_INTERVAL`, defaulting to one minute when the variable is absent.
The Go code used to construct the reporter offers no polling-interval input.
The test cannot choose a shorter interval through the settings it supplies there.

Setting that environment variable in the test would make this one run faster while leaving the input hidden inside the application.
Ziv instead exposes the polling interval alongside the other inputs used to construct the reporter.
The package's Go documentation describes what it needs and what the interval controls.
A caller can find those inputs without searching the implementation for environment lookups, and the composition shows which values a run uses.

The reporter's compatibility adapter continues to read `STATUS_POLL_INTERVAL` and apply its existing default before supplying the interval through the application descriptor.
The test supplies a shorter interval directly.
The applications themselves receive those values explicitly; they do not consult the environment for additional settings or overrides.

## Extend the migration when the work needs it

The team applies the same separation to the status backend.
The Go test uses the toolkit's testing support to start the reporter, backend and gateway together, give the reporter controlled service replies and read the result through the gateway.
This is the setup later used for [the partial-recovery test][M-09-testing-a-partial-recovery].
The dashboard remains useful in Ziv's local browser setup but is unnecessary for a test that reads the gateway directly.

For each migrated service, CI still builds the resulting executable into its own Docker container image.
The team deploys that image through the existing GitOps path and checks the service there.
The compatibility adapter supplies the established settings to the same application code used by the toolkit's host and the Go test.
Engineers check that the deployed service still accepts its normal traffic, produces the expected results and responds to shutdown in its operating environment.
The services continue to run in separate containers in production.

The migration changes how callers construct, configure and run the applications.
The toolkit requires explicit application inputs and participation in its cancellation and failure handling.
It leaves the service authors' internal library choices and communication protocols with them.
The reporter still asks services over HTTP, and the gateway still serves the interface the dashboard expects.

The team has paid for extracting the setup code, making the inputs visible and checking the migrated executables.
It also has local compositions to maintain alongside the applications.
Those costs count when the team evaluates whether the shorter feedback path is worth keeping, as described in [M-01 § Product posture and adoption].
For the next gateway change, Ziv can begin with the retained local program, choose the programs and settings the question needs, and investigate them under her debugger.
The pipeline remains available when she needs to check the deployed arrangement.

[M-08-debugging-the-status-gateway]: M-08-debugging-the-status-gateway.md
[M-09-testing-a-partial-recovery]: M-09-testing-a-partial-recovery.md
[M-01 § Product posture and adoption]: M-01-business-mission-analysis.md#product-posture-and-adoption

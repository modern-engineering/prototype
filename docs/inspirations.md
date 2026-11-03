# Inspirations

Here we credit the people and projects that inspired our Go toolkit for developing, testing and operating backends.
Some of these ideas also shaped our experiments in delivering software tailored to a customer.

## Describing a program others can run

In [Go’s analysis package], people who write a code check can publish it as a single named value.
It pairs the function that performs the check with its settings, help and a list of other checks it depends on.
That inspired the *descriptor* a Go package can publish for an application.
Another program can use that value to see what inputs the application accepts before starting it.

[Go’s flag package] lets each kind of setting define how it reads text.
A caller can supply `20s`, and a duration setting interprets it as twenty seconds.
We borrowed that idea so different tools could supply named application inputs as text.

## Running and testing with familiar Go tools

[`go test`] generates a small launcher when it builds a package’s tests.
We tried the same trick so a command could start a selection of applications.
We also liked how Go tests stay with the code.
Someone investigating a change can choose which applications to run, set them up in Go, and keep that setup in a test a colleague or coding agent can run again.

[`go doc`] makes it easy to look up a name and see its documentation.
We tried that style of lookup for applications and their inputs.
[`go run`] can run a command at a version named on the command line.
We would like that convenience when choosing a version of an application to try.

The [Go team’s work on reproducible toolchains] showed how to rebuild the same toolchain from its source on different machines.
We want a similar experience for assembling and running a Go backend.
Given the Go code that brings the applications together and the settings, services and data they need, an engineer should be able to run that group again in another environment.

[ko] compiles a Go program with `go build` and packages it as a runnable container image in one command.
We wanted the toolkit to offer the same convenience for the program a user selects.
The image would run it with its usual arguments and inputs.

## Reading a solution before running it

In our [work on tailored delivery], a solution design says which applications are meant to run for a customer, how they are configured and how they fit together.
[LLVM] gives compilers for different languages a common way to describe a program.
[GNU’s objdump] can look inside a file in the ELF executable format without starting the program.
Together these ideas suggested that one tool could write down a design and another could read it before running any of its applications.

[Hugo] uses Go modules to track the website themes a site names in its configuration.
We tried that idea with Go packages named in a solution design.

Go offers [a package for loading code] and [another for checking types].
They showed us how to make inspection available to people writing their own tools.
To run the design, a tool also has to connect each name to the right piece of code.
[Go’s gob package] gave us an example of that lookup by pairing names with Go types when saving and restoring data.

[Docker image layers] suggested a possibility for delivery.
Could each stage add its results to what it received while keeping the original design intact?

## Before and after an application goes live

[Kubernetes] can accept a request to run the first application on a cluster.
[PostgreSQL] can accept the first table in a new database.
On a smaller scale, Go’s [`strings.Builder`] needs no setup before it can collect its first word.
These examples helped us imagine a platform that is useful even before a customer’s applications are running.
It can already accept a request to prepare a database they will need.

In [Team Topologies], Matthew Skelton and Manuel Pais give as much attention to how teams work together as to what each team does.
We borrowed that way of looking at a handoff.
The people writing an application decide what it does and what inputs it needs.
The people putting applications together supply what each needs for a run.
One person can do both.

The [INCOSE Systems Engineering Handbook] helped us think beyond the day a service goes live to keeping it useful and eventually retiring it.
An [INCOSE symposium talk] distinguished how an organisation works from how people use a system it builds.
We drew on that distinction when describing the maintainers’ work in the [project concept of operations] and how engineers use the toolkit in the [Operational Concept].
[NASA’s guide to a concept of operations] helped us tell the stories about using the toolkit from the point of view of the people doing the work.

[Go’s analysis package]: https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Analyzer
[Go’s flag package]: https://pkg.go.dev/flag#Value
[`go test`]: https://go.dev/src/cmd/go/internal/load/test.go#L308
[`go doc`]: https://pkg.go.dev/cmd/go#hdr-Show_documentation_for_package_or_symbol
[`go run`]: https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program
[Go team’s work on reproducible toolchains]: https://go.dev/blog/rebuild#making-a-build-reproducible
[ko]: https://ko.build/
[work on tailored delivery]: mission/M-08-solution-management.md
[LLVM]: https://llvm.org/docs/LangRef.html#introduction
[GNU’s objdump]: https://sourceware.org/binutils/docs/binutils/objdump.html
[Hugo]: https://gohugo.io/hugo-modules/use-modules/
[a package for loading code]: https://pkg.go.dev/golang.org/x/tools/go/packages
[another for checking types]: https://pkg.go.dev/go/types
[Go’s gob package]: https://pkg.go.dev/encoding/gob#Register
[Docker image layers]: https://docs.docker.com/get-started/docker-concepts/building-images/understanding-image-layers/
[Kubernetes]: https://kubernetes.io/docs/concepts/overview/components/
[PostgreSQL]: https://www.postgresql.org/docs/current/tutorial-table.html
[`strings.Builder`]: https://pkg.go.dev/strings#Builder
[Team Topologies]: https://teamtopologies.com/key-concepts
[INCOSE Systems Engineering Handbook]: https://www.incose.org/resources-publications/technical-publications/se-handbook/
[NASA’s guide to a concept of operations]: https://www.nasa.gov/reference/appendix-s-concept-of-operations-annotated-outline/
[INCOSE symposium talk]: https://www.incose.org/wp-content/uploads/2026/01/11.5.3.pdf#page=8
[project concept of operations]: mission/M-01-business-mission-analysis.md#project-concept-of-operations
[Operational Concept]: mission/M-02-operational-concept.md

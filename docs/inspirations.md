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

[Go’s analysis package]: https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Analyzer
[Go’s flag package]: https://pkg.go.dev/flag#Value
[`go test`]: https://go.dev/src/cmd/go/internal/load/test.go#L308
[`go doc`]: https://pkg.go.dev/cmd/go#hdr-Show_documentation_for_package_or_symbol
[`go run`]: https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program
[Go team’s work on reproducible toolchains]: https://go.dev/blog/rebuild#making-a-build-reproducible
[ko]: https://ko.build/

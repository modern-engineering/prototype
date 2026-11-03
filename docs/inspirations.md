# Inspirations

Here we credit the people and projects that inspired our Go toolkit for developing, testing and operating backends.
Some of these ideas also shaped our experiments in delivering software tailored to a customer.

## Describing a program others can run

In [Go’s analysis package], people who write a code check can publish it as a single named value.
It pairs the function that performs the check with its settings, help and a list of other checks it depends on.
That inspired the *descriptor* a Go package can publish for an application.
Another program can use that value to see what inputs the application accepts before starting it.

[Go’s flag package] connects command-line options to ordinary variables, such as the time a reporter waits between checks.
We liked that the code that does the reporting can just use the interval.
A test can supply a short interval directly, while a program that starts the reporter in production can read the setting from the environment and pass it in.

## Running and testing with familiar Go tools

[`go test`] generates a small launcher when it builds a package’s tests.
We tried the same trick so a command could start a selection of applications.
We also liked how Go tests stay with the code.
Someone investigating a change can choose which applications to run, set them up in Go, and keep that setup in a test a colleague or coding agent can run again.

[`go doc`] makes it easy to look up a name and see its documentation.
We tried that style of lookup for applications and their inputs.
[`go run`] can run a command at a version named on the command line.
We would like that convenience when choosing a version of an application to try.

The [Go team’s work on reproducible toolchains] showed how a build can depend on explicit inputs instead of quirks of the machine that ran it.
We want a similar experience when revisiting a system test.
An engineer can check out the Go code that sets up the run and supply the settings, services and data it needs.

[ko] uses `go build` and packages the resulting program as a container image.
When we tried this with a generated launcher, the same command-line options worked inside the image.
We liked how simple the step from building to packaging could be.

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

[Go’s analysis package]: https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Analyzer
[Go’s flag package]: https://pkg.go.dev/flag
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

# Inspirations

Here we credit the people and projects whose work influenced the toolkit, from ideas we borrowed to ones we are still exploring.

## Describing a program others can run

In [Go’s analysis package], the people who write a code check can publish it as a single named value.
Alongside the function that performs the check are its settings, help for the person using it, and a list of other checks the tool needs to run first.
We borrowed this idea for a Go package that supplies a reusable application or a job such as setting up a database.
It can publish a *descriptor*, a value another program can use directly to see what the application or job accepts and to run it.

[Go’s flag package] lets a command-line option set an ordinary variable, such as a number of retries.
We liked how the rest of the program can use that number without knowing where it came from, and used the same approach for toolkit inputs.

## What one command can do

With [`go test`], someone can run the tests in a Go package without writing a program to launch them.
The command generates that little program when it builds the tests.
We borrowed the trick so the toolkit can run applications and jobs together without anyone having to write their shared starting point.
[`go doc`] and [`go run`] suggest a similar convenience for individual descriptors.
We would like to look up a descriptor by name and run a particular version of it just as easily.

[Hugo] uses Go modules to share website themes.
We liked how Hugo can keep track of the version a site needs even when none of its Go code mentions the theme.
Our toolkit does much the same for Go packages named in a solution design.
[ko] takes an ordinary Go build one step further and packages the result as a container image, ready to distribute.
That is an experience we would like for the programs the toolkit builds.

## Reading a solution before running it

[LLVM] gives compilers for different languages a common way to describe a program.
The same description can pass between several tools.
[GNU’s objdump] can look inside an executable file, such as one in the ELF format, without starting the program.
We liked both ideas for a solution design.
One tool can write down which applications belong together, and someone using another tool can see what the design contains before running any of them.

Go offers [a package for loading code] and [another for checking types].
They showed us how to make the work of inspection available to people writing their own tools.
Later, when a tool needs to run what the design names, it has to connect each name to the right piece of code.
[Go’s gob package], which pairs names with Go types when saving and restoring data, gave us an example of how to make that connection.

[Docker image layers] suggested a question we are still exploring.
Could each stage of delivery add its results to what it received while keeping the original design intact?

[Go’s analysis package]: https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Analyzer
[Go’s flag package]: https://pkg.go.dev/flag
[`go test`]: https://go.dev/src/cmd/go/internal/load/test.go#L308
[`go doc`]: https://pkg.go.dev/cmd/go#hdr-Show_documentation_for_package_or_symbol
[`go run`]: https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program
[Hugo]: https://gohugo.io/hugo-modules/use-modules/
[ko]: https://ko.build/
[LLVM]: https://llvm.org/docs/LangRef.html#introduction
[GNU’s objdump]: https://sourceware.org/binutils/docs/binutils/objdump.html
[a package for loading code]: https://pkg.go.dev/golang.org/x/tools/go/packages
[another for checking types]: https://pkg.go.dev/go/types
[Go’s gob package]: https://pkg.go.dev/encoding/gob#Register
[Docker image layers]: https://docs.docker.com/get-started/docker-concepts/building-images/understanding-image-layers/

# Inspirations

Here we credit the people and projects whose work influenced the toolkit, from ideas we borrowed to ones we are still exploring.

## Describing a program others can run

In [Go’s analysis package], the people who write a code check can publish it as a single named value.
Alongside the function that performs the check are its settings, help for the person using it, and a list of other checks the tool needs to run first.
We borrowed this idea for a Go package that supplies a reusable application or a job such as setting up a database.
It can publish a *descriptor*, a value another program can use directly to see what the application or job accepts and to run it.

[Go’s flag package] lets a command-line option set an ordinary variable, such as a number of retries.
We liked how the rest of the program can use that number without knowing where it came from, and used the same approach for toolkit inputs.

[Go’s analysis package]: https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Analyzer
[Go’s flag package]: https://pkg.go.dev/flag

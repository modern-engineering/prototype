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

[Go’s analysis package]: https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Analyzer
[Go’s flag package]: https://pkg.go.dev/flag

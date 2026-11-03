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

[Go’s analysis package]: https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Analyzer
[Go’s flag package]: https://pkg.go.dev/flag#Value

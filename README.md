# prototype

[![CI](https://github.com/modern-engineering/prototype/actions/workflows/ci.yml/badge.svg)](https://github.com/modern-engineering/prototype/actions/workflows/ci.yml)

Prototyping the end-to-end experience of developing, testing, operating, and
verifying backend applications.

## Examples

The commands in [`examples/`](examples/) demonstrate the toolkit's binding, linking, and loading primitives.
The [Acme example](examples/acme/) is a separate Go module showing an adopting organization's HTTP package, descriptors, and command.

Run both modules' tests from the repository root:

```sh
go test -race ./...
go -C examples/acme test -race ./...
```

## Documentation

Architectural decisions and design documents live in [`docs/`](docs/).

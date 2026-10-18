# Acme integration example

This module shows how an adopting organization can maintain shared functionality and descriptors separately from its executable.

- `httpclient` defines authenticated HTTP configuration and request behavior.
- `catalogue` exports the `Fetch` job descriptor.
- `cmd/fetch` binds configuration, supplies its HTTP client, and owns the transport lifetime.

The command derives flags from the descriptor's inputs through `command.Parse`.
Defaults and help text stay with the catalogue, and `command.WriteOutputs` emits all declared results as a JSON object of strings.
Your executable remains responsible for resource setup, cancellation, execution, and error reporting.

The module depends on the toolkit through a local `replace` directive so the example runs directly from this checkout.
An independently distributed module would require a published toolkit version instead.

From this directory:

```sh
go test -race ./...
go vet ./...
go run ./cmd/fetch -url http://localhost:8080 -token example-token
```

`Fetch.Make` allocates fresh state without opening connections.
The command's client interface is a local consumer contract satisfied by the runner's promoted `UseHTTPClient` method.
Configuration binding and runtime resource ownership remain separate.

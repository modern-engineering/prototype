# Acme integration example

This module shows how an adopting organization can maintain shared functionality and descriptors separately from its executable.

- `httpclient` defines authenticated HTTP configuration and request behavior.
- `catalogue` exports Acme's job and application descriptors.
- `cmd/fetch` binds configuration, supplies its HTTP client, and owns the transport lifetime.
- `cmd/playground` compiles the organization's catalogue and runs an embedded solution artifact.

The `fetch` command derives flags from the descriptor's inputs through `command.Parse`.
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

## Run a solution artifact

```sh
go run ./cmd/playground
```

The embedded `cmd/playground/testdata/service.sef` supplies occurrence identities and bindings.
An environment job feeds a database-address job, whose result configures the API and reporting applications.
The command registers the compiled catalogue, instantiates the artifact's programs, and supplies a logger through a small interface it consumes.
All jobs finish before either application starts, and Interrupt or SIGTERM cancels both applications and waits for their cleanup.
The jobs derive example configuration; they do not provision cloud resources.

The two commands show different loading styles.
`fetch` holds one named descriptor and wires its tuple directly.
`playground` resolves identities and bindings from serialized input and uses `solution/load` to retain their association with constructed state.

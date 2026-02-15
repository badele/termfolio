# Contributing

Thanks for contributing to termfolio.

## Prerequisites

- Go 1.24+

## Local setup

```bash
git clone https://github.com/badele/termfolio.git
cd termfolio
go build ./cmd/termfolio
```

Run the binary with a config file:

```bash
./termfolio --config docs/termfolio.yaml
```

## Tests

```bash
go test ./...
```

## Style

- Use `gofmt` for all Go code.
- Keep changes focused and scoped to one purpose.

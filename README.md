# Go-solc

Solidity bindings for Golang.

## Usage

Declare a project:

```
p, err := gosolc.NewProject(gosolc.WithContractsDir("."))
if err != nil {
    // handle err
}
p.Compile()
```

## CLI

```
$ go run cmd/gosolc/main.go [--contracts . --artifacts .]
```

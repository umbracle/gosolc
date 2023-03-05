package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/umbracle/gosolc"
)

var contractsDir string
var artifactsDir string

func main() {
	flag.StringVar(&contractsDir, "contracts", "", "")
	flag.StringVar(&artifactsDir, "artifacts", "", "")
	flag.Parse()

	opts := []gosolc.Option{
		gosolc.WithContractsDir(contractsDir),
		gosolc.WithArtifactsDir(artifactsDir),
	}
	p, err := gosolc.NewProject(opts...)
	if err != nil {
		fmt.Printf("[ERROR]: Failed to start project: %v", err)
		os.Exit(1)
	}

	res, err := p.Compile()
	if err != nil {
		fmt.Printf("[ERROR]: Failed to compile: %v", err)
		os.Exit(1)
	}

	fmt.Printf("[RESULT]: Compiled contracts: %s", strings.Join(res.Contracts, ","))
}

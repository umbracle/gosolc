package gosolc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
)

type Optimizer struct {
	Enabled bool
	Runs    int
}

type solcInput struct {
	files  []string
	config *Config
}

type Artifact struct {
	Abi json.RawMessage `json:"abi"`

	EVM struct {
		Bytecode          *Bytecode         `json:"bytecode"`
		DeployedBytecode  *Bytecode         `json:"deployedBytecode"`
		Opcodes           string            `json:"opcodes"`
		SourceMap         string            `json:"sourceMap"`
		MethodIdentifiers map[string]string `json:"methodIdentifiers"`
	} `json:"evm"`

	Metadata string `json:"metadata"`
}

type solcOutput struct {
	Errors    []*solcError
	Contracts map[string]map[string]*Artifact
	Sources   map[string]*solcSourceFile
	Version   string
}

type solcError struct {
	FormattedMessage string `json:"formattedMessage"`
}

type solcSourceFile struct {
	AST json.RawMessage
}

var inputDesc = `{
	"language": "Solidity",
	"sources": {
		{{range $index, $element := .Files}}
		{{if $index}},{{end}}
		"{{ $element }}": {
			"urls": [
				"{{ $element }}"
			]
		}
		{{end}}
	},
	"settings": {
		"optimizer": {
			"runs": 200
		},
		"outputSelection": {
			"*": {
				"": [
					"ast"
				],
				"*": [
					"abi",
					"evm.bytecode",
					"evm.deployedBytecode",
					"evm.methodIdentifiers",
					"metadata"
				]
			}
		}
	}
}`

func Compile(path string, input *solcInput) (*solcOutput, error) {
	tmplInput := map[string]interface{}{
		"Files": input.files,
	}
	tmpl, err := template.New("input").Parse(inputDesc)
	if err != nil {
		return nil, err
	}

	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, tmplInput); err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(input.config.ContractsDir)
	if err != nil {
		return nil, err
	}

	args := []string{
		"--standard-json",
		"--base-path", absPath,
		"--allow-paths", absPath,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(path, args...)

	cmd.Stdin = bytes.NewBuffer(tpl.Bytes())
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile: %s", stderr.String())
	}

	var output *solcOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, err
	}

	var outputErr error
	if len(output.Errors) != 0 {
		for _, err := range output.Errors {
			outputErr = multierror.Append(outputErr, fmt.Errorf(err.FormattedMessage))
		}
		return nil, outputErr
	}

	return output, nil
}

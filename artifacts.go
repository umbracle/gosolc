package gosolc

import (
	"encoding/json"
	"path/filepath"
	"time"
)

type solcConfigSettings struct {
	Optimizer struct {
		Enabled bool   `json:"enabled"`
		Runs    uint64 `json:"runs"`
	} `json:"optimizer"`

	Metadata struct {
		BytecodeHash string `json:"bytecodeHash"`
		AppendCBOR   bool   `json:"appendCBOR"`
	} `json:"metadata"`

	OutputSelection map[string]interface{} `json:"outputSelection"`
}

// contractArtifacts is the output file generated for each contract
type contractArtifact struct {
	ABI               json.RawMessage   `json:"abi"`
	Bytecode          *Bytecode         `json:"bytecode"`
	DeployedBytecode  *Bytecode         `json:"deployedBytecode"`
	MethodIdentifiers map[string]string `json:"methodIdentifiers"`
	RawMetadata       string            `json:"rawMetadata"`
	Metadata          json.RawMessage   `json:"metadata"`
	AST               json.RawMessage   `json:"ast"`
}

type Source struct {
	// Dir is the directory of the file
	Dir string

	// Filename is the name of the file
	Filename string

	// ModTime is the modified time of the source
	ModTime time.Time

	// Versions are the required version for this source
	Version []string

	// Imports is the list of imports defined in this source
	Imports []string

	AST json.RawMessage
}

// relPath returns the relative path of the source inside the contracts directory
func (s *Source) relPath() string {
	return filepath.Join(s.Dir, s.Filename)
}

type Contract struct {
	// Name is the name of the contract
	Name string

	Source string

	// Abi is the abi encoding of the contract
	Abi json.RawMessage

	Bytecode *Bytecode

	DeployedBytecode *Bytecode

	Metadata string

	MethodIdentifiers map[string]string
}

type Bytecode struct {
	Object         string          `json:"object"`
	SrcMap         string          `json:"sourceMap"`
	LinkReferences json.RawMessage `json:"linkReferences"`
}

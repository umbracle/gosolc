package gosolc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	version "github.com/hashicorp/go-version"
	"github.com/umbracle/gosolc/dag"
)

func (p *Project) findLocalDiff() ([]*FileDiff, error) {
	files, err := readDir(p.config.ContractsDir)
	if err != nil {
		return nil, err
	}

	sources, err := p.ListSources()
	if err != nil {
		return nil, err
	}
	diffFiles2, err := calcDiff(sources, p.config.ContractsDir, files)
	if err != nil {
		return nil, err
	}

	// parse the files and update the sources
	for _, diff := range diffFiles2 {
		file, err := os.Stat(filepath.Join(p.config.ContractsDir, diff.Path))
		if err != nil {
			return nil, err
		}

		source, err := parseSource(string(diff.Content), diff.Path)
		if err != nil {
			return nil, err
		}

		source.ModTime = file.ModTime()

		if err := p.UpsertSource(source); err != nil {
			return nil, err
		}
	}

	return diffFiles2, nil
}

type fileWriter struct {
	absPath string
}

func (f *fileWriter) Write(path string, content interface{}) error {
	var data []byte

	if strings.HasSuffix(path, ".json") {
		jsonData, err := json.MarshalIndent(content, "", "    ")
		if err != nil {
			return err
		}
		data = jsonData
	} else {
		return fmt.Errorf("marshaling not found")
	}

	fullPath := filepath.Join(f.absPath, path)

	// create the parent directory if it does not exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(fullPath, data, 0644); err != nil {
		return err
	}
	return nil
}

// Compile compiles the application
func (p *Project) Compile() (*CompilationResult, error) {
	diffFiles, err := p.findLocalDiff()
	if err != nil {
		return nil, err
	}

	fileW := &fileWriter{
		absPath: p.config.ArtifactsDir,
	}

	diffSources := []string{}
	for _, diffFile := range diffFiles {
		diffSources = append(diffSources, diffFile.Path)
	}
	result, err := p.compileImpl(diffSources)
	if err != nil {
		return nil, err
	}

	// write artifacts!
	for _, name := range result.Contracts {
		// name has the format <path>:<contract>
		// remove the contract name
		spl := strings.Split(name, ":")

		// resolve the ast from the source file
		source := p.getSourceByPath(spl[0])
		if source == nil {
			panic("bug")
		}

		contract := p.findContractByFullName(name)
		if contract == nil {
			panic("bug 2")
		}

		artifact := &contractArtifact{
			ABI:               contract.Abi,
			Bytecode:          contract.Bytecode,
			DeployedBytecode:  contract.DeployedBytecode,
			RawMetadata:       contract.Metadata,
			Metadata:          json.RawMessage(contract.Metadata),
			MethodIdentifiers: contract.MethodIdentifiers,
			AST:               source.AST,
		}

		if err := fileW.Write(filepath.Join("out", strings.Replace(name, ":", "/", -1))+".json", artifact); err != nil {
			return nil, err
		}
	}

	return result, nil
}

type CompilationResult struct {
	// Contracts is the list of contracts compiled
	Contracts []string

	// Runs is the list of independent compilation components
	Runs []*CompilationRun
}

type CompilationRun struct {
	// Paths of the Solidity contracts for this run
	Components []string

	// ExecutionTime is the time it took this component to compile
	ExecutionTime time.Duration
}

func (p *Project) compileImpl(updatedFiles []string) (*CompilationResult, error) {
	solidityVersion, err := version.NewVersion(p.config.SolidityVersion)
	if err != nil {
		return nil, err
	}

	updatedSources := []*Source{}

	sources, err := p.ListSources()
	if err != nil {
		return nil, err
	}
	sourcesMap := map[string]*Source{}
	for _, s := range sources {
		sourcesMap[s.relPath()] = s

		for _, i := range updatedFiles {
			if i == s.relPath() {
				updatedSources = append(updatedSources, s)
			}
		}
	}

	// detect dependencies only for new and modified files
	diffSources := updatedSources

	// build dag map
	dd := &dag.Dag{}
	for _, f := range sourcesMap {
		dd.AddVertex(f)
	}
	// add edges
	for _, src := range sourcesMap {
		for _, dst := range src.Imports {
			dst, ok := sourcesMap[dst]
			if !ok {
				panic(fmt.Errorf("BUG: elem in DAG not found: %s", dst.relPath()))
			}
			dd.AddEdge(dag.Edge{
				Src: src,
				Dst: dst,
			})
		}
	}

	// Create an independent component set for each end node of the graph.
	// Include the node + all their parent nodes. Only recompute the sets in
	// which at least one node has been modified.
	rawComponents := dd.FindComponents()

	components := [][]string{}
	for _, comp := range rawComponents {
		found := false
		for _, i := range comp {
			for _, j := range diffSources {
				if i == j {
					found = true
				}
			}
		}
		if found {
			subComp := []string{}
			for _, i := range comp {
				subComp = append(subComp, i.(*Source).relPath())
			}
			components = append(components, subComp)
		}
	}

	resp := &CompilationResult{
		Contracts: []string{},
		Runs:      []*CompilationRun{},
	}

	// generate the outputs and compile
	for _, comp := range components {
		pragmas := []string{}
		for _, i := range comp {
			pragmas = append(pragmas, strings.Split(sourcesMap[i].Version[0], " ")...)
		}
		pragmas = unique(pragmas)

		versionConstraint, err := version.NewConstraint(strings.Join(pragmas, ", "))
		if err != nil {
			return nil, err
		}
		if !versionConstraint.Check(solidityVersion) {
			panic("not match in solidity compiler")
		}

		input := &solcInput{
			files:  comp,
			config: p.config,
		}

		path, err := p.svm.Resolve(solidityVersion.String())
		if err != nil {
			return nil, err
		}

		now := time.Now()

		output, err := Compile(path, input)
		if err != nil {
			return nil, err
		}

		resp.Runs = append(resp.Runs, &CompilationRun{
			Components:    comp,
			ExecutionTime: time.Since(now),
		})

		for sourceName, sourceContracts := range output.Contracts {
			for contractName, contract := range sourceContracts {
				ctnr := &Contract{
					Name:              contractName,
					Source:            sourceName,
					Abi:               contract.Abi,
					Bytecode:          contract.EVM.Bytecode,
					DeployedBytecode:  contract.EVM.DeployedBytecode,
					Metadata:          contract.Metadata,
					MethodIdentifiers: contract.EVM.MethodIdentifiers,
				}
				if err := p.UpsertContract(ctnr); err != nil {
					return nil, err
				}
				resp.Contracts = append(resp.Contracts, sourceName+":"+contractName)
			}
		}

		for sourceName, source := range output.Sources {
			src := p.getSourceByPath(sourceName)
			if src == nil {
				panic("BUG")
			}
			src.AST = source.AST
		}
	}

	return resp, nil
}

var (
	importRegexp = regexp.MustCompile(`import (".*"|'.*')`)
)

func parseDependencies(contract string) []string {
	res := importRegexp.FindAllStringSubmatch(contract, -1)
	if len(res) == 0 {
		return []string{}
	}

	clean := []string{}
	for _, j := range res {
		i := j[1]
		i = strings.Trim(i, "'")
		i = strings.Trim(i, "\"")
		clean = append(clean, i)
	}
	return clean
}

var (
	pragmaRegexp = regexp.MustCompile(`pragma\s+solidity\s+(.*);`)
)

func parsePragma(contract string) ([]string, error) {
	res := pragmaRegexp.FindStringSubmatch(contract)
	if len(res) == 0 {
		return nil, fmt.Errorf("pragma not found")
	}
	return res[1:], nil
}

func unique(a []string) []string {
	b := []string{}
	for _, i := range a {
		found := false
		for _, j := range b {
			if i == j {
				found = true
			}
		}
		if !found {
			b = append(b, i)
		}
	}
	return b
}

type FileDiffType string

const (
	FileDiffAdd FileDiffType = "add"
	FileDiffDel FileDiffType = "del"
	FileDiffMod FileDiffType = "mod"
)

// FileDiff describes a file update
type FileDiff struct {
	// Path of the file being updated
	Path string

	// Type of the file update
	Type FileDiffType

	// Time of modification for the file
	Mod time.Time

	// Content is the content of the file
	Content []byte
}

func calcDiff(sources []*Source, contractsDir string, files []*fileRef) ([]*FileDiff, error) {
	diff := []*FileDiff{}

	sourcesMap := map[string]*Source{}
	for _, src := range sources {
		sourcesMap[filepath.Join(src.Dir, src.Filename)] = src
	}

	visited := map[string]struct{}{}
	for _, file := range files {
		visited[file.path] = struct{}{}

		if src, ok := sourcesMap[file.path]; ok {
			if !src.ModTime.Equal(file.modTime) {
				// mod file
				content, err := ioutil.ReadFile(filepath.Join(contractsDir, file.path))
				if err != nil {
					return nil, err
				}

				diff = append(diff, &FileDiff{
					Path:    file.path,
					Type:    FileDiffMod,
					Mod:     file.modTime,
					Content: content,
				})
			}
		} else {
			// new file
			content, err := ioutil.ReadFile(filepath.Join(contractsDir, file.path))
			if err != nil {
				return nil, err
			}

			diff = append(diff, &FileDiff{
				Path:    file.path,
				Type:    FileDiffAdd,
				Mod:     file.modTime,
				Content: content,
			})
		}
	}

	for path := range sourcesMap {
		if _, ok := visited[path]; !ok {
			// deleted
			diff = append(diff, &FileDiff{
				Path: path,
				Type: FileDiffDel,
				Mod:  time.Time{},
			})
		}
	}

	return diff, nil
}

func parseSource(content, path string) (*Source, error) {
	// new file
	dir, filename := filepath.Dir(path), filepath.Base(path)

	relImports := parseDependencies(string(content))

	absImports, err := resolveRelativeImports(relImports, dir)
	if err != nil {
		return nil, err
	}

	pragma, err := parsePragma(string(content))
	if err != nil {
		return nil, err
	}

	source := &Source{
		Dir:      dir,
		Filename: filename,
		Version:  pragma,
		Imports:  absImports,
	}
	return source, nil
}

func resolveRelativeImports(deps []string, path string) ([]string, error) {
	res := []string{}
	for _, dep := range deps {
		if !strings.HasPrefix(dep, ".") {
			// global import, return as it is
			res = append(res, dep)
		} else {
			// local
			fullPath := filepath.Join(path, dep)
			if strings.Contains(fullPath, "..") {
				// if even after the `Join` there are `..` on the path it means that
				// the supplied path is not up enough to cover all the dependencies which
				// should not happen.
				return nil, fmt.Errorf("path '%s' does not contain import '%s'", path, dep)
			}
			res = append(res, filepath.Join(path, dep))
		}
	}
	return res, nil
}

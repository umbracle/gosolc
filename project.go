package gosolc

import (
	"path/filepath"

	"github.com/umbracle/gosolc/svm"
)

type Project struct {
	// config is the configuration of the Solidity project
	config *Config

	// svm handles the lifecycle of the Solidity compiler binaries
	svm *svm.SolidityVersionManager

	sources []*Source

	contracts contractsList
}

type contractsList []*Contract

func (c *contractsList) Filter(cond func(c *Contract) bool) (res contractsList) {
	for _, cc := range *c {
		if cond(cc) {
			res = append(res, cc)
		}
	}
	return
}

func NewProject(opts ...Option) (*Project, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.ContractsDir = filepath.Clean(cfg.ContractsDir)

	// default artifacts directory to the contracts directory if not set
	if cfg.ArtifactsDir == "" {
		cfg.ArtifactsDir = cfg.ContractsDir
	} else {
		cfg.ArtifactsDir = filepath.Clean(cfg.ArtifactsDir)
	}

	p := &Project{
		config:    cfg,
		sources:   []*Source{},
		contracts: []*Contract{},
	}

	svm, err := svm.NewSolidityVersionManager()
	if err != nil {
		return nil, err
	}
	p.svm = svm

	return p, nil
}

func (p *Project) findContractByFullName(name string) *Contract {
	res := p.contracts.Filter(func(c *Contract) bool {
		return name == c.Source+":"+c.Name
	})
	if len(res) != 1 {
		return nil
	}
	return res[0]
}

func (p *Project) getSourceByPath(path string) *Source {
	for _, s := range p.sources {
		sourcePath := filepath.Join(s.Dir, s.Filename)
		if path == sourcePath {
			return s
		}
	}
	return nil
}

func (p *Project) ListContracts() ([]*Contract, error) {
	return p.contracts, nil
}

func (p *Project) ListSources() ([]*Source, error) {
	return p.sources, nil
}

func (p *Project) UpsertContract(c *Contract) error {
	for indx, cc := range p.contracts {
		if cc.Source == c.Source && cc.Name == c.Name {
			p.contracts[indx] = c
			return nil
		}
	}
	p.contracts = append(p.contracts, c)
	return nil
}

func (p *Project) UpsertSource(src *Source) error {
	for indx, ss := range p.sources {
		if ss.Dir == src.Dir && ss.Filename == src.Filename {
			p.sources[indx] = src
			return nil
		}
	}
	p.sources = append(p.sources, src)
	return nil
}

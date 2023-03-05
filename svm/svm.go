package svm

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type config struct {
	logger *log.Logger
	dir    string
}

type Option func(*config)

func WithLogger(logger *log.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

func WithDir(dir string) Option {
	return func(c *config) {
		c.dir = dir
	}
}

// SolidityVersionManager is a service to manage solidity compiler versions
type SolidityVersionManager struct {
	config *config
}

// NewSolidityVersionManager creates a new Solidity Version Manager
func NewSolidityVersionManager(opts ...Option) (*SolidityVersionManager, error) {
	cfg := &config{
		logger: log.New(ioutil.Discard, "", 0),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.dir == "" {
		// use the default $HOME/.solc-svm dir
		dirname, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %v", err)
		}
		cfg.dir = filepath.Join(dirname, ".solc-svm")
	}

	s := &SolidityVersionManager{
		config: cfg,
	}
	return s, nil
}

// Resolve returns the path for the compiler and downloads it if necessary
func (s *SolidityVersionManager) Resolve(version string) (string, error) {
	path := filepath.Join(s.config.dir, "solidity-"+version)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.config.logger.Printf("[INFO]: Downloading solc compiler (%s)...\n", version)

			// download the compiler
			if err := downloadSolidity(version, s.config.dir); err != nil {
				return "", err
			}
		} else {
			// unexpected error
			return "", err
		}
	}

	return path, nil
}

func downloadSolidity(version string, dst string) error {
	url := "https://github.com/ethereum/solidity/releases/download/v" + version + "/solc-static-linux"

	// check if the dst is correct
	exists := false
	fi, err := os.Stat(dst)
	if err == nil {
		switch mode := fi.Mode(); {
		case mode.IsDir():
			exists = true
		case mode.IsRegular():
			return fmt.Errorf("dst is a file")
		}
	} else {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat dst '%s': %v", dst, err)
		}
	}

	// create the destiny path if does not exists
	if !exists {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return fmt.Errorf("cannot create dst path: %v", err)
		}
	}

	// rename binary
	name := "solidity-" + version

	// tmp folder to download the binary
	tmpDir, err := ioutil.TempDir(dst, "solc-download-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, name)

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// make binary executable
	if err := os.Chmod(path, 0755); err != nil {
		return err
	}

	// move file to dst
	if err := os.Rename(path, filepath.Join(dst, name)); err != nil {
		return err
	}
	return nil
}

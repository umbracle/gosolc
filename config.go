package gosolc

const (
	defaultSolidityVersion = "0.8.4"
)

// Config is the Project configuration
type Config struct {
	ContractsDir    string
	ArtifactsDir    string
	SolidityVersion string
}

func DefaultConfig() *Config {
	return &Config{
		ContractsDir:    "",
		SolidityVersion: defaultSolidityVersion,
	}
}

type Option func(*Config)

func WithSolidityVersion(version string) Option {
	return func(c *Config) {
		c.SolidityVersion = version
	}
}

func WithArtifactsDir(artifactsDir string) Option {
	return func(c *Config) {
		c.ArtifactsDir = artifactsDir
	}
}

func WithContractsDir(contractsDir string) Option {
	return func(c *Config) {
		c.ContractsDir = contractsDir
	}
}

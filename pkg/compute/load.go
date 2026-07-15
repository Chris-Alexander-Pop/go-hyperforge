package compute

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
)

// Config is the root env-tagged configuration for compute capability drivers.
// Subpackages retain their own Config types; this aggregates common driver selection.
type Config struct {
	// VMDriver selects VM backend: memory, ec2, gce, azure-vm.
	VMDriver string `env:"VM_DRIVER" env-default:"memory"`

	// ContainerDriver selects container backend: memory, docker, k8s, fargate.
	ContainerDriver string `env:"CONTAINER_DRIVER" env-default:"memory"`

	// ServerlessDriver selects serverless backend: memory, lambda, gcf, azure-functions.
	ServerlessDriver string `env:"SERVERLESS_DRIVER" env-default:"memory"`
}

// LoadConfig loads compute.Config via pkg/config (env / optional .env) and validates it.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

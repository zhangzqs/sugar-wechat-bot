package autoconfig

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfig[T any]() (*T, error) {
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("usage: %s <config-file>", os.Args[0])
	}

	cfgFile, err := os.ReadFile(os.Args[1])
	if err != nil {
		return nil, err
	}

	var cfg T
	if err = yaml.Unmarshal(cfgFile, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func MustLoadConfig[T any]() *T {
	cfg, err := LoadConfig[T]()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

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

	cfgFile, err := os.Open(os.Args[1])
	if err != nil {
		return nil, err
	}
	defer cfgFile.Close()

	var cfg T
	decoder := yaml.NewDecoder(cfgFile)
	decoder.KnownFields(true) // Enable strict mode to catch unknown fields
	if err = decoder.Decode(&cfg); err != nil {
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

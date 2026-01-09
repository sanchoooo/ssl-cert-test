package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
)

// Export DefaultPorts so main can use it
var (
	DefaultPorts = []int{443, 5091, 5061}
)

func LoadConfig(filePath string) (Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %v", err)
	}

	if len(config.Domains) == 0 {
		if len(config.Cidr) == 0 {
			return Config{}, errors.New("invalid config: missing required domains or cidr")
		}
	}

	var ips []string
	for _, r := range config.Cidr {
		// Ensure ConvertCidrToIPList is defined in helpers.go or utils.go
		ips, err = ConvertCidrToIPList(r) 
		if err != nil {
			log.Fatalf("Error loading cidr configuration: %v", err)
		}
		config.Domains = append(config.Domains, ips...)
	}

	if len(config.Ports) == 0 {
		config.Ports = DefaultPorts // Updated to use exported variable
	}

	return config, nil
}
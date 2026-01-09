package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/3th1nk/cidr"
)

// ParsePorts parses a comma-separated string of ports into a slice of ints
func ParsePorts(portString string) ([]int, error) {
	if portString == "" {
		return nil, nil
	}
	parts := strings.Split(portString, ",")
	ports := make([]int, 0, len(parts))
	for _, part := range parts {
		port, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid port value: %s", part)
		}
		ports = append(ports, port)
	}
	return ports, nil
}

// MergePorts merges two slices of ports and removes duplicates
func MergePorts(configPorts, cliPorts []int) []int {
	portSet := make(map[int]struct{})
	for _, port := range configPorts {
		portSet[port] = struct{}{}
	}
	for _, port := range cliPorts {
		portSet[port] = struct{}{}
	}

	mergedPorts := make([]int, 0, len(portSet))
	for port := range portSet {
		mergedPorts = append(mergedPorts, port)
	}

	sort.Ints(mergedPorts) 
	return mergedPorts
}

// ConvertCidrToIPList converts a CIDR string (e.g., "10.0.0.1/24") to a list of IPs
func ConvertCidrToIPList(ip string) ([]string, error) {
	// FIX: Capture the error here instead of using '_'
	c, err := cidr.Parse(ip)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CIDR: %w", err)
	}
	
	var ips []string
	c.Each(func(ip string) bool {
		ips = append(ips, ip)
		return true
	})
	return ips, nil
}
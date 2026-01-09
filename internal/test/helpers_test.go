package test

import (
	"reflect"
	"testing"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []int
		expectErr bool
	}{
		{
			name:      "Valid single port",
			input:     "443",
			want:      []int{443},
			expectErr: false,
		},
		{
			name:      "Valid multiple ports",
			input:     "80,443,8080",
			want:      []int{80, 443, 8080},
			expectErr: false,
		},
		{
			name:      "Valid with whitespace",
			input:     " 80 , 443 ,  8080 ",
			want:      []int{80, 443, 8080},
			expectErr: false,
		},
		{
			name:      "Empty string",
			input:     "",
			want:      nil,
			expectErr: false,
		},
		{
			name:      "Invalid non-number",
			input:     "443,abc,80",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Invalid mixed format",
			input:     "443,,80", // double comma
			want:      nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePorts(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ParsePorts() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePorts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergePorts(t *testing.T) {
	tests := []struct {
		name        string
		configPorts []int
		cliPorts    []int
		want        []int
	}{
		{
			name:        "Merge distinct lists",
			configPorts: []int{80, 443},
			cliPorts:    []int{8080, 8443},
			want:        []int{80, 443, 8080, 8443},
		},
		{
			name:        "Merge with overlaps (deduplication)",
			configPorts: []int{80, 443},
			cliPorts:    []int{443, 8080},
			want:        []int{80, 443, 8080},
		},
		{
			name:        "Handle empty config",
			configPorts: []int{},
			cliPorts:    []int{443},
			want:        []int{443},
		},
		{
			name:        "Handle empty CLI",
			configPorts: []int{80},
			cliPorts:    nil,
			want:        []int{80},
		},
		{
			name:        "Sorts output",
			configPorts: []int{9000},
			cliPorts:    []int{80},
			want:        []int{80, 9000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergePorts(tt.configPorts, tt.cliPorts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergePorts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertCidrToIPList(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		wantLen   int    // We check length to avoid hardcoding huge IP lists
		firstIP   string // Check the first IP to ensure logic is sound
		expectErr bool
	}{
		{
			name:      "Valid /30 CIDR (4 IPs)",
			cidr:      "192.168.1.0/30",
			wantLen:   4,
			firstIP:   "192.168.1.0",
			expectErr: false,
		},
		{
			name:      "Single IP /32",
			cidr:      "10.0.0.1/32",
			wantLen:   1,
			firstIP:   "10.0.0.1",
			expectErr: false,
		},
		{
			name:      "Invalid CIDR",
			cidr:      "999.999.999.999/24",
			wantLen:   0,
			firstIP:   "",
			expectErr: true,
		},
		{
			name:      "Garbage input",
			cidr:      "not-an-ip",
			wantLen:   0,
			firstIP:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertCidrToIPList(tt.cidr)
			if (err != nil) != tt.expectErr {
				t.Errorf("ConvertCidrToIPList() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				if len(got) != tt.wantLen {
					t.Errorf("ConvertCidrToIPList() length = %d, want %d", len(got), tt.wantLen)
				}
				if len(got) > 0 && got[0] != tt.firstIP {
					t.Errorf("ConvertCidrToIPList() first IP = %s, want %s", got[0], tt.firstIP)
				}
			}
		})
	}
}

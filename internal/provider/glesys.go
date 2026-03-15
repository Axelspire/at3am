package provider

import (
	"fmt"

	"github.com/libdns/glesys"
)

// newGleSYS configures a GleSYS provider.
// project and api_key are required (project is the GleSYS customer number, e.g. "CL12345").
func newGleSYS(creds map[string]string) (DNSProvider, error) {
	if creds["project"] == "" {
		return nil, fmt.Errorf("glesys: 'project' is required in credentials file (e.g. CL12345)")
	}
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("glesys: 'api_key' is required in credentials file")
	}
	return &glesys.Provider{
		Project: creds["project"],
		APIKey:  creds["api_key"],
	}, nil
}


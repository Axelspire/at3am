package provider

import (
	"fmt"

	"github.com/libdns/powerdns"
)

// newPowerDNS configures a PowerDNS provider.
// server_url and api_token are required.
// server_id defaults to "localhost" when empty.
func newPowerDNS(creds map[string]string) (DNSProvider, error) {
	if creds["server_url"] == "" {
		return nil, fmt.Errorf("powerdns: 'server_url' is required in credentials file")
	}
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("powerdns: 'api_token' is required in credentials file")
	}
	serverID := creds["server_id"]
	if serverID == "" {
		serverID = "localhost"
	}
	return &powerdns.Provider{
		ServerURL: creds["server_url"],
		APIToken:  creds["api_token"],
		ServerID:  serverID,
	}, nil
}


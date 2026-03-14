package provider

import (
	"fmt"

	"github.com/libdns/bluecat"
)

// newBlueCat configures a BlueCat Address Manager provider.
// server_url, username, password, configuration_name, and view_name are required.
// dns_deployed_server is optional; when set, records are immediately deployed to
// that managed DNS server (e.g. "bdds1.example.com") after being written to BAM.
func newBlueCat(creds map[string]string) (DNSProvider, error) {
	if creds["server_url"] == "" {
		return nil, fmt.Errorf("bluecat: 'server_url' is required (e.g. https://bam.example.com)")
	}
	if creds["username"] == "" {
		return nil, fmt.Errorf("bluecat: 'username' is required in credentials file")
	}
	if creds["password"] == "" {
		return nil, fmt.Errorf("bluecat: 'password' is required in credentials file")
	}
	if creds["configuration_name"] == "" {
		return nil, fmt.Errorf("bluecat: 'configuration_name' is required in credentials file")
	}
	if creds["view_name"] == "" {
		return nil, fmt.Errorf("bluecat: 'view_name' is required in credentials file")
	}
	return &bluecat.Provider{
		ServerURL:         creds["server_url"],
		Username:          creds["username"],
		Password:          creds["password"],
		ConfigurationName: creds["configuration_name"],
		ViewName:          creds["view_name"],
	}, nil
}


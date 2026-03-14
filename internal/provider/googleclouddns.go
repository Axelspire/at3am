package provider

import (
	"fmt"

	"github.com/libdns/googleclouddns"
)

// newGoogleCloudDNS configures a Google Cloud DNS provider.
// gcp_project is required. gcp_application_default may be a path to a
// service account JSON file; when empty, Application Default Credentials are used.
func newGoogleCloudDNS(creds map[string]string) (DNSProvider, error) {
	project := creds["gcp_project"]
	if project == "" {
		return nil, fmt.Errorf("googleclouddns: 'gcp_project' is required in credentials file")
	}
	return &googleclouddns.Provider{
		Project:            project,
		ServiceAccountJSON: creds["gcp_application_default"],
	}, nil
}


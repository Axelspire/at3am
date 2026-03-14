package provider

import (
	"fmt"

	"github.com/libdns/scaleway"
)

// newScaleway configures a Scaleway DNS provider.
// secret_key is required. organization_id is optional.
func newScaleway(creds map[string]string) (DNSProvider, error) {
	if creds["secret_key"] == "" {
		return nil, fmt.Errorf("scaleway: 'secret_key' is required in credentials file")
	}
	return &scaleway.Provider{
		SecretKey:      creds["secret_key"],
		OrganizationID: creds["organization_id"],
	}, nil
}


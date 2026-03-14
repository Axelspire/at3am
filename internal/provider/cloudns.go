package provider

import (
	"fmt"

	"github.com/libdns/cloudns"
)

// newClouDNS configures a ClouDNS provider.
// auth_id (or sub_auth_id) and auth_password are required.
// Provide either auth_id for a master account or sub_auth_id for a sub-user.
func newClouDNS(creds map[string]string) (DNSProvider, error) {
	if creds["auth_password"] == "" {
		return nil, fmt.Errorf("cloudns: 'auth_password' is required in credentials file")
	}
	if creds["auth_id"] == "" && creds["sub_auth_id"] == "" {
		return nil, fmt.Errorf("cloudns: either 'auth_id' or 'sub_auth_id' is required in credentials file")
	}
	return &cloudns.Provider{
		AuthId:       creds["auth_id"],
		SubAuthId:    creds["sub_auth_id"],
		AuthPassword: creds["auth_password"],
	}, nil
}


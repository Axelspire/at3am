package provider

import (
	"fmt"

	"github.com/libdns/inwx"
)

// newINWX configures an INWX provider.
// username and password are required. shared_secret is required only when
// "Mobile TAN" (2FA) is enabled on the account. endpoint_url defaults to the
// production API when empty.
func newINWX(creds map[string]string) (DNSProvider, error) {
	if creds["username"] == "" {
		return nil, fmt.Errorf("inwx: 'username' is required in credentials file")
	}
	if creds["password"] == "" {
		return nil, fmt.Errorf("inwx: 'password' is required in credentials file")
	}
	return &inwx.Provider{
		Username:    creds["username"],
		Password:    creds["password"],
		SharedSecret: creds["shared_secret"],
		EndpointURL: creds["endpoint_url"],
	}, nil
}


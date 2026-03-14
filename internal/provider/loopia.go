package provider

import (
	"fmt"

	"github.com/libdns/loopia"
)

// newLoopia configures a Loopia provider.
// username and password are required (Loopia API user, not your login).
// customer is optional (Loopia customer number for reseller accounts).
func newLoopia(creds map[string]string) (DNSProvider, error) {
	if creds["username"] == "" {
		return nil, fmt.Errorf("loopia: 'username' is required in credentials file (API user)")
	}
	if creds["password"] == "" {
		return nil, fmt.Errorf("loopia: 'password' is required in credentials file (API password)")
	}
	return &loopia.Provider{
		Username: creds["username"],
		Password: creds["password"],
		Customer: creds["customer"],
	}, nil
}


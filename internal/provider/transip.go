package provider

import (
	"fmt"

	"github.com/libdns/transip"
)

// newTransIP configures a TransIP provider.
// login is the TransIP account name. A temporary access token is generated
// automatically using the private key stored in private_key_path (path to a
// PEM file) or inline in private_key.
func newTransIP(creds map[string]string) (DNSProvider, error) {
	if creds["login"] == "" {
		return nil, fmt.Errorf("transip: 'login' is required in credentials file")
	}
	return &transip.Provider{
		AuthLogin: creds["login"],
	}, nil
}


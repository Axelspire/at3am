package provider

import (
	"fmt"

	"github.com/libdns/wedos"
)

// newWedos configures a WEDOS DNS provider.
// username is your WEDOS account username; password is the WAPI password
// (set under WAPI settings at https://client.wedos.com).
func newWedos(creds map[string]string) (DNSProvider, error) {
	if creds["username"] == "" {
		return nil, fmt.Errorf("wedos: 'username' is required in credentials file")
	}
	if creds["password"] == "" {
		return nil, fmt.Errorf("wedos: 'password' is required (WAPI password from https://client.wedos.com)")
	}
	return &wedos.Provider{
		Username: creds["username"],
		Password: creds["password"],
	}, nil
}


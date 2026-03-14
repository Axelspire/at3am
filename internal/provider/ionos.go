package provider

import (
	"fmt"

	"github.com/libdns/ionos"
)

func newIONOS(creds map[string]string) (DNSProvider, error) {
	if creds["auth_api_token"] == "" {
		return nil, fmt.Errorf("ionos: 'auth_api_token' is required in credentials file")
	}
	return &ionos.Provider{AuthAPIToken: creds["auth_api_token"]}, nil
}


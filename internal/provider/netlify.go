package provider

import (
	"fmt"

	"github.com/libdns/netlify"
)

func newNetlify(creds map[string]string) (DNSProvider, error) {
	if creds["personal_access_token"] == "" {
		return nil, fmt.Errorf("netlify: 'personal_access_token' is required in credentials file")
	}
	return &netlify.Provider{PersonalAccessToken: creds["personal_access_token"]}, nil
}


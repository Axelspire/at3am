package provider

import (
	"fmt"

	"github.com/libdns/gandi"
)

func newGandi(creds map[string]string) (DNSProvider, error) {
	token := creds["bearer_token"]
	if token == "" {
		return nil, fmt.Errorf("gandi: 'bearer_token' is required in credentials file")
	}
	return &gandi.Provider{BearerToken: token}, nil
}


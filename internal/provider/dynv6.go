package provider

import (
	"fmt"

	"github.com/libdns/dynv6"
)

func newDynv6(creds map[string]string) (DNSProvider, error) {
	if creds["token"] == "" {
		return nil, fmt.Errorf("dynv6: 'token' is required in credentials file (generate at https://dynv6.com/keys)")
	}
	return &dynv6.Provider{Token: creds["token"]}, nil
}


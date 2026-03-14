package provider

import (
	"fmt"

	"github.com/libdns/desec"
)

func newDeSEC(creds map[string]string) (DNSProvider, error) {
	if creds["token"] == "" {
		return nil, fmt.Errorf("desec: 'token' is required in credentials file (generate at https://desec.io/tokens)")
	}
	return &desec.Provider{Token: creds["token"]}, nil
}


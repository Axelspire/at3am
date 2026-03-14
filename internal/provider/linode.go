package provider

import (
	"fmt"

	"github.com/libdns/linode"
)

func newLinode(creds map[string]string) (DNSProvider, error) {
	token := creds["api_token"]
	if token == "" {
		return nil, fmt.Errorf("linode: 'api_token' is required in credentials file")
	}
	return &linode.Provider{APIToken: token}, nil
}


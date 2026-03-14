package provider

import (
	"fmt"

	"github.com/libdns/luadns"
)

func newLuaDNS(creds map[string]string) (DNSProvider, error) {
	if creds["email"] == "" {
		return nil, fmt.Errorf("luadns: 'email' is required in credentials file")
	}
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("luadns: 'api_key' is required in credentials file")
	}
	return &luadns.Provider{
		Email:  creds["email"],
		APIKey: creds["api_key"],
	}, nil
}


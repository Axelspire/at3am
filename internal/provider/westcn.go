package provider

import (
	"fmt"

	"github.com/libdns/westcn"
)

// newWestCN configures a West.cn DNS provider.
// username is your West.cn account username; api_password is the API password
// (separate from your login password — set at https://www.west.cn/CustomerCenter).
func newWestCN(creds map[string]string) (DNSProvider, error) {
	if creds["username"] == "" {
		return nil, fmt.Errorf("westcn: 'username' is required in credentials file")
	}
	if creds["api_password"] == "" {
		return nil, fmt.Errorf("westcn: 'api_password' is required in credentials file")
	}
	return &westcn.Provider{
		Username:    creds["username"],
		APIPassword: creds["api_password"],
	}, nil
}


package provider

import (
	"fmt"

	"github.com/libdns/directadmin"
)

// newDirectAdmin configures a DirectAdmin provider.
// server_url, user, and login_key are required.
// The login key needs CMD_API_SHOW_DOMAINS and CMD_API_DNS_CONTROL permissions.
func newDirectAdmin(creds map[string]string) (DNSProvider, error) {
	if creds["server_url"] == "" {
		return nil, fmt.Errorf("directadmin: 'server_url' is required (e.g. https://cp.example.com:2222)")
	}
	if creds["user"] == "" {
		return nil, fmt.Errorf("directadmin: 'user' is required in credentials file")
	}
	if creds["login_key"] == "" {
		return nil, fmt.Errorf("directadmin: 'login_key' is required in credentials file")
	}
	return &directadmin.Provider{
		ServerURL: creds["server_url"],
		User:      creds["user"],
		LoginKey:  creds["login_key"],
	}, nil
}


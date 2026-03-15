package provider

import (
	"fmt"

	"github.com/libdns/godaddy"
)

// newGoDaddy configures a GoDaddy provider.
// api_token must be in "key:secret" format as shown in the GoDaddy developer portal.
func newGoDaddy(creds map[string]string) (DNSProvider, error) {
	token := creds["api_token"]
	if token == "" {
		return nil, fmt.Errorf("godaddy: 'api_token' is required (format: 'key:secret')")
	}
	return &godaddy.Provider{APIToken: token}, nil
}


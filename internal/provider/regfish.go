package provider

import (
	"fmt"

	"github.com/libdns/regfish"
)

// newRegfish configures a Regfish DNS provider.
// api_token is required (generate at https://app.regfish.de).
func newRegfish(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("regfish: 'api_token' is required in credentials file")
	}
	return &regfish.Provider{APIToken: creds["api_token"]}, nil
}


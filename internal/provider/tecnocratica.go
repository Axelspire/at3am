package provider

import (
	"fmt"

	"github.com/libdns/tecnocratica"
)

// newTecnocratica configures a Tecnocratica / NeoDigit DNS provider.
// api_token is required. api_url defaults to the production endpoint when empty.
func newTecnocratica(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("tecnocratica: 'api_token' is required in credentials file")
	}
	return &tecnocratica.Provider{
		APIToken: creds["api_token"],
		APIURL:   creds["api_url"],
	}, nil
}


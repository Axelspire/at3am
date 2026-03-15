package provider

import (
	"fmt"

	allinkl "github.com/libdns/all-inkl"
)

// newAllInkl configures an ALL-INKL.COM provider.
// kas_username is your ALL-INKL KAS username; kas_password is the KAS API password
// (set under KAS → Einstellungen → KAS-API in the customer panel).
// Note: the ALL-INKL API enforces a 2.5 s inter-request flood-delay; operations
// on this provider will be noticeably slower than on REST-based providers.
func newAllInkl(creds map[string]string) (DNSProvider, error) {
	if creds["kas_username"] == "" {
		return nil, fmt.Errorf("all-inkl: 'kas_username' is required in credentials file")
	}
	if creds["kas_password"] == "" {
		return nil, fmt.Errorf("all-inkl: 'kas_password' is required in credentials file")
	}
	return &allinkl.Provider{
		KasUsername: creds["kas_username"],
		KasPassword: creds["kas_password"],
	}, nil
}


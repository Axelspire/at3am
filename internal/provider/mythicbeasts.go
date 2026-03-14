package provider

import (
	"fmt"

	"github.com/libdns/mythicbeasts"
)

// newMythicBeasts configures a Mythic Beasts DNS provider.
// key_id and secret are the API credentials from https://auth.mythic-beasts.com
func newMythicBeasts(creds map[string]string) (DNSProvider, error) {
	if creds["key_id"] == "" {
		return nil, fmt.Errorf("mythicbeasts: 'key_id' is required in credentials file")
	}
	if creds["secret"] == "" {
		return nil, fmt.Errorf("mythicbeasts: 'secret' is required in credentials file")
	}
	return &mythicbeasts.Provider{
		KeyID:  creds["key_id"],
		Secret: creds["secret"],
	}, nil
}


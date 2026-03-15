package provider

import (
	"fmt"

	"github.com/libdns/ovh"
)

// newOVH configures an OVH provider.
// endpoint is one of: ovh-eu, ovh-us, ovh-ca, soyoustart-eu, soyoustart-ca, kimsufi-eu, kimsufi-ca
func newOVH(creds map[string]string) (DNSProvider, error) {
	if creds["application_key"] == "" {
		return nil, fmt.Errorf("ovh: 'application_key' is required in credentials file")
	}
	if creds["application_secret"] == "" {
		return nil, fmt.Errorf("ovh: 'application_secret' is required in credentials file")
	}
	if creds["consumer_key"] == "" {
		return nil, fmt.Errorf("ovh: 'consumer_key' is required in credentials file")
	}
	endpoint := creds["endpoint"]
	if endpoint == "" {
		endpoint = "ovh-eu"
	}
	return &ovh.Provider{
		Endpoint:          endpoint,
		ApplicationKey:    creds["application_key"],
		ApplicationSecret: creds["application_secret"],
		ConsumerKey:       creds["consumer_key"],
	}, nil
}


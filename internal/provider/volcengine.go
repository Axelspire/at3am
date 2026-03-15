package provider

import (
	"fmt"

	"github.com/libdns/volcengine"
)

// newVolcEngine configures a Volcengine (ByteDance Cloud) DNS provider.
// access_key_id and access_key_secret are required.
// region_id defaults to "cn-beijing" when empty.
func newVolcEngine(creds map[string]string) (DNSProvider, error) {
	if creds["access_key_id"] == "" {
		return nil, fmt.Errorf("volcengine: 'access_key_id' is required in credentials file")
	}
	if creds["access_key_secret"] == "" {
		return nil, fmt.Errorf("volcengine: 'access_key_secret' is required in credentials file")
	}
	return &volcengine.Provider{
		CredentialInfo: volcengine.CredentialInfo{
			AccessKeyID:     creds["access_key_id"],
			AccessKeySecret: creds["access_key_secret"],
			RegionID:        creds["region_id"],
		},
	}, nil
}


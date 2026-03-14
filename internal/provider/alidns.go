package provider

import (
	"fmt"

	"github.com/libdns/alidns"
)

// newAliDNS configures an Alibaba Cloud DNS (AliDNS) provider.
// access_key_id and access_key_secret are required.
// region_id defaults to "cn-hangzhou" when empty.
// security_token is needed only for STS temporary credentials.
func newAliDNS(creds map[string]string) (DNSProvider, error) {
	if creds["access_key_id"] == "" {
		return nil, fmt.Errorf("alidns: 'access_key_id' is required in credentials file")
	}
	if creds["access_key_secret"] == "" {
		return nil, fmt.Errorf("alidns: 'access_key_secret' is required in credentials file")
	}
	return &alidns.Provider{
		CredentialInfo: alidns.CredentialInfo{
			AccessKeyID:     creds["access_key_id"],
			AccessKeySecret: creds["access_key_secret"],
			RegionID:        creds["region_id"],
			SecurityToken:   creds["security_token"],
		},
	}, nil
}


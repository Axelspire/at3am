package provider

import (
	"github.com/libdns/route53"
)

// newRoute53 configures an AWS Route53 provider.
// When access_key_id / secret_access_key are empty, the provider falls back to
// the standard AWS credential chain (env vars, ~/.aws/credentials, EC2 IMDS, etc.).
func newRoute53(creds map[string]string) (DNSProvider, error) {
	return &route53.Provider{
		AccessKeyId:     creds["access_key_id"],
		SecretAccessKey: creds["secret_access_key"],
		SessionToken:    creds["session_token"],
		Region:          creds["region"],
		Profile:         creds["profile"],
	}, nil
}


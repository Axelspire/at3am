package provider

import (
	"fmt"

	"github.com/libdns/tencentcloud"
)

// newTencentCloud configures a Tencent Cloud DNSPod provider.
// secret_id and secret_key are required. session_token is needed only for
// temporary STS credentials. region defaults to "ap-guangzhou" when empty.
func newTencentCloud(creds map[string]string) (DNSProvider, error) {
	if creds["secret_id"] == "" {
		return nil, fmt.Errorf("tencentcloud: 'secret_id' is required in credentials file")
	}
	if creds["secret_key"] == "" {
		return nil, fmt.Errorf("tencentcloud: 'secret_key' is required in credentials file")
	}
	return &tencentcloud.Provider{
		SecretId:     creds["secret_id"],
		SecretKey:    creds["secret_key"],
		SessionToken: creds["session_token"],
		Region:       creds["region"],
	}, nil
}


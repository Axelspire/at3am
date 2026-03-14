package provider

import (
	"fmt"

	"github.com/libdns/huaweicloud"
)

// newHuaweiCloud configures a Huawei Cloud DNS provider.
// access_key_id, secret_access_key, and region_id are required.
// iam_endpoint and dns_endpoint may be set to override the default public endpoints.
func newHuaweiCloud(creds map[string]string) (DNSProvider, error) {
	if creds["access_key_id"] == "" {
		return nil, fmt.Errorf("huaweicloud: 'access_key_id' is required in credentials file")
	}
	if creds["secret_access_key"] == "" {
		return nil, fmt.Errorf("huaweicloud: 'secret_access_key' is required in credentials file")
	}
	if creds["region_id"] == "" {
		return nil, fmt.Errorf("huaweicloud: 'region_id' is required in credentials file (e.g. cn-north-4)")
	}
	return &huaweicloud.Provider{
		AccessKeyId:     creds["access_key_id"],
		SecretAccessKey: creds["secret_access_key"],
		RegionId:        creds["region_id"],
	}, nil
}


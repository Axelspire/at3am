package provider

import (
	"fmt"

	"github.com/libdns/azure"
)

// newAzure configures an Azure DNS provider.
// subscription_id and resource_group_name are always required.
// tenant_id / client_id / client_secret are required for service-principal auth;
// omit all three to use Managed Identity / Azure CLI credentials instead.
func newAzure(creds map[string]string) (DNSProvider, error) {
	if creds["subscription_id"] == "" {
		return nil, fmt.Errorf("azure: 'subscription_id' is required in credentials file")
	}
	if creds["resource_group_name"] == "" {
		return nil, fmt.Errorf("azure: 'resource_group_name' is required in credentials file")
	}
	return &azure.Provider{
		SubscriptionId:    creds["subscription_id"],
		ResourceGroupName: creds["resource_group_name"],
		TenantId:          creds["tenant_id"],
		ClientId:          creds["client_id"],
		ClientSecret:      creds["client_secret"],
	}, nil
}


package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CredFile is the parsed YAML credentials file.
// Top-level fields:
//
//	provider: cloudflare
//	cloudflare:
//	  api_token: "..."
type CredFile struct {
	Provider string                       `yaml:"provider"`
	Sections map[string]map[string]string `yaml:",inline"`
}

// LoadCredentials reads the YAML file at path and returns the provider name
// and a flat key→value map of credentials for that provider section.
// Environment variables of the form AT3AM_DNS_<UPPER_KEY> override individual keys.
func LoadCredentials(path string) (providerName string, creds map[string]string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("credentials: cannot read %s: %w", path, err)
	}

	var raw struct {
		Provider string                 `yaml:"provider"`
		Extra    map[string]interface{} `yaml:",inline"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", nil, fmt.Errorf("credentials: parse error in %s: %w", path, err)
	}

	if raw.Provider == "" {
		return "", nil, fmt.Errorf("credentials: 'provider' field is missing in %s", path)
	}

	// Extract the section for this provider.
	creds = make(map[string]string)
	if section, ok := raw.Extra[raw.Provider]; ok {
		if m, ok := section.(map[string]interface{}); ok {
			for k, v := range m {
				creds[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Env-var overrides: AT3AM_DNS_API_TOKEN → key "api_token"
	for k := range creds {
		envKey := "AT3AM_DNS_" + strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
		if v := os.Getenv(envKey); v != "" {
			creds[k] = v
		}
	}
	// Also pick up env vars that weren't in the file (empty creds map case).
	for _, pair := range os.Environ() {
		const pfx = "AT3AM_DNS_"
		if !strings.HasPrefix(pair, pfx) {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(parts[0], pfx), "_", "_"))
		if _, exists := creds[k]; !exists {
			creds[k] = parts[1]
		}
	}

	return raw.Provider, creds, nil
}

// EnsureTemplate writes a commented YAML template for providerName to path if
// the file does not already exist. It returns true when the file was created.
func EnsureTemplate(path, providerName string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil // already exists
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, err
	}
	tmpl := buildTemplate(providerName)
	if err := os.WriteFile(path, []byte(tmpl), 0o600); err != nil {
		return false, fmt.Errorf("credentials: cannot write template to %s: %w", path, err)
	}
	return true, nil
}

func buildTemplate(name string) string {
	fields := templateFields(name)
	var sb strings.Builder
	sb.WriteString("# at3am-hook DNS credentials\n")
	sb.WriteString("# Generated automatically — fill in the values and re-run.\n")
	sb.WriteString("# Protect this file: chmod 600 <path>\n\n")
	sb.WriteString("provider: " + name + "\n\n")
	sb.WriteString(name + ":\n")
	for _, f := range fields {
		sb.WriteString("  " + f.key + ": \"" + f.placeholder + "\"\n")
	}
	return sb.String()
}

type field struct{ key, placeholder string }

func templateFields(name string) []field {
	switch name {
	case "cloudflare":
		return []field{{"api_token", "your-api-token"}}
	case "route53":
		return []field{
			{"access_key_id", "your-access-key-id"},
			{"secret_access_key", "your-secret-access-key"},
			{"region", "us-east-1"},
		}
	case "googleclouddns":
		return []field{
			{"gcp_project", "your-gcp-project-id"},
			{"gcp_application_default", "/path/to/service-account.json (or leave empty for ADC)"},
		}
	case "azure":
		return []field{
			{"subscription_id", "your-subscription-id"},
			{"resource_group_name", "your-resource-group"},
			{"tenant_id", "your-tenant-id (leave empty for managed identity)"},
			{"client_id", "your-client-id (leave empty for managed identity)"},
			{"client_secret", "your-client-secret (leave empty for managed identity)"},
		}
	case "digitalocean":
		return []field{{"auth_token", "your-do-api-token"}}
	case "hetzner":
		return []field{{"auth_api_token", "your-hetzner-dns-token"}}
	case "godaddy":
		return []field{{"api_token", "your-key:your-secret"}}
	case "namecheap":
		return []field{
			{"api_key", "your-api-key"},
			{"user", "your-username"},
		}
	case "porkbun":
		return []field{
			{"api_key", "your-api-key"},
			{"api_secret_key", "your-secret-key"},
		}
	case "ovh":
		return []field{
			{"endpoint", "ovh-eu"},
			{"application_key", "your-application-key"},
			{"application_secret", "your-application-secret"},
			{"consumer_key", "your-consumer-key"},
		}
	case "gandi":
		return []field{{"bearer_token", "your-bearer-token"}}
	case "linode":
		return []field{{"api_token", "your-linode-token"}}
	case "powerdns":
		return []field{
			{"server_url", "http://localhost:8081"},
			{"api_token", "your-pdns-api-token"},
			{"server_id", "localhost"},
		}
	case "dnsimple":
		return []field{
			{"api_access_token", "your-api-access-token"},
			{"account_id", "(optional — autodetected)"},
		}
	case "scaleway":
		return []field{
			{"secret_key", "your-secret-key"},
			{"organization_id", "(optional)"},
		}
	case "inwx":
		return []field{
			{"username", "your-inwx-username"},
			{"password", "your-inwx-password"},
			{"shared_secret", "(optional — only if 2FA/Mobile TAN enabled)"},
		}
	case "rfc2136":
		return []field{
			{"server", "ns1.example.com:53"},
			{"key_name", "your-tsig-key-name (leave empty for unauthenticated)"},
			{"key_alg", "hmac-sha256"},
			{"key", "base64-encoded-tsig-key"},
		}
	case "netlify":
		return []field{{"personal_access_token", "your-personal-access-token"}}
	case "tencentcloud":
		return []field{
			{"secret_id", "your-secret-id"},
			{"secret_key", "your-secret-key"},
			{"region", "ap-guangzhou"},
		}
	case "alidns":
		return []field{
			{"access_key_id", "your-access-key-id"},
			{"access_key_secret", "your-access-key-secret"},
			{"region_id", "cn-hangzhou"},
		}
	case "desec":
		return []field{{"token", "your-desec-token"}}
	case "transip":
		return []field{{"login", "your-transip-login"}}
	case "bunny":
		return []field{{"access_key", "your-bunny-api-key"}}
	case "ionos":
		return []field{{"auth_api_token", "your-ionos-api-token"}}
	case "namesilo":
		return []field{{"api_token", "your-namesilo-api-token"}}
	case "acmedns":
		return []field{
			{"server_url", "https://auth.acme-dns.io"},
			{"username", "your-acmedns-username"},
			{"password", "your-acmedns-password"},
			{"subdomain", "your-acmedns-subdomain"},
		}
	case "duckdns":
		return []field{
			{"api_token", "your-duckdns-token"},
			{"override_domain", "(optional — for CNAME delegation)"},
		}
	case "he":
		return []field{{"api_key", "your-he-ddns-key"}}
	case "infomaniak":
		return []field{{"api_token", "your-infomaniak-api-token"}}
	case "luadns":
		return []field{
			{"email", "your@email.com"},
			{"api_key", "your-luadns-api-key"},
		}
	case "cloudns":
		return []field{
			{"auth_id", "your-cloudns-auth-id (or leave empty and set sub_auth_id)"},
			{"sub_auth_id", "(optional — for sub-user auth)"},
			{"auth_password", "your-cloudns-auth-password"},
		}
	case "directadmin":
		return []field{
			{"server_url", "https://cp.example.com:2222"},
			{"user", "your-directadmin-user"},
			{"login_key", "your-login-key"},
		}
	case "loopia":
		return []field{
			{"username", "your-loopia-api-user"},
			{"password", "your-loopia-api-password"},
			{"customer", "(optional — reseller customer number)"},
		}
	case "glesys":
		return []field{
			{"project", "CL12345"},
			{"api_key", "your-glesys-api-key"},
		}
	case "huaweicloud":
		return []field{
			{"access_key_id", "your-access-key-id"},
			{"secret_access_key", "your-secret-access-key"},
			{"region_id", "cn-north-4"},
		}
	case "acmeproxy":
		return []field{
			{"address", "https://acmeproxy.example.com"},
			{"username", "(optional)"},
			{"password", "(optional)"},
		}
	case "dynv6":
		return []field{{"token", "your-dynv6-token"}}
	case "mythicbeasts":
		return []field{
			{"key_id", "your-key-id"},
			{"secret", "your-api-secret"},
		}
	case "simplydotcom":
		return []field{
			{"account_name", "your-simply-account-name"},
			{"api_key", "your-api-key"},
		}
	case "hosttech":
		return []field{{"api_token", "your-hosttech-jwt-bearer-token"}}
	case "dynu":
		return []field{{"api_token", "your-dynu-api-token"}}
	case "gcore":
		return []field{{"api_key", "your-gcore-permanent-api-key"}}
	// Batch 3
	case "all-inkl":
		return []field{
			{"kas_username", "your-kas-username"},
			{"kas_password", "your-kas-api-password"},
		}
	case "autodns":
		return []field{
			{"username", "your-autodns-username"},
			{"password", "your-autodns-password"},
			{"context", "4 (reseller context, or omit for default)"},
			{"endpoint", "(optional — leave empty for production)"},
		}
	case "bluecat":
		return []field{
			{"server_url", "https://bam.example.com"},
			{"username", "your-bam-username"},
			{"password", "your-bam-password"},
			{"configuration_name", "your-configuration-name (or leave empty for first)"},
			{"view_name", "your-view-name (or leave empty for first)"},
		}
	case "dnsupdate":
		return []field{
			{"addr", "ns1.example.com:53"},
			{"tsig", "(optional) hmac-sha256.:keyname.:base64secret=="},
		}
	case "dnsmadeeasy":
		return []field{
			{"api_key", "your-api-key"},
			{"secret_key", "your-secret-key"},
			{"api_endpoint", "(optional — leave empty for production)"},
		}
	case "domainnameshop":
		return []field{
			{"api_token", "your-api-token"},
			{"api_secret", "your-api-secret"},
		}
	case "mijnhost":
		return []field{{"api_key", "your-mijn-host-api-key"}}
	case "njalla":
		return []field{{"api_token", "your-njalla-api-token"}}
	case "regfish":
		return []field{{"api_token", "your-regfish-api-token"}}
	case "tecnocratica":
		return []field{
			{"api_token", "your-tecnocratica-api-token"},
			{"api_url", "(optional — leave empty for production)"},
		}
	case "unifi":
		return []field{
			{"api_key", "your-unifi-api-key"},
			{"base_url", "https://192.168.1.1/proxy/network/integration/v1"},
			{"site_id", "(optional — defaults to primary site)"},
		}
	case "websupport":
		return []field{
			{"api_key", "your-websupport-api-key"},
			{"api_secret", "your-websupport-api-secret"},
		}
	case "wedos":
		return []field{
			{"username", "your-wedos-username"},
			{"password", "your-wedos-wapi-password"},
		}
	case "westcn":
		return []field{
			{"username", "your-west-cn-username"},
			{"api_password", "your-west-cn-api-password"},
		}
	case "volcengine":
		return []field{
			{"access_key_id", "your-access-key-id"},
			{"access_key_secret", "your-access-key-secret"},
			{"region_id", "cn-beijing"},
		}
	default:
		return []field{{"token", "your-token"}}
	}
}


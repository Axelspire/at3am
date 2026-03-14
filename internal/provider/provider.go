// Package provider wraps libdns provider packages and adds autodetect,
// credential loading, zone discovery, and an early-access test.
//
// This package is a CONSUMER of the libdns interfaces, not an implementer.
// All provider struct types (cloudflare.Provider, route53.Provider, …) satisfy
// the DNSProvider interface automatically because they implement the standard
// libdns.RecordAppender and libdns.RecordDeleter interfaces.
//
// libdns is MIT-licensed. See https://github.com/libdns/libdns
package provider

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/libdns/libdns"
	"github.com/miekg/dns"

	logger "github.com/axelspire/at3am/internal/log"
)

// DNSProvider is the combined interface required by at3am-hook.
// Every libdns provider package satisfies it automatically.
type DNSProvider interface {
	libdns.RecordAppender
	libdns.RecordDeleter
}

// nsPattern maps a nameserver hostname suffix to a provider name.
type nsPattern struct {
	suffix   string
	provider string
}

// knownProviders is the ordered NS-suffix → provider-name table used for autodetect.
// Suffixes are matched case-insensitively against each NS hostname.
var knownProviders = []nsPattern{
	{".ns.cloudflare.com", "cloudflare"},
	{".awsdns-", "route53"}, // e.g. ns-123.awsdns-45.com
	{"awsdns-", "route53"},  // handles leading match
	{"ns-cloud-", "googleclouddns"},
	{".azure-dns.com", "azure"},
	{".azure-dns.net", "azure"},
	{".azure-dns.org", "azure"},
	{".azure-dns.info", "azure"},
	{"ns1.digitalocean.com", "digitalocean"},
	{"ns2.digitalocean.com", "digitalocean"},
	{"ns3.digitalocean.com", "digitalocean"},
	{"hydrogen.ns.hetzner.com", "hetzner"},
	{"oxygen.ns.hetzner.com", "hetzner"},
	{"helium.ns.hetzner.de", "hetzner"},
	{".domaincontrol.com", "godaddy"},
	{"dns1.registrar-servers.com", "namecheap"},
	{"dns2.registrar-servers.com", "namecheap"},
	{".porkbun.com", "porkbun"},
	{".ovh.net", "ovh"},
	{".ovh.ca", "ovh"},
	{"b.dns.gandi.net", "gandi"},
	{"a.dns.gandi.net", "gandi"},
	{"c.dns.gandi.net", "gandi"},
	{".linode.com", "linode"},
	{".linodedns.com", "linode"},
	// Additional providers (autodetect via NS)
	{"ns1.dnsimple.com", "dnsimple"},
	{"ns2.dnsimple.com", "dnsimple"},
	{"ns3.dnsimple.com", "dnsimple"},
	{"ns4.dnsimple.com", "dnsimple"},
	{".scw.cloud", "scaleway"},
	{"ns1.scaleway.com", "scaleway"},
	{"ns2.scaleway.com", "scaleway"},
	{"ns3.scaleway.com", "scaleway"},
	{".inwx.net", "inwx"},
	{".inwx.de", "inwx"},
	{"ns1.netlify.com", "netlify"},
	{"ns2.netlify.com", "netlify"},
	{"dns1.p01.nsone.net", "netcup"},
	{"ns1.tencentcloud-dns.com", "tencentcloud"},
	{"ns2.tencentcloud-dns.com", "tencentcloud"},
	{"ns3.tencentcloud-dns.com", "tencentcloud"},
	{"ns4.tencentcloud-dns.com", "tencentcloud"},
	{"ns3.alidns.com", "alidns"},
	{"ns4.alidns.com", "alidns"},
	{"desec.io", "desec"},
	{".transip.nl", "transip"},
	{".transip.net", "transip"},
	{"ns0.transip.net", "transip"},
	{"ns1.bunny.net", "bunny"},
	{"ns2.bunny.net", "bunny"},
	{"ns1103.ui-dns.com", "ionos"},
	{"ns1107.ui-dns.org", "ionos"},
	{"ns1106.ui-dns.biz", "ionos"},
	{"ns1108.ui-dns.de", "ionos"},
	{"ns1.namesilo.com", "namesilo"},
	{"ns2.namesilo.com", "namesilo"},
	{"ns1.duckdns.org", "duckdns"},
	{"auth1.dns.he.net", "he"},
	{"auth2.dns.he.net", "he"},
	{"auth3.dns.he.net", "he"},
	{"ns1.infomaniak.ch", "infomaniak"},
	{"ns2.infomaniak.ch", "infomaniak"},
	{"ns1.luadns.net", "luadns"},
	{"ns2.luadns.net", "luadns"},
	{"ns3.luadns.net", "luadns"},
	{"ns4.luadns.net", "luadns"},
	{"ns1.cloudns.net", "cloudns"},
	{"ns2.cloudns.net", "cloudns"},
	{"ns3.cloudns.net", "cloudns"},
	{"ns4.cloudns.net", "cloudns"},
	{"ns1.loopia.se", "loopia"},
	{"ns2.loopia.se", "loopia"},
	{"ns1.glesys.se", "glesys"},
	{"ns2.glesys.se", "glesys"},
	{"ns1.huaweicloud-dns.com", "huaweicloud"},
	{"ns2.huaweicloud-dns.com", "huaweicloud"},
	{"ns1.dynv6.com", "dynv6"},
	{"ns2.dynv6.com", "dynv6"},
	{"ns1.mythic-beasts.com", "mythicbeasts"},
	{"ns2.mythic-beasts.com", "mythicbeasts"},
	{"ns1.simply.com", "simplydotcom"},
	{"ns2.simply.com", "simplydotcom"},
	{"ns1.dreamhost.com", "dreamhost"},
	{"ns2.dreamhost.com", "dreamhost"},
	{"ns3.dreamhost.com", "dreamhost"},
	{"ns1.dynu.com", "dynu"},
	{"ns2.dynu.com", "dynu"},
	{"ns3.dynu.com", "dynu"},
	{"ns4.dynu.com", "dynu"},
	{"ns1.vercel-dns.com", "vercel"},
	{"ns2.vercel-dns.com", "vercel"},
	// Batch 3 NS patterns
	{"ns1.all-inkl.com", "all-inkl"},
	{"ns2.all-inkl.com", "all-inkl"},
	{"ns1.autodns.com", "autodns"},
	{"ns2.autodns.com", "autodns"},
	{"ns1.domainnameshop.com", "domainnameshop"},
	{"ns2.domainnameshop.com", "domainnameshop"},
	{"ns1.mijnhost.nl", "mijnhost"},
	{"ns2.mijnhost.nl", "mijnhost"},
	{"ns1.njalla.net", "njalla"},
	{"ns2.njalla.net", "njalla"},
	{"ns1.regfish.de", "regfish"},
	{"ns2.regfish.de", "regfish"},
	{"ns1.websupport.sk", "websupport"},
	{"ns2.websupport.sk", "websupport"},
	{"ns1.wedos.net", "wedos"},
	{"ns2.wedos.net", "wedos"},
	{"ns3.wedos.net", "wedos"},
	{"ns4.wedos.net", "wedos"},
	{"dns1.west.cn", "westcn"},
	{"dns2.west.cn", "westcn"},
	{".volcengine-dns.com", "volcengine"},
}

// Autodetect resolves the authoritative nameservers for domain and returns
// the matching provider name, or "" if no match is found.
func Autodetect(ctx context.Context, domain string) (string, error) {
	nss, err := discoverNS(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("NS discovery failed: %w", err)
	}
	logger.Debug("provider autodetect | domain=%s ns=%v", domain, nss)
	for _, ns := range nss {
		lower := strings.ToLower(ns)
		for _, p := range knownProviders {
			if strings.Contains(lower, p.suffix) || lower == p.suffix {
				logger.Info("provider autodetected | ns=%s provider=%s", ns, p.provider)
				return p.provider, nil
			}
		}
	}
	return "", nil
}

// DiscoverZone walks up the labels of domain (using Google's public DNS) and
// returns the apex zone name with a trailing dot, e.g. "example.com.".
func DiscoverZone(ctx context.Context, domain string) (string, error) {
	parts := strings.Split(strings.TrimSuffix(domain, "."), ".")
	for i := 0; i < len(parts)-1; i++ {
		zone := strings.Join(parts[i:], ".") + "."
		if hasNS(ctx, zone) {
			logger.Debug("zone discovered | domain=%s zone=%s", domain, zone)
			return zone, nil
		}
	}
	return "", fmt.Errorf("could not discover zone for %s", domain)
}

// RelativeName returns the record name relative to zone (without trailing dots).
// e.g. RelativeName("_acme-challenge.example.com.", "example.com.") → "_acme-challenge"
func RelativeName(fqdn, zone string) string {
	fqdn = strings.TrimSuffix(fqdn, ".")
	zone = strings.TrimSuffix(zone, ".")
	rel := strings.TrimSuffix(fqdn, "."+zone)
	if rel == zone {
		return "@"
	}
	return rel
}

// EarlyAccessTest creates a canary TXT record and immediately deletes it to
// verify that the credentials and zone access work before the real operation.
func EarlyAccessTest(ctx context.Context, p DNSProvider, zone string) error {
	epoch := time.Now().Unix()
	name := fmt.Sprintf("_at3am_test_%d", epoch)
	rec := libdns.TXT{Name: name, TTL: 30 * time.Second, Text: fmt.Sprintf("at3am_%d", epoch)}

	logger.Info("early access test | zone=%s name=%s", zone, name)

	appended, err := p.AppendRecords(ctx, zone, []libdns.Record{rec})
	if err != nil {
		return fmt.Errorf("early access test: create failed: %w", err)
	}
	_, err = p.DeleteRecords(ctx, zone, appended)
	if err != nil {
		return fmt.Errorf("early access test: delete failed (record may be left behind): %w", err)
	}
	logger.Info("early access test passed | zone=%s", zone)
	return nil
}

// discoverNS returns the NS hostnames for domain using Google's public DNS.
func discoverNS(ctx context.Context, domain string) ([]string, error) {
	parts := strings.Split(strings.TrimSuffix(domain, "."), ".")
	client := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	for i := 0; i < len(parts)-1; i++ {
		zone := strings.Join(parts[i:], ".")
		msg := new(dns.Msg)
		msg.SetQuestion(dns.Fqdn(zone), dns.TypeNS)
		msg.RecursionDesired = true
		resp, _, err := client.ExchangeContext(ctx, msg, "8.8.8.8:53")
		if err != nil || resp == nil {
			continue
		}
		var nss []string
		for _, rr := range resp.Answer {
			if ns, ok := rr.(*dns.NS); ok {
				nss = append(nss, strings.TrimSuffix(ns.Ns, "."))
			}
		}
		if len(nss) > 0 {
			return nss, nil
		}
	}
	return nil, fmt.Errorf("no NS records found for %s", domain)
}

// hasNS returns true if zone has NS records at 8.8.8.8.
func hasNS(ctx context.Context, zone string) bool {
	client := &dns.Client{Net: "udp", Timeout: 3 * time.Second}
	msg := new(dns.Msg)
	msg.SetQuestion(zone, dns.TypeNS)
	msg.RecursionDesired = true
	resp, _, err := client.ExchangeContext(ctx, msg, "8.8.8.8:53")
	if err != nil || resp == nil {
		return false
	}
	for _, rr := range resp.Answer {
		if _, ok := rr.(*dns.NS); ok {
			return true
		}
	}
	// Also check authority section (NXDOMAIN / referral)
	for _, rr := range resp.Ns {
		if soa, ok := rr.(*dns.SOA); ok {
			return strings.EqualFold(strings.TrimSuffix(soa.Hdr.Name, "."), strings.TrimSuffix(zone, "."))
		}
	}
	return false
}

// SupportedProviders returns the list of provider names that can be configured.
func SupportedProviders() []string {
	return []string{
		// Original 13
		"cloudflare", "route53", "googleclouddns", "azure",
		"digitalocean", "hetzner", "godaddy", "namecheap",
		"porkbun", "ovh", "gandi", "linode", "powerdns",
		// Additional 30
		"dnsimple", "scaleway", "inwx", "rfc2136", "netlify",
		"tencentcloud", "alidns", "desec", "transip", "bunny",
		"ionos", "namesilo", "acmedns", "duckdns", "he",
		"infomaniak", "luadns", "cloudns", "directadmin", "loopia",
		"glesys", "huaweicloud", "acmeproxy", "dynv6",
		"mythicbeasts", "simplydotcom", "netcup", "dynu", "gcore",
		// Batch 3 (from user list)
		"all-inkl", "autodns", "bluecat", "dnsupdate",
		"domainnameshop", "mijnhost", "njalla", "regfish", "tecnocratica",
		"unifi", "websupport", "wedos", "westcn", "volcengine",
	}
}

// Lookup returns a configured DNSProvider for the given provider name and credentials.
func Lookup(ctx context.Context, name string, creds map[string]string) (DNSProvider, error) {
	_ = net.LookupHost // keep net import
	switch name {
	// ── Original 13 ──────────────────────────────────────────────────────────
	case "cloudflare":
		return newCloudflare(creds)
	case "route53":
		return newRoute53(creds)
	case "googleclouddns":
		return newGoogleCloudDNS(creds)
	case "azure":
		return newAzure(creds)
	case "digitalocean":
		return newDigitalOcean(creds)
	case "hetzner":
		return newHetzner(creds)
	case "godaddy":
		return newGoDaddy(creds)
	case "namecheap":
		return newNamecheap(creds)
	case "porkbun":
		return newPorkbun(creds)
	case "ovh":
		return newOVH(creds)
	case "gandi":
		return newGandi(creds)
	case "linode":
		return newLinode(creds)
	case "powerdns":
		return newPowerDNS(creds)
	// ── Additional 30 ────────────────────────────────────────────────────────
	case "dnsimple":
		return newDNSimple(creds)
	case "scaleway":
		return newScaleway(creds)
	case "inwx":
		return newINWX(creds)
	case "rfc2136":
		return newRFC2136(creds)
	case "netlify":
		return newNetlify(creds)
	case "tencentcloud":
		return newTencentCloud(creds)
	case "alidns":
		return newAliDNS(creds)
	case "desec":
		return newDeSEC(creds)
	case "transip":
		return newTransIP(creds)
	case "bunny":
		return newBunny(creds)
	case "ionos":
		return newIONOS(creds)
	case "namesilo":
		return newNameSilo(creds)
	case "acmedns":
		return newACMEDNS(creds)
	case "duckdns":
		return newDuckDNS(creds)
	case "he":
		return newHurricaneElectric(creds)
	case "infomaniak":
		return newInfomaniak(creds)
	case "luadns":
		return newLuaDNS(creds)
	case "cloudns":
		return newClouDNS(creds)
	case "directadmin":
		return newDirectAdmin(creds)
	case "loopia":
		return newLoopia(creds)
	case "glesys":
		return newGleSYS(creds)
	case "huaweicloud":
		return newHuaweiCloud(creds)
	case "acmeproxy":
		return newACMEProxy(creds)
	case "dynv6":
		return newDynv6(creds)
	case "mythicbeasts":
		return newMythicBeasts(creds)
	case "simplydotcom":
		return newSimplyDotCom(creds)
	case "netcup":
		return newNetcup(creds)
	case "dynu":
		return newDynu(creds)
	case "gcore":
		return newGCore(creds)
	// ── Batch 3 ──────────────────────────────────────────────────────────────
	case "all-inkl":
		return newAllInkl(creds)
	case "autodns":
		return newAutoDNS(creds)
	case "bluecat":
		return newBlueCat(creds)
	case "dnsupdate":
		return newDNSUpdate(creds)
	case "domainnameshop":
		return newDomainNameShop(creds)
	case "mijnhost":
		return newMijnHost(creds)
	case "njalla":
		return newNjalla(creds)
	case "regfish":
		return newRegfish(creds)
	case "tecnocratica":
		return newTecnocratica(creds)
	case "unifi":
		return newUnifi(creds)
	case "websupport":
		return newWebsupport(creds)
	case "wedos":
		return newWedos(creds)
	case "westcn":
		return newWestCN(creds)
	case "volcengine":
		return newVolcEngine(creds)
	default:
		return nil, fmt.Errorf("unknown provider %q; supported: %s",
			name, strings.Join(SupportedProviders(), ", "))
	}
}


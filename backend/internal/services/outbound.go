package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// isInternalIP reports whether an address belongs to the infrastructure
// rather than the public internet.
func isInternalIP(ip net.IP) bool {
	return ip == nil ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast()
}

// NewGuardedHTTPClient returns a client that refuses to reach private,
// loopback or link-local addresses.
//
// Tenant-supplied URLs (the bot's "call webhook" node) were fetched with no
// restriction at all, and the response was surfaced back into the
// conversation — which turned the flow builder into a probe for anything
// reachable from the server, including the Evolution API, the database and
// cloud metadata endpoints.
//
// The check runs in DialContext, on the address actually being connected to,
// so it also covers redirects and leaves no window for a hostname to resolve
// to a public address during validation and a private one when dialled.
func NewGuardedHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, candidate := range ips {
					if isInternalIP(candidate.IP) {
						return nil, fmt.Errorf("destino de rede interna bloqueado: %s", candidate.IP)
					}
				}
				// Dial the address that was just validated rather than the
				// hostname, so it cannot resolve to something else now.
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
			},
		},
	}
}

// ValidatePublicURL rejects a tenant-supplied URL before it is ever fetched.
func ValidatePublicURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("URL inválida")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL deve usar http ou https")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("URL sem host")
	}
	// A literal private address is rejected up front; names are checked when
	// they resolve, in the dialer above.
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && isInternalIP(ip) {
		return fmt.Errorf("destino de rede interna bloqueado")
	}
	return nil
}

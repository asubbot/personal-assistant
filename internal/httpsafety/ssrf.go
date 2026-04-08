// Package httpsafety provides URL validation for outbound HTTPS fetches (EP-011, REQ-11.013, REQ-11.014).
package httpsafety

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrDisallowedURL is returned when a URL fails scheme or SSRF policy checks.
type ErrDisallowedURL struct {
	Reason string
}

func (e *ErrDisallowedURL) Error() string { return "httpsafety: " + e.Reason }

// ValidateFetchURL parses rawURL, enforces HTTPS-only, resolves the host, and rejects disallowed destinations.
//
//nolint:gocyclo // sequential policy checks; branches are independent guards
func ValidateFetchURL(ctx context.Context, rawURL string, resolver *net.Resolver) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return &ErrDisallowedURL{Reason: "invalid URL"}
	}
	if u.Scheme != "https" {
		return &ErrDisallowedURL{Reason: "only https URLs are allowed"}
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return &ErrDisallowedURL{Reason: "missing host"}
	}
	if metadataHost(host) {
		return &ErrDisallowedURL{Reason: "metadata hostnames are not allowed"}
	}
	if strings.Contains(u.Host, "@") {
		return &ErrDisallowedURL{Reason: "userinfo in URL is not allowed"}
	}

	if ip := net.ParseIP(host); ip != nil {
		if disallowedIP(ip) {
			return &ErrDisallowedURL{Reason: fmt.Sprintf("address %s is not allowed", ip.String())}
		}
		return nil
	}

	r := resolver
	if r == nil {
		r = net.DefaultResolver
	}
	ips, err := r.LookupIPAddr(ctx, host)
	if err != nil {
		return &ErrDisallowedURL{Reason: "could not resolve host"}
	}
	if len(ips) == 0 {
		return &ErrDisallowedURL{Reason: "host has no addresses"}
	}
	for _, ia := range ips {
		if disallowedIP(ia.IP) {
			return &ErrDisallowedURL{Reason: fmt.Sprintf("resolved address %s is not allowed", ia.IP.String())}
		}
	}
	return nil
}

func metadataHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "metadata.google.internal" || strings.HasSuffix(h, ".metadata.google.internal") {
		return true
	}
	if h == "metadata" {
		return true
	}
	return false
}

func disallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.Equal(net.IPv4zero) || ip.IsUnspecified() {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

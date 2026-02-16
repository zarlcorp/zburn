package namecheap

import (
	"context"
	"fmt"
	"net"
)

// openDNSResolver is the address of OpenDNS's resolver1.
const openDNSResolver = "208.67.222.222:53"

// detectPublicIP resolves the caller's public IP via DNS.
// It queries myip.opendns.com against resolver1.opendns.com.
func detectPublicIP(ctx context.Context) (string, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", openDNSResolver)
		},
	}

	addrs, err := r.LookupHost(ctx, "myip.opendns.com")
	if err != nil {
		return "", fmt.Errorf("detect public ip: %w", err)
	}

	if len(addrs) == 0 {
		return "", fmt.Errorf("detect public ip: no addresses returned")
	}

	return addrs[0], nil
}

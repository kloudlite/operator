package client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/kloudlite/operator/apps/multi-cluster/constants"
)

func wait() {
	time.Sleep(constants.ReconDuration * time.Second)
}

func generateIp(ip string) string {
	s := strings.Split(ip, ".")
	if len(s) != 4 {
		return ""
	}

	s2 := strings.Split(constants.IP_RANGE, ".")
	return fmt.Sprintf("%s.%s.%s.%s", s2[0], s2[1], s[2], s[3])
}

func customDialer(server string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		dnsServerAddress := fmt.Sprintf("%s:53", server)
		d := net.Dialer{
			Timeout: time.Millisecond * time.Duration(10000),
		}
		return d.DialContext(ctx, "udp", dnsServerAddress)
	}
}

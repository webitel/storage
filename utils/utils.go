package utils

import (
	"net"
	"net/http"
	"strings"

	"github.com/webitel/storage/model"
)

func GetIpAddress(r *http.Request) string {
	address := ""

	header := r.Header.Get(model.HEADER_FORWARDED)
	if len(header) > 0 {
		addresses := strings.Fields(header)
		if len(addresses) > 0 {
			address = strings.TrimRight(addresses[0], ",")
		}
	}

	if len(address) == 0 {
		address = r.Header.Get(model.HEADER_REAL_IP)
	}

	if len(address) == 0 {
		address, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	return address
}

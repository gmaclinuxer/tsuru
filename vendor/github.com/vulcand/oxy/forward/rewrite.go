package forward

import (
	"net"
	"net/http"
	"strings"

	"github.com/vulcand/oxy/utils"
)

// Rewriter is responsible for removing hop-by-hop headers and setting forwarding headers
type HeaderRewriter struct {
	TrustForwardHeader bool
	Hostname           string
}

// clean up IP in case if it is ipv6 address and it has {zone} infromation in it, like "[fe80::d806:a55d:eb1b:49cc%vEthernet (vmxnet3 Ethernet Adapter - Virtual Switch)]:64692"
func ipv6fix(clientIP string) string {
	return strings.Split(clientIP, "%")[0]
}

func (rw *HeaderRewriter) Rewrite(req *http.Request) {
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		clientIP = ipv6fix(clientIP)
		if rw.TrustForwardHeader {
			if prior, ok := req.Header[XForwardedFor]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
		}
		req.Header.Set(XForwardedFor, clientIP)
	}

	if xfp := req.Header.Get(XForwardedProto); xfp != "" && rw.TrustForwardHeader {
		req.Header.Set(XForwardedProto, xfp)
	} else if req.TLS != nil {
		req.Header.Set(XForwardedProto, "https")
	} else {
		req.Header.Set(XForwardedProto, "http")
	}

	if IsWebsocketRequest(req) {
		if req.Header.Get(XForwardedProto) == "https" {
			req.Header.Set(XForwardedProto, "wss")
		} else {
			req.Header.Set(XForwardedProto, "ws")
		}
	}

	if xfh := req.Header.Get(XForwardedHost); xfh != "" && rw.TrustForwardHeader {
		req.Header.Set(XForwardedHost, xfh)
	} else if req.Host != "" {
		req.Header.Set(XForwardedHost, req.Host)
	}

	if rw.Hostname != "" {
		req.Header.Set(XForwardedServer, rw.Hostname)
	}

	if !IsWebsocketRequest(req) {
		// Remove hop-by-hop headers to the backend.  Especially important is "Connection" because we want a persistent
		// connection, regardless of what the client sent to us.
		utils.RemoveHeaders(req.Header, HopHeaders...)
	}
}

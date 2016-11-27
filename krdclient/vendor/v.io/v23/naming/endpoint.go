// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package naming

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

const (
	separator          = "@"
	suffix             = "@@"
	blessingsSeparator = ","
	routeSeparator     = ","
)

var (
	errInvalidEndpointString = errors.New("invalid endpoint string")
	hostportEP               = regexp.MustCompile("^(?:\\((.*)\\)@)?([^@]+)$")
	// DefaultEndpointVersion is the default of endpoints that we will create
	// when the version is otherwise unspecified.
	DefaultEndpointVersion = 6
)

// Endpoint represents unique identifiers for entities communicating over a
// network.  End users don't use endpoints - they deal solely with object names,
// with the MountTable providing translation of object names to endpoints.
type Endpoint struct {
	Protocol string
	Address  string
	// RoutingID returns the RoutingID associated with this Endpoint.
	RoutingID     RoutingID
	routes        []string
	blessingNames []string

	// ServesMountTable is true if this endpoint serves a mount table.
	// TODO(mattr): Remove it?
	ServesMountTable bool
}

// ParseEndpoint returns an Endpoint by parsing the supplied endpoint
// string as per the format described above. It can be used to test
// a string to see if it's in valid endpoint format.
//
// NewEndpoint will accept strings both in the @ format described
// above and in internet host:port format.
//
// All implementations of NewEndpoint should provide appropriate
// defaults for any endpoint subfields not explicitly provided as
// follows:
// - a missing protocol will default to a protocol appropriate for the
//   implementation hosting NewEndpoint
// - a missing host:port will default to :0 - i.e. any port on all
//   interfaces
// - a missing routing id should default to the null routing id
// - a missing codec version should default to AnyCodec
// - a missing RPC version should default to the highest version
//   supported by the runtime implementation hosting NewEndpoint
func ParseEndpoint(input string) (Endpoint, error) {
	// If the endpoint does not end in a @, it must be in [blessing@]host:port format.
	if parts := hostportEP.FindStringSubmatch(input); len(parts) > 0 {
		hostport := parts[len(parts)-1]
		var blessing string
		if len(parts) > 2 {
			blessing = parts[1]
		}
		return parseHostPort(blessing, hostport)
	}

	// Trim the prefix and suffix and parse the rest.
	input = strings.TrimPrefix(strings.TrimSuffix(input, suffix), separator)
	parts := strings.Split(input, separator)
	version, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return Endpoint{}, fmt.Errorf("invalid version: %v", err)
	}

	switch version {
	case 6:
		return parseV6(parts)
	default:
		return Endpoint{}, errInvalidEndpointString
	}
}

func parseHostPort(blessing, hostport string) (Endpoint, error) {
	// Could be in host:port format.
	var ep Endpoint
	if _, _, err := net.SplitHostPort(hostport); err != nil {
		return ep, errInvalidEndpointString
	}
	if strings.HasSuffix(hostport, "#") {
		hostport = strings.TrimSuffix(hostport, "#")
	} else {
		ep.ServesMountTable = true
	}
	ep.Protocol = UnknownProtocol
	ep.Address = hostport
	ep.RoutingID = NullRoutingID
	if len(blessing) > 0 {
		ep.blessingNames = []string{blessing}
	}
	return ep, nil
}

func parseV6(parts []string) (Endpoint, error) {
	var ep Endpoint
	if len(parts) < 6 {
		return ep, errInvalidEndpointString
	}

	ep.Protocol = parts[1]
	if len(ep.Protocol) == 0 {
		ep.Protocol = UnknownProtocol
	}

	var ok bool
	if ep.Address, ok = Unescape(parts[2]); !ok {
		return ep, fmt.Errorf("invalid address: bad escape %s", parts[2])
	}
	if len(ep.Address) == 0 {
		ep.Address = net.JoinHostPort("", "0")
	}

	if len(parts[3]) > 0 {
		ep.routes = strings.Split(parts[3], routeSeparator)
		for i := range ep.routes {
			if ep.routes[i], ok = Unescape(ep.routes[i]); !ok {
				return ep, fmt.Errorf("invalid route: bad escape %s", ep.routes[i])
			}
		}
	}

	if err := ep.RoutingID.FromString(parts[4]); err != nil {
		return ep, fmt.Errorf("invalid routing id: %v", err)
	}
	switch p := parts[5]; p {
	case "", "m":
		ep.ServesMountTable = true
	case "s", "l":
	default:
		return ep, fmt.Errorf("invalid mount table flag (%v)", p)
	}
	// Join the remaining and re-split.
	if str := strings.Join(parts[6:], separator); len(str) > 0 {
		ep.blessingNames = strings.Split(str, blessingsSeparator)
	}
	return ep, nil
}

// WithBlessingNames derives a new endpoint with the given
// blessing names, but otherwise identical to e.
func (e Endpoint) WithBlessingNames(names []string) Endpoint {
	e.blessingNames = append([]string{}, names...)
	return e
}

// WithRoutes derives a new endpoint with the given
// blessing names, but otherwise identical to e.
func (e Endpoint) WithRoutes(routes []string) Endpoint {
	e.routes = append([]string{}, routes...)
	return e
}

// BlessingNames returns the blessings that the process associated with
// this Endpoint will present.
func (e Endpoint) BlessingNames() []string {
	return append([]string{}, e.blessingNames...)
}

// Routes returns the local routing identifiers used for proxying connections
// with multiple proxies.
func (e Endpoint) Routes() []string {
	return append([]string{}, e.routes...)
}

// IsZero returns true if the endpoint is equivalent to the zero value.
func (e Endpoint) IsZero() bool {
	return e.Protocol == "" &&
		e.Address == "" &&
		e.RoutingID == RoutingID{} &&
		len(e.routes) == 0 &&
		len(e.blessingNames) == 0 &&
		!e.ServesMountTable
}

// VersionedString returns a string in the specified format. If the version
// number is unsupported, the current 'default' version will be used.
func (e Endpoint) VersionedString(version int) string {
	// nologcall
	switch version {
	case 6:
		mt := "s"
		if e.ServesMountTable {
			mt = "m"
		}
		blessings := strings.Join(e.blessingNames, blessingsSeparator)
		escaped := make([]string, len(e.routes))
		for i := range e.routes {
			escaped[i] = Escape(e.routes[i], routeSeparator)
		}
		routes := strings.Join(escaped, routeSeparator)
		return fmt.Sprintf("@6@%s@%s@%s@%s@%s@%s@@",
			e.Protocol, Escape(e.Address, "@"), routes, e.RoutingID, mt, blessings)
	default:
		return e.VersionedString(DefaultEndpointVersion)
	}
}

func (e Endpoint) String() string {
	//nologcall
	return e.VersionedString(DefaultEndpointVersion)
}

// Name returns a string reprsentation of this Endpoint that can
// be used as a name with rpc.StartCall.
func (e Endpoint) Name() string {
	//nologcall
	return JoinAddressName(e.String(), "")
}

// Addr returns a net.Addr whose String method will return the
// the underlying network address encoded in the endpoint rather than
// the endpoint string itself.
// For example, for TCP based endpoints it will return a net.Addr
// whose network is "tcp" and string representation is <host>:<port>,
// than the full Vanadium endpoint as per the String method above.
func (e Endpoint) Addr() net.Addr {
	//nologcall
	return addr{network: e.Protocol, address: e.Address}
}

type addr struct {
	network, address string
}

// Network returns "v23" so that Endpoint can implement net.Addr.
func (a addr) Network() string {
	return a.network
}

// String returns a string representation of the endpoint.
//
// The String method formats the endpoint as:
//   @<version>@<version specific fields>@@
// Where version is an unsigned integer.
//
// Version 6 is the current version for RPC:
//   @6@<protocol>@<address>@<route>[,<route>]...@<routingid>@m|s@[<blessing>[,<blessing>]...]@@
//
// Along with Network, this method ensures that Endpoint implements net.Addr.
func (a addr) String() string {
	return a.address
}

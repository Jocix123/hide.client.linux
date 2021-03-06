package wireguard

import (
	"fmt"
	"github.com/eventure/hide.client.linux/rest"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"time"
)

type Config struct {
	Name					string				`yaml:"name,omitempty"`							// Interface name to use for the created WireGuard interface
	ListenPort				int					`yaml:"listenPort,omitempty"`					// Local UDP listen/bind port - 0 for automatic
	FirewallMark			int					`yaml:"firewallMark,omitempty"`					// Firewall mark for the traffic generated by the wireguard module
	PrivateKey				string				`yaml:"privateKey,omitempty"`					// Explicitly specified private key
	RoutingTable			int					`yaml:"routingTable,omitempty"`					// Routing table number to operate on when managing wireguard routes
	LeakProtection			bool				`yaml:"leakProtection,omitempty"`				// Enable or disable leak protection ( loopback routes )
	ResolvConfBackupFile	string				`yaml:"resolvConfBackupFile,omitempty"`			// Name of the resolv.conf backup file
	DpdTimeout				time.Duration		`yaml:"dpdTimeout,omitempty"`					// DPD timeout
	SplitTunnel				string				`yaml:"splitTunnel,omitempty"`					// A comma separated list of networks (CIDRs) for which to bypass the wireguard tunnel ( Split-Tunneling )
}

type Link struct {
	Config
	
	wireguardLink	netlink.Link
	mtu				int
	wgClient		*wgctrl.Client
	
	privateKey		wgtypes.Key
	presharedKey	*wgtypes.Key
	
	peer			wgtypes.PeerConfig
	
	routes			[]*netlink.Route																														// Unfortunately, the netlink library can't list routes in the routing tables ...
	gatewayRoutes	[]*netlink.Route																														// ... other than "main"
	loopbackRoutes	[]*netlink.Route
	
	dhcpRule		*netlink.Rule
	markRule		*netlink.Rule
	markRule6		*netlink.Rule
	splitRules		[]*netlink.Rule
	
	unwindStack		[]func() error
}

func NewLink( config Config) *Link { return &Link{Config: config} }
func (l *Link) PublicKey() wgtypes.Key { return l.privateKey.PublicKey() }

// Open the wireguard link, i.e. create or open an existing wireguard interface
func ( l *Link ) Open() ( err error ) {
	if err = l.handlePrivateKey(); err != nil { return }																									// Check the private key first
	if l.wgClient, err = wgctrl.New(); err != nil { fmt.Println( "Link: [ERR] Wireguard control client failed,", err ); return }							// Create a wireguard control client
	if err = l.ipLinkUp(); err != nil { return }																											// Bring the networking interface UP
	if err = l.wgLinkUp(); err != nil { return }																											// Configure the wireguard private key and listen port
	return
}

// Close the wireguard interface
func ( l *Link ) Close() { l.ipLinkDown() }

// Adds a wireguard peer and routes it
func ( l *Link ) Up( response *rest.ConnectResponse ) ( err error ) {
	// Avoid fragmentation if possible, set a small MTU
	// On IPv4, DS-Lite carrier connection takes MTU down as low as 1452 bytes
	// On IPv6, assume the lowest Internet IPv6 MTU of 1280 bytes
	// IPv4 header is 20 bytes, IPv6 header is 40 bytes and UDP header is 8 bytes
	// Wireguard overhead is 32 bytes
	if response.Endpoint.IP.To4() == nil { l.mtu = 1280 - 80 } else { l.mtu = 1452 - 60 }																	// Calculate MTU according to the carrier connection protocol
	if err = l.ipLinkSetMtu(); err != nil { return }																										// Set the wireguard interface MTU
	l.unwindStack = nil
	if err = l.wgAddPeer( response.PublicKey, response.PresharedKey, response.Endpoint, response.PersistentKeepaliveInterval ); err != nil { return }		// Add a wireguard peer
	l.unwindStack = append( l.unwindStack, l.wgRemovePeer )
	if err = l.ipAddrsAdd( response.AllowedIps ); err != nil { l.Down(); return }																			// Add the IP addresses to the wireguard device
	l.unwindStack = append( l.unwindStack, l.ipAddrsFlush )
	if err = l.ipRoutesAdd( response ); err != nil { l.Down(); return }																						// Add the default routes over the wireguard interface
	l.unwindStack = append( l.unwindStack, l.ipRoutesRemove )
	if err = l.dnsSet( response.DNS ); err != nil { l.Down(); return }																						// Set the DNS
	l.unwindStack = append( l.unwindStack, l.dnsRestore )
	fmt.Println( "Link: Up" )
	return
}

// Removes the wireguard peer and un-routes it
func ( l *Link ) Down() ( err error ) {
	rxBytes, txBytes, err := l.Acct()
	if err == nil { fmt.Println( "Link: Received", rxBytes, "bytes, transmitted", txBytes, "bytes" ) }
	n := len( l.unwindStack ) - 1
	for i := n; i >= 0; i-- { _ = l.unwindStack[i]() }
	fmt.Println( "Link: Down" )
	return
}
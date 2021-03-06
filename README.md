# Hide.me CLI VPN client for Linux

Hide.me CLI is a VPN client for use with eVenture Ltd. Hide.me VPN service based on the WireGuard protocol. Client's
features include:

* Completely standalone solution which does not depend on any external binaries or tools
* Key exchange via RESTful requests secured with TLS 1.3
* TLS certificate pinning of server's certificates to defeat man-in-the-middle sort of attacks
* Dead peer detection
* Leak protection a.k.a. kill-switch based on routing subsystem and packet marking
* Mobility/Roaming support
* DNS management
* IPv6 support
* SystemD notification support
* Split tunneling

TODO:
* Server lists and server chooser
* Automatic server selection
* Client certificate authentication/authorization

## Installation

You may clone this repository and run:

```go build -o hide.me```

Alternatively, download the latest build from the releases section.

## Hide.me WireGuard implementation details

WireGuard is one of the most secure and simplest VPN tunneling solutions in the industry. It is easy to set up and use as
long as no WireGuard public key exchange over an insecure medium (such as Internet) is required. Any sort of WireGuard
public key exchange is out of the scope of the WireGuard specification.

### Key exchange

The complicated task of public key exchange and secret key negotiation over an insecure medium is, usually, being handled
by:
* IKE protocol - a hard to understand, and a rather complicated part of IPSec
* TLS protocol - a foundation for HTTPS and virtually any other secure protocol

hide.me implementation of WireGuard leverages HTTPS (TLS) for the exchange of:
* WireGuard Public keys
* WireGuard Shared keys
* IP addressing information (IP addresses, DNS server addresses,gateways...)

Authentication for all operations requires the use of an Access-Token. An Access-Token is a just a binary blob which is
cryptographically tied to a hide.me account.

### Connection setup flow

Connection to a hide.me VPN server gets established in these steps:
1. hide.me CLI contacts a REST endpoint, over a secured channel, requesting a public key exchange and a server-side
connection setup
2. Server authenticates the request, sets up the connection and serves the IP addressing information (including
the WireGuard endpoint address). Server issues a randomized Session-Token which may be used to disconnect this
particular session
3. hide.me CLI sets up a WireGuard peer according to the server's instruction and starts the DPD check loop

### Leak protection

In contrast with many other solutions, hide.me CLI does not use any sort of Linux firewalling technology (IPTables, NFTables
or eBPF). Instead of relying on Linux'es IP filtering frameworks, hide.me CLI selectively routes traffic by setting up a
special routing table and a set of routing policy database rules. Blackhole routes in the aforementioned routing table drop
all traffic unless it meets one of the following conditions:
* Traffic is local ( loopback interfaces, local broadcasts and IPv6 link-local multicast )
* DHCPv4 traffic
* Traffic is explicitly allowed by the means of the Split-tunneling option
* Traffic is marked
* Traffic is about to be tunneled

This mode of operation makes it possible for the users to establish their own firewalling policies with which hide.me CLI
won't interfere.

## Usage
Usage instructions may be printed by running hide.me CLI without any parameters.
```
Usage:
  ./hide.me [options...] <command> [host]
...
```
### Commands
hide.me CLI user interface is quite simple. There are just three commands available:
```
command:
  token - request an Access-Token (required for connect)
  connect - connect to a vpn server
  conf - generate a configuration file to be used with the -c option
```
In order to connect to a VPN server an Access-Token must be requested from a VPN server. An Access-Token request is
issued by the **token** command.
An Access-Token issued by any server may be used, for authentication purposes, with any other hide.me VPN server.
When a server issues an Access-Token that token must be stored in a file. Default filename for an Access-Token is
"accessToken.txt".

Once an Access-Token is in place it may be used for **connect** requests. Stale access tokens get updated automatically.

hide.me CLI does not necessarily have to be invoked with a bunch of command line parameters. Instead, a YAML formatted
configuration file may be used to specify all the options. To generate such a configuration file the **conf** command
may be used.

Note that there are a few options which are configurable only through the configuration file. Such options are:
* Password - **DANGEROUS**, do not use this option unless you're aware of the security implications 
* ConnectTimeout
* AccessTokenUpdateDelay
```
host:
  fqdn, short name or an IP address of a hide.me server
  Required when the configuration file does not contain it
```
The hostname of a hide.me REST enpoint may be specified as a fully qualified domain name (nl.hide.me), short name (nl)
or an IP address. There's no guarantee that the REST endpoint will match a WireGuard endpoint.

## Options
```
  -b filename
    	resolv.conf backup filename (default "/etc/resolv.conf.backup.hide.me")
```
When applying the DNS servers to the system hide.me CLI will back up /etc/resolv.conf file and create a new file in its
place. Once the VPN session is over, DNS is restored by restoring the backup. 
```
  -c filename
    	Configuration filename
```
Use a configuration file named "filename".
```
  -ca string
    	CA certificate bundle (default "CA.pem")
```
During TLS negotiation the VPN server's certificate needs to be verified. This option makes it possible to specify
an alternate CA certificate bundle file.
```
  -d DNS servers
    	comma separated list of DNS servers used for client requests (default "209.250.251.37:53,217.182.206.81:53")
```
By default, Hide.me CLI uses hide.me operated DNS servers to resolve VPN server names when issuing token or connect
requests. The set of DNS servers used for these purposes may be customized with this option.
```
  -dpd duration
    	DPD timeout (default 1m0s)
```
In order to detect if a connection has stalled, usually due to networking issues, hide.me CLI periodically checks
the connection state. The checking period can be changed with this option, but can't be higher than a minute.
```
  -i interface
    	network interface name (default "vpn")
```
Use this option to specify the name of the networking interface to create or use.
```
  -l port
    	listen port
```
WireGuard kernel module may listen on a user-defined port for encrypted traffic.
```
  -m mark
    	firewall mark for wireguard and hide.me client originated traffic (default 55555)
```
Hide.me CLI takes advantage of the WireGuard module's ability to mark WireGuard traffic. Packet marks make it possible for
Linux to selectively route traffic according to RPDB policies.

Setting this option to any other value than 0 is supported, however setting it to 0 is not as such a setting turns off
the required functionality.
```
  -p port
    	remote port (default 432)
```
Remote REST endpoint port may be changed with this option.
```
  -r table
    	routing table to use (default 55555)
```
Set the routing table to use for general traffic and leak protection mechanism.
```
  -s networks
    	comma separated list of networks (CIDRs) for which to bypass the VPN
```
List of split-tunneled networks, i.e. the networks for which the traffic should not be tunneled over the VPN.
```
  -t string
    	access token filename (default "accessToken.txt")
```
Name of the file which contains an Access-Token.
```
  -u username
    	hide.me username
```
Set the hide.me username.

## Contributing

If you want to contribute to this project, please read the [contribution guide](https://github.com/eventure/hide.client.linux/blob/master/CONTRIBUTING.md).

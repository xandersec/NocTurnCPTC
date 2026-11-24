package main

// web_home.go backs the Home dashboard (who/where you are) and the NmapResponder
// import that drops recommended commands onto the current project's map.

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

func currentUser() map[string]string {
	m := map[string]string{"uid": "", "name": "", "username": ""}
	if u, err := user.Current(); err == nil {
		m["uid"] = u.Uid
		m["username"] = u.Username
		m["name"] = u.Name
	}
	m["root"] = "false"
	if os.Geteuid() == 0 {
		m["root"] = "true"
	}
	return m
}

func hostnameOr() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}

// interfaceAddrs lists up non-loopback interfaces and their IPv4 addresses,
// flagging tun/tap/ppp interfaces as the likely VPN link.
func interfaceAddrs() []map[string]any {
	var out []map[string]any
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 || ifc.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		var v4 []string
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				if ip4 := ipn.IP.To4(); ip4 != nil {
					v4 = append(v4, ip4.String())
				}
			}
		}
		if len(v4) == 0 {
			continue
		}
		name := ifc.Name
		vpn := strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap") || strings.HasPrefix(name, "ppp")
		out = append(out, map[string]any{"name": name, "addrs": v4, "vpn": vpn})
	}
	return out
}

// ── NmapResponder import ────────────────────────────────────────────────────

type nmapRespReport struct {
	Target string `json:"target"`
	Hosts  []struct {
		Address  string `json:"address"`
		Hostname string `json:"hostname"`
		Ports    []struct {
			Port     string   `json:"port"`
			Protocol string   `json:"protocol"`
			Service  string   `json:"service"`
			Banner   string   `json:"banner"`
			Note     string   `json:"note"`
			Commands []string `json:"commands"`
			Exploits []struct {
				EdbID string   `json:"edb_id"`
				Title string   `json:"title"`
				CVEs  []string `json:"cves"`
			} `json:"exploits"`
		} `json:"ports"`
	} `json:"hosts"`
}

// nmapResponderBin locates the nmapresponder binary.
func nmapResponderBin() string {
	if p := os.Getenv("PHANTOMDECK_NMAPRESPONDER"); p != "" {
		return p
	}
	if p, err := exec.LookPath("nmapresponder"); err == nil {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		cand := filepath.Join(home, "Projects", "NmapResponder", "nmapresponder")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return ""
}

// handleNmapImport accepts either raw nmap -oN text or NmapResponder JSON and
// appends the recommended commands to the current project as "recommended"
// entries (the attack vectors shown on the map).
func handleNmapImport(w http.ResponseWriter, r *http.Request, st *store) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		http.Error(w, "empty body — paste nmap .nmap output or NmapResponder JSON", http.StatusBadRequest)
		return
	}

	var rep nmapRespReport
	if trimmed[0] == '{' {
		// Already NmapResponder JSON.
		if err := json.Unmarshal(trimmed, &rep); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		// Raw nmap text — shell out to nmapresponder -json.
		bin := nmapResponderBin()
		if bin == "" {
			http.Error(w, "nmapresponder binary not found (build ~/Projects/NmapResponder or set PHANTOMDECK_NMAPRESPONDER)", http.StatusServiceUnavailable)
			return
		}
		args := []string{"-json"}
		if r.URL.Query().Get("exploits") != "1" {
			args = append(args, "-no-exploits")
		}
		if t := r.URL.Query().Get("target"); t != "" {
			args = append(args, "-t", t)
		}
		cmd := exec.Command(bin, args...)
		cmd.Stdin = bytes.NewReader(body)
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		if err := cmd.Run(); err != nil {
			msg := strings.TrimSpace(errBuf.String())
			if msg == "" {
				msg = err.Error()
			}
			http.Error(w, "nmapresponder: "+msg, http.StatusBadGateway)
			return
		}
		if err := json.Unmarshal(outBuf.Bytes(), &rep); err != nil {
			http.Error(w, "nmapresponder output parse: "+err.Error(), http.StatusBadGateway)
			return
		}
	}

	var entries []histEntry
	for _, h := range rep.Hosts {
		host := h.Address
		if host == "" {
			host = rep.Target
		}
		for _, p := range h.Ports {
			for _, c := range p.Commands {
				entries = append(entries, histEntry{
					Kind: "recommended", Command: c,
					Tool: p.Service, Host: host, Port: p.Port,
					Service: p.Service, Note: p.Note,
				})
			}
			for _, ex := range p.Exploits {
				note := ex.Title
				if len(ex.CVEs) > 0 {
					note += " (" + strings.Join(ex.CVEs, ", ") + ")"
				}
				entries = append(entries, histEntry{
					Kind: "recommended", Command: "searchsploit -p " + ex.EdbID,
					Tool: "exploit", Host: host, Port: p.Port,
					Service: "exploit", Note: note,
				})
			}
		}
	}

	hist, err := st.Append(entries...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"imported": len(entries), "entries": hist, "canvas": st.canvasPath(st.Current())})
}

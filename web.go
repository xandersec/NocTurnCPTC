package main

// web.go restores PhantomDeck's `--web` browser interface: a small, local-only
// HTTP server that serves a single-page GUI mirroring the TUI. You pick a
// catalog and tool, toggle/fill flags, and the assembled command is built by
// the *same* Tool.Command() logic the TUI uses, then copied from the browser.
//
// It also adds a VPN panel: upload an OpenVPN (.ovpn) profile and connect /
// disconnect. Connecting actually runs `openvpn` (this is the one place
// PhantomDeck executes something), so it is gated to a localhost listener.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// runWeb builds the catalogs and serves the GUI until the process is killed.
func runWeb(addr string) error {
	modes := catalogModes()

	mux := http.NewServeMux()
	v := &vpnManager{}
	st, err := newStore()
	if err != nil {
		return fmt.Errorf("init project store: %w", err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, webIndexHTML)
	})

	// /api/catalogs — the full catalog/tool/flag tree as JSON. Tools are cloned
	// so the pristine catalog is never mutated.
	mux.HandleFunc("/api/catalogs", func(w http.ResponseWriter, r *http.Request) {
		type outTool struct {
			Name     string `json:"Name"`
			Binary   string `json:"Binary"`
			Desc     string `json:"Desc"`
			Category string `json:"Category"`
			Flags    []Flag `json:"Flags"`
		}
		type outCat struct {
			Key   string    `json:"Key"`
			Label string    `json:"Label"`
			Tools []outTool `json:"Tools"`
		}
		var out []outCat
		for _, md := range modes {
			oc := outCat{Key: md.Key, Label: md.Label}
			for _, t := range md.tools() {
				c := t.clone()
				oc.Tools = append(oc.Tools, outTool{
					Name: c.Name, Binary: c.Binary, Desc: c.Desc,
					Category: c.Category, Flags: c.Flags,
				})
			}
			out = append(out, oc)
		}
		writeJSON(w, out)
	})

	// /api/command — accepts an edited Tool (flag values/toggles) and returns the
	// assembled command plus any still-missing required flags. Reuses the exact
	// TUI assembly logic so the browser and TUI never disagree.
	mux.HandleFunc("/api/command", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var t Tool
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{
			"command": t.Command(),
			"missing": t.MissingRequired(),
		})
	})

	// VPN endpoints.
	mux.HandleFunc("/api/vpn/upload", v.handleUpload)
	mux.HandleFunc("/api/vpn/connect", v.handleConnect)
	mux.HandleFunc("/api/vpn/disconnect", v.handleDisconnect)
	mux.HandleFunc("/api/vpn/status", v.handleStatus)

	// Home: identity + VPN + current project.
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		v.mu.Lock()
		vpn := v.statusLocked()
		v.mu.Unlock()
		writeJSON(w, map[string]any{
			"user":       currentUser(),
			"hostname":   hostnameOr(),
			"interfaces": interfaceAddrs(),
			"vpn":        vpn,
			"project":    st.Current(),
			"projects":   st.Projects(),
		})
	})

	// Projects.
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			cur, err := st.Create(body.Name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, map[string]any{"current": cur, "projects": st.Projects()})
			return
		}
		writeJSON(w, map[string]any{"current": st.Current(), "projects": st.Projects()})
	})
	mux.HandleFunc("/api/projects/select", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		cur, err := st.Select(body.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"current": cur, "projects": st.Projects()})
	})

	// History / map.
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var e histEntry
			if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			e.Kind = "run"
			h, err := st.Append(e)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"entries": h, "canvas": st.canvasPath(st.Current())})
		default:
			writeJSON(w, map[string]any{"entries": st.History(), "canvas": st.canvasPath(st.Current())})
		}
	})
	mux.HandleFunc("/api/history/clear", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Clear(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"entries": []histEntry{}, "canvas": st.canvasPath(st.Current())})
	})

	// Download the generated Obsidian .canvas for the current project.
	mux.HandleFunc("/api/canvas", func(w http.ResponseWriter, r *http.Request) {
		p := st.canvasPath(st.Current())
		b, err := os.ReadFile(p)
		if err != nil {
			http.Error(w, "no canvas yet", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(p)))
		w.Write(b)
	})

	// Import NmapResponder recommendations onto the current project's map.
	mux.HandleFunc("/api/import/nmap", func(w http.ResponseWriter, r *http.Request) {
		handleNmapImport(w, r, st)
	})

	// Omarchy themes (read-only): list + palette for live restyling.
	mux.HandleFunc("/api/themes", handleThemes)
	mux.HandleFunc("/api/theme", handleTheme)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	fmt.Printf("PhantomDeck web GUI: http://%s\n", ln.Addr().String())
	fmt.Println("Press ctrl+c to stop.")
	srv := &http.Server{Handler: mux}
	return srv.Serve(ln)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ── VPN management ──────────────────────────────────────────────────────────

type vpnManager struct {
	mu      sync.Mutex
	profile string       // path to the saved .ovpn
	name    string       // original filename
	cmd     *exec.Cmd    // running openvpn, or nil
	log     bytes.Buffer // combined stdout/stderr
	started time.Time
}

func (v *vpnManager) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil { // 8 MiB cap
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, hdr, err := r.FormFile("profile")
	if err != nil {
		http.Error(w, "no profile file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	dir := filepath.Join(os.TempDir(), "phantomdeck-vpn")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Keep the original basename (sanitised) so openvpn's relative references
	// (ca/cert paths next to it, if any) resolve predictably.
	base := filepath.Base(hdr.Filename)
	base = strings.ReplaceAll(base, "..", "")
	if base == "" || base == "." || base == "/" {
		base = "profile.ovpn"
	}
	dst := filepath.Join(dir, base)
	out, err := os.Create(dst)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out.Close()

	v.mu.Lock()
	v.profile = dst
	v.name = base
	v.mu.Unlock()

	writeJSON(w, v.statusLocked())
}

func (v *vpnManager) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.cmd != nil && v.cmd.Process != nil {
		http.Error(w, "already connected (disconnect first)", http.StatusConflict)
		return
	}
	if v.profile == "" {
		http.Error(w, "upload a .ovpn profile first", http.StatusBadRequest)
		return
	}

	// openvpn needs root to configure the tun device and routes. If we are not
	// already root, prefix with a non-interactive sudo — this works on the Kali
	// pentest boxes PhantomDeck targets (passwordless sudo / running as root)
	// and fails loudly in the log otherwise instead of hanging on a prompt.
	var name string
	var args []string
	if os.Geteuid() == 0 {
		name, args = "openvpn", []string{"--config", v.profile}
	} else {
		name, args = "sudo", []string{"-n", "openvpn", "--config", v.profile}
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = filepath.Dir(v.profile)
	v.log.Reset()
	fmt.Fprintf(&v.log, "$ %s %s\n", name, strings.Join(args, " "))
	cmd.Stdout = &v.log
	cmd.Stderr = &v.log

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(&v.log, "failed to start: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v.cmd = cmd
	v.started = time.Now()

	// Reap the process when it exits so status flips back to disconnected.
	go func(c *exec.Cmd) {
		err := c.Wait()
		v.mu.Lock()
		if err != nil {
			fmt.Fprintf(&v.log, "\n[openvpn exited: %v]\n", err)
		} else {
			fmt.Fprintf(&v.log, "\n[openvpn exited]\n")
		}
		if v.cmd == c {
			v.cmd = nil
		}
		v.mu.Unlock()
	}(cmd)

	writeJSON(w, v.statusLocked())
}

func (v *vpnManager) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.cmd == nil || v.cmd.Process == nil {
		writeJSON(w, v.statusLocked())
		return
	}
	// A sudo'd openvpn runs as root, so an unprivileged kill won't reach it;
	// try a normal kill, then fall back to `sudo kill` for that case.
	proc := v.cmd.Process
	if err := proc.Kill(); err != nil && os.Geteuid() != 0 {
		_ = exec.Command("sudo", "-n", "kill", fmt.Sprintf("%d", proc.Pid)).Run()
	}
	fmt.Fprintf(&v.log, "\n[disconnect requested]\n")
	writeJSON(w, v.statusLocked())
}

func (v *vpnManager) handleStatus(w http.ResponseWriter, r *http.Request) {
	v.mu.Lock()
	defer v.mu.Unlock()
	writeJSON(w, v.statusLocked())
}

// statusLocked builds the status payload; caller must hold v.mu.
func (v *vpnManager) statusLocked() map[string]any {
	connected := v.cmd != nil && v.cmd.Process != nil
	m := map[string]any{
		"profile":   v.name,
		"connected": connected,
		"log":       v.log.String(),
	}
	if connected {
		m["uptime"] = int(time.Since(v.started).Seconds())
	}
	return m
}

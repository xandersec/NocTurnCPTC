package main

// web_store.go gives the web GUI a per-project engagement store: each project
// keeps a command history on disk and a generated Obsidian canvas (.canvas)
// that visualises the attack path (commands you ran, chained in order) plus the
// recommended "attack vectors" imported from NmapResponder.
//
// Layout on disk:
//   ~/.local/share/phantomdeck/
//     current.txt                      # name of the active project
//     projects/<name>/history.json     # []histEntry
//     projects/<name>/<name>.canvas    # Obsidian canvas, regenerated on change

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// histEntry is one command on the map: either a command you assembled/ran, or a
// recommendation imported from a scan.
type histEntry struct {
	Time    string `json:"time"`
	Kind    string `json:"kind"` // "run" or "recommended"
	Catalog string `json:"catalog,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Command string `json:"command"`
	Host    string `json:"host,omitempty"`
	Port    string `json:"port,omitempty"`
	Service string `json:"service,omitempty"`
	Note    string `json:"note,omitempty"`
}

type store struct {
	mu      sync.Mutex
	base    string
	current string
}

var safeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeProject(name string) string {
	name = strings.TrimSpace(name)
	name = safeName.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-._")
	if name == "" {
		name = "default"
	}
	return name
}

// newStore prepares the base dir and selects a current project (restoring the
// last one, else "default").
func newStore() (*store, error) {
	// PHANTOMDECK_DATA lets a container (or anyone) relocate the engagement
	// store onto a mounted volume so projects/canvas survive restarts.
	base := os.Getenv("PHANTOMDECK_DATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		base = filepath.Join(home, ".local", "share", "phantomdeck")
	}
	if err := os.MkdirAll(filepath.Join(base, "projects"), 0o755); err != nil {
		return nil, err
	}
	s := &store{base: base}

	cur := "default"
	if b, err := os.ReadFile(filepath.Join(base, "current.txt")); err == nil {
		if v := sanitizeProject(string(b)); v != "" {
			cur = v
		}
	}
	s.current = cur
	if err := s.ensureProject(cur); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *store) projectDir(name string) string {
	return filepath.Join(s.base, "projects", sanitizeProject(name))
}
func (s *store) historyPath(name string) string {
	return filepath.Join(s.projectDir(name), "history.json")
}
func (s *store) canvasPath(name string) string {
	n := sanitizeProject(name)
	return filepath.Join(s.projectDir(n), n+".canvas")
}

func (s *store) ensureProject(name string) error {
	dir := s.projectDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	hp := s.historyPath(name)
	if _, err := os.Stat(hp); os.IsNotExist(err) {
		if err := os.WriteFile(hp, []byte("[]"), 0o644); err != nil {
			return err
		}
		s.writeCanvas(name, nil)
	}
	return nil
}

func (s *store) Projects() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, _ := os.ReadDir(filepath.Join(s.base, "projects"))
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		out = []string{"default"}
	}
	return out
}

func (s *store) Current() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *store) Create(name string) (string, error) {
	n := sanitizeProject(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureProject(n); err != nil {
		return "", err
	}
	s.current = n
	os.WriteFile(filepath.Join(s.base, "current.txt"), []byte(n), 0o644)
	return n, nil
}

func (s *store) Select(name string) (string, error) {
	n := sanitizeProject(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureProject(n); err != nil {
		return "", err
	}
	s.current = n
	os.WriteFile(filepath.Join(s.base, "current.txt"), []byte(n), 0o644)
	return n, nil
}

func (s *store) loadHistory(name string) []histEntry {
	var h []histEntry
	b, err := os.ReadFile(s.historyPath(name))
	if err != nil {
		return h
	}
	json.Unmarshal(b, &h)
	return h
}

func (s *store) History() []histEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadHistory(s.current)
}

func (s *store) saveHistory(name string, h []histEntry) error {
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.historyPath(name), b, 0o644); err != nil {
		return err
	}
	s.writeCanvas(name, h)
	return nil
}

// Append adds entries to the current project (dedup'ing identical commands that
// are already present) and regenerates the canvas. Returns the new history.
func (s *store) Append(entries ...histEntry) ([]histEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.loadHistory(s.current)
	seen := map[string]bool{}
	for _, e := range h {
		seen[e.Kind+"|"+e.Command] = true
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, e := range entries {
		if e.Command == "" {
			continue
		}
		if e.Kind == "" {
			e.Kind = "run"
		}
		if e.Time == "" {
			e.Time = now
		}
		key := e.Kind + "|" + e.Command
		if seen[key] {
			continue
		}
		seen[key] = true
		h = append(h, e)
	}
	if err := s.saveHistory(s.current, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (s *store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveHistory(s.current, nil)
}

// ── Obsidian canvas generation ──────────────────────────────────────────────

type canvasNode struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Text   string `json:"text"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Color  string `json:"color,omitempty"`
}

type canvasEdge struct {
	ID       string `json:"id"`
	FromNode string `json:"fromNode"`
	FromSide string `json:"fromSide"`
	ToNode   string `json:"toNode"`
	ToSide   string `json:"toSide"`
}

type canvasFile struct {
	Nodes []canvasNode `json:"nodes"`
	Edges []canvasEdge `json:"edges"`
}

// writeCanvas regenerates <name>.canvas from the history. Caller holds s.mu.
//
// Two columns: the RUN column (right) chains the commands you actually ran in
// order — the path taken; the RECOMMENDED column (left) groups imported attack
// vectors under a per-host hub node, fanning out to each suggested command.
func (s *store) writeCanvas(name string, h []histEntry) {
	var cf canvasFile
	id := 0
	nid := func() string { id++; return fmt.Sprintf("n%d", id) }
	eid := 0
	eidf := func() string { eid++; return fmt.Sprintf("e%d", eid) }

	// Title.
	title := canvasNode{ID: nid(), Type: "text", X: 240, Y: -220, Width: 400, Height: 80, Color: "6",
		Text: fmt.Sprintf("# %s\nPhantomDeck engagement map", name)}
	cf.Nodes = append(cf.Nodes, title)

	const (
		recX = -120
		runX = 640
		colW = 460
	)

	// RUN column — the path taken.
	runY := 0
	var prevRun string
	for _, e := range h {
		if e.Kind == "recommended" {
			continue
		}
		n := canvasNode{ID: nid(), Type: "text", X: runX, Y: runY, Width: colW, Height: 130, Color: "4",
			Text: runNodeText(e)}
		cf.Nodes = append(cf.Nodes, n)
		if prevRun != "" {
			cf.Edges = append(cf.Edges, canvasEdge{ID: eidf(), FromNode: prevRun, FromSide: "bottom", ToNode: n.ID, ToSide: "top"})
		} else {
			cf.Edges = append(cf.Edges, canvasEdge{ID: eidf(), FromNode: title.ID, FromSide: "bottom", ToNode: n.ID, ToSide: "top"})
		}
		prevRun = n.ID
		runY += 170
	}

	// RECOMMENDED column — grouped by host hub.
	recY := 0
	hubOrder := []string{}
	hubID := map[string]string{}
	for _, e := range h {
		if e.Kind != "recommended" {
			continue
		}
		hubKey := e.Host
		if hubKey == "" {
			hubKey = "recommendations"
		}
		hub, ok := hubID[hubKey]
		if !ok {
			hubNode := canvasNode{ID: nid(), Type: "text", X: recX - colW - 80, Y: recY, Width: 240, Height: 80, Color: "1",
				Text: "## " + hubKey + "\nattack vectors"}
			cf.Nodes = append(cf.Nodes, hubNode)
			hub = hubNode.ID
			hubID[hubKey] = hub
			hubOrder = append(hubOrder, hubKey)
		}
		color := "5"
		if strings.Contains(strings.ToLower(e.Note), "cve") || e.Service == "exploit" {
			color = "2"
		}
		n := canvasNode{ID: nid(), Type: "text", X: recX, Y: recY, Width: colW, Height: 120, Color: color,
			Text: recNodeText(e)}
		cf.Nodes = append(cf.Nodes, n)
		cf.Edges = append(cf.Edges, canvasEdge{ID: eidf(), FromNode: hub, FromSide: "right", ToNode: n.ID, ToSide: "left"})
		recY += 150
	}
	_ = hubOrder

	b, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(s.canvasPath(name), b, 0o644)
}

func runNodeText(e histEntry) string {
	head := e.Tool
	if head == "" {
		head = "command"
	}
	if e.Catalog != "" {
		head += " · " + e.Catalog
	}
	return fmt.Sprintf("**%s**\n```sh\n%s\n```\n_%s_", head, e.Command, e.Time)
}

func recNodeText(e histEntry) string {
	loc := strings.TrimSpace(strings.Join([]string{e.Service, hostPort(e)}, " "))
	if loc == "" {
		loc = "recommended"
	}
	s := fmt.Sprintf("**%s**\n```sh\n%s\n```", loc, e.Command)
	if e.Note != "" {
		s += "\n" + e.Note
	}
	return s
}

func hostPort(e histEntry) string {
	if e.Host == "" {
		return ""
	}
	if e.Port != "" {
		return e.Host + ":" + e.Port
	}
	return e.Host
}

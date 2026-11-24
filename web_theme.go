package main

// web_theme.go lets the web GUI adopt the user's Omarchy themes. It reads each
// theme's palette (colors.toml when present, else alacritty.toml) and the
// currently-active theme, and serves them so the page can restyle itself. It
// only reads Omarchy's files — it never changes the desktop theme.

import (
	"bufio"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func omarchyThemesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "omarchy", "themes")
}

func omarchyCurrentThemeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "omarchy", "current", "theme")
}

func omarchyCurrentThemeName() string {
	home, _ := os.UserHomeDir()
	b, err := os.ReadFile(filepath.Join(home, ".local", "state", "omarchy", "current", "theme.name"))
	if err == nil {
		if n := strings.TrimSpace(string(b)); n != "" {
			return n
		}
	}
	return ""
}

func listOmarchyThemes() []string {
	entries, _ := os.ReadDir(omarchyThemesDir())
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// parseKeyVals reads simple `key = "#hex"` / `key = '#hex'` lines, ignoring
// section headers and comments. Good enough for Omarchy's colors.toml.
func parseKeyVals(path string) map[string]string {
	m := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return m
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if k != "" && v != "" {
			m[k] = v
		}
	}
	return m
}

// parseAlacrittyColors reads an alacritty.toml [colors.*] palette into the same
// flat color0..color15 / background / foreground shape as colors.toml.
func parseAlacrittyColors(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	section := ""
	normalIdx := map[string]string{"black": "0", "red": "1", "green": "2", "yellow": "3", "blue": "4", "magenta": "5", "cyan": "6", "white": "7"}
	brightIdx := map[string]string{"black": "8", "red": "9", "green": "10", "yellow": "11", "blue": "12", "magenta": "13", "cyan": "14", "white": "15"}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			section = strings.Trim(line, "[]")
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if v == "" {
			continue
		}
		switch section {
		case "colors.primary":
			if k == "background" || k == "foreground" {
				out[k] = v
			}
		case "colors.normal":
			if idx, ok := normalIdx[k]; ok {
				out["color"+idx] = v
			}
		case "colors.bright":
			if idx, ok := brightIdx[k]; ok {
				out["color"+idx] = v
			}
		}
	}
	return out
}

// themePalette returns a normalized palette for a theme name (or the current
// theme when name is empty).
func themePalette(name string) map[string]any {
	var dir string
	if name == "" || name == omarchyCurrentThemeName() {
		dir = omarchyCurrentThemeDir()
		if name == "" {
			name = omarchyCurrentThemeName()
		}
	} else {
		dir = filepath.Join(omarchyThemesDir(), name)
	}

	raw := parseKeyVals(filepath.Join(dir, "colors.toml"))
	if _, ok := raw["background"]; !ok {
		// Fall back to alacritty.toml for themes without a generated colors.toml.
		raw = parseAlacrittyColors(filepath.Join(dir, "alacritty.toml"))
	}

	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok && v != "" {
				return v
			}
		}
		return ""
	}

	bg := get("background", "color0")
	fg := get("foreground", "color7")
	accent := get("accent", "color5", "color4")

	pal := map[string]any{
		"name":       name,
		"bg":         bg,
		"fg":         fg,
		"accent":     accent,
		"selection":  get("selection", "color8"),
		"red":        get("color1"),
		"green":      get("color2"),
		"yellow":     get("color3", "color11"),
		"blue":       get("color4"),
		"magenta":    get("color5"),
		"cyan":       get("color6"),
		"brightBlue": get("color12", "color4"),
	}
	return pal
}

func handleThemes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"current": omarchyCurrentThemeName(),
		"themes":  listOmarchyThemes(),
	})
}

func handleTheme(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, themePalette(r.URL.Query().Get("name")))
}

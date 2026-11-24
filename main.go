// Command phantomdeck is an interactive terminal UI for assembling pentest
// tool command lines. Pick a tool, toggle and fill in flags, watch the command
// build live, then yank it to the clipboard (ctrl+y) and run it in a separate
// terminal.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	web := flag.Bool("web", false, "serve the browser GUI instead of the terminal UI")
	addr := flag.String("web-addr", "127.0.0.1:8777", "address for --web to listen on")
	flag.Parse()

	// --web launches the browser interface (see web.go) and never returns until
	// the server is stopped. Everything else starts the terminal UI below.
	if *web {
		if err := runWeb(*addr); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	res, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	// Print the last assembled command so it survives leaving the alt screen
	// (copy with ctrl+y inside the TUI, or print-and-quit with ctrl+o).
	if m, ok := res.(model); ok && strings.TrimSpace(m.finalCmd) != "" {
		fmt.Println(m.finalCmd)
	}
}

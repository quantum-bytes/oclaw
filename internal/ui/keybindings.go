package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Send          key.Binding
	Newline       key.Binding
	Abort         key.Binding
	Quit          key.Binding
	AgentPicker   key.Binding
	SessionPicker key.Binding
	NewSession    key.Binding
	ToggleThink   key.Binding
	Help          key.Binding
	ScrollUp      key.Binding
	ScrollDown    key.Binding
	ScrollTop     key.Binding
	ScrollBottom  key.Binding
	Escape        key.Binding
}

var keys = keyMap{
	Send: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send message"),
	),
	Newline: key.NewBinding(
		key.WithKeys("shift+enter"),
		key.WithHelp("shift+enter", "new line"),
	),
	Abort: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "abort/quit"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "quit"),
	),
	AgentPicker: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "switch agent"),
	),
	SessionPicker: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "sessions"),
	),
	NewSession: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new session"),
	),
	ToggleThink: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "toggle thinking"),
	),
	Help: key.NewBinding(
		key.WithKeys("ctrl+/"),
		key.WithHelp("ctrl+/", "help"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "scroll up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "scroll down"),
	),
	ScrollTop: key.NewBinding(
		key.WithKeys("home"),
		key.WithHelp("home", "scroll to top"),
	),
	ScrollBottom: key.NewBinding(
		key.WithKeys("end"),
		key.WithHelp("end", "scroll to bottom"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close/cancel"),
	),
}

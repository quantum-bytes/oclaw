package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/quantum-bytes/oclaw/internal/config"
	"github.com/quantum-bytes/oclaw/internal/ui"
)

var (
	flagURL   string
	flagToken string
	flagAgent string
)

var rootCmd = &cobra.Command{
	Use:   "oclaw",
	Short: "Terminal UI for OpenClaw — interactive agent chat",
	Long: `oclaw is an interactive terminal UI for OpenClaw that provides
streaming chat, session persistence, and multi-agent switching.

It connects directly to the OpenClaw gateway via WebSocket,
bypassing the buggy built-in TUI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(flagURL, flagToken, flagAgent)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.AgentID == "" {
			return fmt.Errorf("no agent configured — use --agent flag or set agents.default in openclaw.json")
		}

		app := ui.NewApp(cfg)
		p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "Gateway WebSocket URL (default: from config or ws://127.0.0.1:39421)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Gateway auth token (default: from config)")
	rootCmd.PersistentFlags().StringVar(&flagAgent, "agent", "", "Default agent ID to connect to")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

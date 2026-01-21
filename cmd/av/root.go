package main

import (
	"fmt"
	"os"

	"github.com/buddyh/av/internal/output"
	"github.com/buddyh/av/internal/process"
	"github.com/buddyh/av/internal/tmux"
	"github.com/buddyh/av/internal/version"
	"github.com/spf13/cobra"
)

var Version = "dev"

type rootFlags struct {
	json    bool
	plain   bool
	noColor bool
	noFetch bool
}

func execute(args []string) error {
	flags := &rootFlags{}
	out := output.New(os.Stdout, os.Stderr)

	rootCmd := &cobra.Command{
		Use:           "av",
		Short:         "Agent Versions - Monitor and manage AI coding agents",
		Long:          `Monitor installed and running versions of Claude Code and OpenAI Codex.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			out.Configure(flags.json, flags.plain, flags.noColor)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(out, flags)
		},
	}

	rootCmd.PersistentFlags().BoolVar(&flags.json, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flags.plain, "plain", false, "Plain output (no colors/symbols)")
	rootCmd.PersistentFlags().BoolVar(&flags.noColor, "no-color", false, "Disable colors")
	rootCmd.PersistentFlags().BoolVar(&flags.noFetch, "no-fetch", false, "Skip fetching latest versions")

	rootCmd.AddCommand(newRestartCmd(flags, out))
	rootCmd.AddCommand(newCheckCmd(flags, out))

	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		out.Error(err)
		return err
	}
	return nil
}

func runStatus(out *output.Output, flags *rootFlags) error {
	// Get installed versions
	claudeInstalled := version.GetInstalledClaude()
	codexInstalled := version.GetInstalledCodex()

	// Fetch latest versions (unless --no-fetch)
	var claudeLatest, codexLatest string
	if !flags.noFetch {
		claudeLatest = version.FetchLatestClaude()
		codexLatest = version.FetchLatestCodex()
	}

	// Find running sessions
	sessions := process.FindAgentSessions()

	// Enrich with tmux info
	tmuxPanes := tmux.GetPanes()
	process.EnrichWithTmux(sessions, tmuxPanes)

	// Output
	if flags.json {
		return out.JSON(map[string]any{
			"installed": map[string]string{
				"claude": claudeInstalled,
				"codex":  codexInstalled,
			},
			"latest": map[string]string{
				"claude": claudeLatest,
				"codex":  codexLatest,
			},
			"sessions": sessions,
		})
	}

	out.PrintHeader("Installed Versions")
	out.PrintVersion("Claude Code", claudeInstalled, claudeLatest)
	out.PrintVersion("Codex", codexInstalled, codexLatest)
	fmt.Println()

	out.PrintHeader("Running Sessions")
	needsRestart := out.PrintSessions(sessions, claudeInstalled, codexInstalled)

	if needsRestart > 0 {
		fmt.Printf("\n%d session(s) need restart. Run `av restart` to update them.\n", needsRestart)
	}

	return nil
}

func newCheckCmd(flags *rootFlags, out *output.Output) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check for updates (no process scan)",
		RunE: func(cmd *cobra.Command, args []string) error {
			claudeInstalled := version.GetInstalledClaude()
			codexInstalled := version.GetInstalledCodex()
			claudeLatest := version.FetchLatestClaude()
			codexLatest := version.FetchLatestCodex()

			if flags.json {
				return out.JSON(map[string]any{
					"installed": map[string]string{"claude": claudeInstalled, "codex": codexInstalled},
					"latest":    map[string]string{"claude": claudeLatest, "codex": codexLatest},
					"claude_update_available": claudeLatest != "" && claudeInstalled != claudeLatest,
					"codex_update_available":  codexLatest != "" && codexInstalled != codexLatest,
				})
			}

			out.PrintVersion("Claude Code", claudeInstalled, claudeLatest)
			out.PrintVersion("Codex", codexInstalled, codexLatest)
			return nil
		},
	}
}

func newRestartCmd(flags *rootFlags, out *output.Output) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart outdated sessions (tmux only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			claudeInstalled := version.GetInstalledClaude()
			codexInstalled := version.GetInstalledCodex()

			sessions := process.FindAgentSessions()
			tmuxPanes := tmux.GetPanes()
			process.EnrichWithTmux(sessions, tmuxPanes)

			var toRestart []*process.Session
			for _, s := range sessions {
				if s.TmuxSession == "" {
					continue // Can't restart non-tmux
				}
				currentVersion := claudeInstalled
				if s.Agent == "codex" {
					currentVersion = codexInstalled
				}
				if all || s.RunningVersion != currentVersion {
					toRestart = append(toRestart, s)
				}
			}

			if len(toRestart) == 0 {
				out.Success("All sessions are up to date")
				return nil
			}

			out.Info(fmt.Sprintf("Restarting %d session(s)...", len(toRestart)))

			for _, s := range toRestart {
				if err := tmux.RestartSession(s.TmuxSession, s.Agent); err != nil {
					out.Warn(fmt.Sprintf("Failed to restart %s: %v", s.TmuxSession, err))
				} else {
					out.Success(fmt.Sprintf("Restarted %s", s.TmuxSession))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Restart all sessions, even if current")
	return cmd
}

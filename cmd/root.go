package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/longkey1/lnkr/internal/version"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lnkr",
	Short: "Sync files to a remote directory and keep links in place",
	Long: `lnkr manages files that live in a remote directory (e.g. cloud-synced
storage) while staying accessible from a local project directory via links.

Typical workflow:
  lnkr init --remote <path>   set up the project (.lnkr.toml)
  lnkr add <path>             move a file to remote and link it back
  lnkr status                 show the state of all links
  lnkr link                   re-create links (e.g. after cloning)
  lnkr unlink                 remove the links (entries and remote files kept)
  lnkr remove <path>          restore a file from remote back to local
  lnkr clean                  remove .lnkr.toml and its git exclude entries`,
	Version:       version.GetVersion(),
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Initialize global configuration (viper)
	lnkr.InitGlobalConfig()
}

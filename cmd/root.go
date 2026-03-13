package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spluca/mikrom-cli/internal/api"
	"github.com/spluca/mikrom-cli/internal/config"
)

var (
	cfg     *config.Config
	apiURL  string
	token   string
)

var rootCmd = &cobra.Command{
	Use:   "mikrom",
	Short: "CLI for the Mikrom API",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "Mikrom API URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Authentication token (overrides config)")

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(vmCmd)
	rootCmd.AddCommand(ippoolCmd)
}

func initConfig() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	if apiURL != "" {
		cfg.APIURL = apiURL
	}
	if token != "" {
		cfg.Token = token
	}
}

func newClient() *api.Client {
	return api.NewClient(cfg.APIURL, cfg.Token)
}

func requireAuth() {
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "not authenticated — run: mikrom auth login")
		os.Exit(1)
	}
}

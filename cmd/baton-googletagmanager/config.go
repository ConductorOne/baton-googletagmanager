package main

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/spf13/cobra"
)

// config defines the external configuration required for the connector to run.
type config struct {
	cli.BaseConfig `mapstructure:",squash"` // Puts the base config options in the same place as the connector options

	CredentialsJSONFilePath string   `mapstructure:"credentials-json-file-path"`
	Accounts                []string `mapstructure:"accounts"`
}

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *config) error {
	if cfg.CredentialsJSONFilePath == "" {
		return fmt.Errorf("path to credentials JSON file is required, use --help for more information")
	}

	return nil
}

// cmdFlags sets the cmdFlags required for the connector.
func cmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(
		"credentials-json-file-path",
		"",
		"Path to the credentials JSON file for the service account to use for authentication with Google Tag Manager ($BATON_CREDENTIALS_JSON_FILE_PATH)",
	)
	cmd.PersistentFlags().StringSlice("accounts", []string{}, "Limit syncing to the specified accounts ($BATON_ACCOUNTS)")
}

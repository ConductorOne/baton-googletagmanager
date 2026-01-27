package config

import (
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	CredentialsJSONFilePath = field.StringField(
		"credentials-json-file-path",
		field.WithDescription("Path to the credentials JSON file for the service account to use for authentication with Google Tag Manager"),
		field.WithRequired(true),
		field.WithDisplayName("Credentials JSON File Path"),
	)
	Accounts = field.StringSliceField(
		"accounts",
		field.WithDescription("Limit syncing to the specified accounts"),
		field.WithDisplayName("Accounts"),
	)

	// FieldRelationships defines relationships between the fields listed in
	// Config that can be automatically validated.
	FieldRelationships = []field.SchemaFieldRelationship{}
)

//go:generate go run ./gen
var Config = field.NewConfiguration([]field.SchemaField{
	CredentialsJSONFilePath,
	Accounts,
})

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid.
func ValidateConfig(cfg *Googletagmanager) error {
	if cfg.CredentialsJsonFilePath == "" {
		return fmt.Errorf("path to credentials JSON file is required, use --help for more information")
	}

	return nil
}

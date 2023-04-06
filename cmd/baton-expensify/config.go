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

	PartnerUserID     string `mapstructure:"partner-user-id"`
	PartnerUserSecret string `mapstructure:"partner-user-secret"`
}

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *config) error {
	if cfg.PartnerUserID == "" {
		return fmt.Errorf("partner user id is missing")
	}

	if cfg.PartnerUserSecret == "" {
		return fmt.Errorf("partner user secret is missing")
	}

	return nil
}

// cmdFlags sets the cmdFlags required for the connector.
func cmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("partner-user-id", "", "The Expensify partner user id used to connect to the Expensify API. ($BATON_PARTNER_USER_ID)")
	cmd.PersistentFlags().String("partner-user-secret", "", "The Expensify partner user secret used to connect to the Expensify API. ($BATON_PARTNER_USER_SECRET)")
}

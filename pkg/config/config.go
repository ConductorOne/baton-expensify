package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	partnerUserIdField = field.StringField(
		"partner-user-id",
		field.WithDisplayName("User ID"),
		field.WithDescription("The Expensify partner user id used to connect to the Expensify API."),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)

	partnerUserSecretField = field.StringField(
		"partner-user-secret",
		field.WithDisplayName("User Secret"),
		field.WithDescription("The Expensify partner user secret used to connect to the Expensify API."),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)

	BaseURLField = field.StringField(
		"base-url",
		field.WithDescription("Override the Expensify API URL (for testing)"),
		field.WithHidden(true),
		field.WithExportTarget(field.ExportTargetCLIOnly),
	)
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{
		partnerUserIdField,
		partnerUserSecretField,
		BaseURLField,
	},
	field.WithConnectorDisplayName("Expensify"),
	field.WithHelpUrl("/docs/baton/expensify"),
	field.WithIconUrl("/static/app-icons/expensify.svg"),
)

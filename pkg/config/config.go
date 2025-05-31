package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	partnerUserIdField = field.StringField(
		"expensify_user_id",
		field.WithDisplayName("User ID"),
		field.WithDescription("The Expensify partner user id used to connect to the Expensify API."),
		field.WithIsSecret(true),
	)

	partnerUserSecretField = field.StringField(
		"expensify_user_secret",
		field.WithDisplayName("User Secret"),
		field.WithDescription("The Expensify partner user secret used to connect to the Expensify API."),
		field.WithIsSecret(true),
	)
)

//go:generate go run ./gen
var Config = field.NewConfiguration([]field.SchemaField{
	partnerUserIdField,
	partnerUserSecretField,
})

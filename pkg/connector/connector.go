package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-expensify/pkg/expensify"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: annotationsForUserResourceType(),
	}
	resourceTypePolicy = &v2.ResourceType{
		Id:          "policy",
		DisplayName: "Policy",
	}
)

type Expensify struct {
	client *expensify.Client
}

func (as *Expensify) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(as.client),
		policyBuilder(as.client),
	}
}

// Metadata returns metadata about the connector.
func (as *Expensify) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Expensify",
		Description: "Connector syncing users and policies from Expensify to Baton",
	}, nil
}

// Validate hits the Expensify API to validate API credentials.
func (as *Expensify) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, _, err := as.client.GetPolicies(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("expensify-connector: %w", err)
	}
	return nil, nil
}

// New returns the Expensify connector.
func New(ctx context.Context, partnerUserID string, partnerUserSecret string) (*Expensify, error) {
	client, err := expensify.NewClient(ctx, partnerUserID, partnerUserSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create expensify client: %w", err)
	}

	return &Expensify{
		client: client,
	}, nil
}

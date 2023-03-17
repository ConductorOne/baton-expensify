package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-expensify/pkg/expensify"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
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
	}, nil
}

// Validate hits the Expensify API to validate API credentials.
func (as *Expensify) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, err := as.client.GetPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("expensify-connector: %w", err)
	}
	return nil, nil
}

// New returns the Expensify connector.
func New(ctx context.Context, partnerUserID string, partnerUserSecret string) (*Expensify, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	return &Expensify{
		client: expensify.NewClient(partnerUserID, partnerUserSecret, httpClient),
	}, nil
}

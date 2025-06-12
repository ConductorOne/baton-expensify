package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-expensify/pkg/expensify"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"

	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

var roles = map[string]string{
	"admin":   "admin",
	"auditor": "auditor",
	"user":    "user",
}

type policyResourceType struct {
	resourceType *v2.ResourceType
	client       *expensify.Client
}

func (o *policyResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func policyBuilder(client *expensify.Client) *policyResourceType {
	return &policyResourceType{
		resourceType: resourceTypePolicy,
		client:       client,
	}
}

// Create a new connector resource for an Expensify policy.
func policyResource(ctx context.Context, policy expensify.Policy) (*v2.Resource, error) {
	policyOptions := []rs.ResourceOption{
		rs.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: resourceTypeUser.Id},
		),
	}

	ret, err := rs.NewResource(policy.Name, resourceTypePolicy, policy.ID, policyOptions...)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *policyResourceType) List(ctx context.Context, resourceId *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	var annos annotations.Annotations
	policies, rl, err := o.client.GetPolicies(ctx)
	nextPage := pt.Token + "1"

	if err != nil {
		annos.WithRateLimiting(rl)
		return nil, nextPage, annos, err
	}

	for _, policy := range policies {
		pr, err := policyResource(ctx, policy)
		if err != nil {
			annos.WithRateLimiting(rl)
			return nil, nextPage, annos, err
		}
		rv = append(rv, pr)
	}

	annos.WithRateLimiting(rl)
	return rv, nextPage, annos, nil
}

func (o *policyResourceType) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	for _, role := range roles {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeUser),
			ent.WithDescription(fmt.Sprintf("Role in %s Expensify policy", resource.DisplayName)),
			ent.WithDisplayName(fmt.Sprintf("%s Policy %s", resource.DisplayName, role)),
		}

		permissionEn := ent.NewPermissionEntitlement(resource, role, permissionOptions...)
		rv = append(rv, permissionEn)
	}
	return rv, "", nil, nil
}

func (o *policyResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var annos annotations.Annotations
	policyEmployees, rl, err := o.client.GetPolicyEmployees(ctx, resource.Id.Resource)
	nextPage := pt.Token + "1"

	if err != nil {
		annos.WithRateLimiting(rl)
		return nil, nextPage, annos, err
	}

	var rv []*v2.Grant
	for _, policyEmployee := range policyEmployees {
		roleName, ok := roles[policyEmployee.Role]
		if !ok {
			ctxzap.Extract(ctx).Warn("Unknown Expensify Role Name, skipping",
				zap.String("role_name", policyEmployee.Role),
				zap.String("user", policyEmployee.Email),
			)
			continue
		}
		policyEmployeeCopy := policyEmployee
		ur, err := userResource(ctx, &policyEmployeeCopy, resource.Id)
		if err != nil {
			annos.WithRateLimiting(rl)
			return nil, nextPage, annos, err
		}

		permissionGrant := grant.NewGrant(resource, roleName, ur.Id)
		rv = append(rv, permissionGrant)
	}

	annos.WithRateLimiting(rl)
	return rv, nextPage, annos, nil
}

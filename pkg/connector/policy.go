package connector

import (
	"context"
	"fmt"
	"sync"

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

	// Cache for policy employees to avoid redundant API calls
	cachedPolicyEmployees map[string][]expensify.User
	cacheMutex            sync.Mutex
}

func (o *policyResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func policyBuilder(client *expensify.Client) *policyResourceType {
	return &policyResourceType{
		resourceType:          resourceTypePolicy,
		client:                client,
		cachedPolicyEmployees: make(map[string][]expensify.User),
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

// getCachedPolicyEmployees returns cached policy employees data, fetching it if not cached.
func (o *policyResourceType) getCachedPolicyEmployees(ctx context.Context, policyId string) ([]expensify.User, error) {
	// Lock the mutex to ensure thread safety
	o.cacheMutex.Lock()
	defer o.cacheMutex.Unlock()

	l := ctxzap.Extract(ctx)

	// Return cached data if it exists
	if employees, found := o.cachedPolicyEmployees[policyId]; found {
		l.Info("Returning cached employees")
		return employees, nil
	}

	// Otherwise, fetch the data from the API
	employees, err := o.client.GetPolicyEmployees(ctx, policyId)
	if err != nil {
		l.Error("Error fetching employees from API", zap.Error(err))
		return nil, err
	}

	// Store in cache for future use
	o.cachedPolicyEmployees[policyId] = employees

	l.Info("Storing employees in cache")
	return employees, nil
}

func (o *policyResourceType) List(ctx context.Context, resourceId *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	policies, err := o.client.GetPolicies(ctx)

	if err != nil {
		return nil, "", nil, err
	}

	// Pre-populate the cache with all policy employees to avoid individual API calls
	policyIDs := make([]string, len(policies))
	for i, policy := range policies {
		policyIDs[i] = policy.ID
	}

	// Fetch all policy employees in one batch if possible
	// Note: This would require adding a batch method to the client
	// For now, we'll let the cache handle it as needed

	for _, policy := range policies {
		pr, err := policyResource(ctx, policy)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, pr)
	}

	return rv, "", nil, nil
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
	policyEmployees, err := o.getCachedPolicyEmployees(ctx, resource.Id.Resource)

	if err != nil {
		return nil, "", nil, err
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
			return nil, "", nil, err
		}

		permissionGrant := grant.NewGrant(resource, roleName, ur.Id)
		rv = append(rv, permissionGrant)
	}

	return rv, "", nil, nil
}

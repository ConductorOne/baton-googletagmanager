package connector

import (
	"context"
	"fmt"
	"slices"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/api/tagmanager/v2"
)

type containerBuilder struct {
	client       *tagmanager.Service
	resourceType *v2.ResourceType
}

func (c *containerBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return containerResourceType
}

func containerResource(ctx context.Context, container *tagmanager.Container, parent *v2.ResourceId) (*v2.Resource, error) {
	resource, err := rs.NewResource(
		container.Name,
		containerResourceType,
		container.ContainerId,
		rs.WithParentResourceID(parent),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

// List returns all the containers from the database as resource objects.
func (c *containerBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentResourceID == nil {
		return nil, "", nil, nil
	}

	bag, page, err := parsePageToken(pToken.Token, &v2.ResourceId{ResourceType: containerResourceType.Id})
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to parse page token: %w", err)
	}

	parentPath := fmt.Sprintf("accounts/%s", parentResourceID.Resource)
	clreq := c.client.Accounts.Containers.List(parentPath).Context(ctx)

	if page != "" {
		clreq = clreq.PageToken(page)
	}

	cl, err := clreq.Do()
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to list containers: %w", err)
	}

	var rv []*v2.Resource
	for _, container := range cl.Container {
		cr, err := containerResource(ctx, container, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, cr)
	}

	nextPage, err := bag.NextToken(cl.NextPageToken)
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to set next page token: %w", err)
	}

	return rv, nextPage, nil, nil
}

// Entitlements returns slice of entitlements representing all possible permissions user can have on the container.
func (c *containerBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	for _, perm := range containerPermissions {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(userResourceType),
			ent.WithDisplayName(fmt.Sprintf("%s permission", perm)),
			ent.WithDescription(fmt.Sprintf("%s permission in GoogleTagManager under container %s", perm, resource.DisplayName)),
		}

		rv = append(
			rv,
			ent.NewPermissionEntitlement(
				resource,
				perm,
				permissionOptions...,
			),
		)
	}

	return rv, "", nil, nil
}

// Grants returns slice of grants representing all permissions user have granted on the container.
func (c *containerBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	bag, page, err := parsePageToken(pToken.Token, resource.Id)
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to parse page token: %w", err)
	}

	parentAccID := resource.ParentResourceId.Resource
	parentPath := fmt.Sprintf("accounts/%s", parentAccID)
	upl := c.client.Accounts.UserPermissions.List(parentPath).Context(ctx)

	if page != "" {
		upl = upl.PageToken(page)
	}

	ul, err := upl.Do()
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to list user permissions: %w", err)
	}

	var rv []*v2.Grant
	for _, up := range ul.UserPermission {
		for _, ca := range up.ContainerAccess {
			if ca.ContainerId != resource.Id.Resource {
				continue
			}

			if !slices.Contains(containerPermissions, ca.Permission) {
				l.Warn("found invalid permission during container grant creation", zap.String("permission", ca.Permission))

				continue
			}

			id := fmt.Sprintf("%s:%s", parentAccID, up.EmailAddress)
			principalID, err := rs.NewResourceID(userResourceType, id)
			if err != nil {
				return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to create resource id: %w", err)
			}

			rv = append(rv, grant.NewGrant(resource, ca.Permission, principalID))
		}
	}

	nextPage, err := bag.NextToken(ul.NextPageToken)
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to set next page token: %w", err)
	}

	return rv, nextPage, nil, nil
}

func newContainerBuilder(client *tagmanager.Service) *containerBuilder {
	return &containerBuilder{
		client:       client,
		resourceType: containerResourceType,
	}
}

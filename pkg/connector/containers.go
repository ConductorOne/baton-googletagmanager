package connector

import (
	"context"
	"fmt"
	"slices"
	"strings"

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

func (c *containerBuilder) FindRelevantPermissions(ctx context.Context, accID, containerID, userID, permission string, revoke bool) ([]string, error) {
	var rv []string
	pageToken := ""
	for {
		parentPath := fmt.Sprintf("accounts/%s", accID)
		uplreq := c.client.Accounts.UserPermissions.List(parentPath).Context(ctx)

		if pageToken != "" {
			uplreq.PageToken(pageToken)
		}

		ups, err := uplreq.Do()
		if err != nil {
			return nil, fmt.Errorf("googletagmanager-connector: failed to list user permissions: %w", err)
		}

		uParts := strings.Split(userID, ":")
		if len(uParts) != 2 {
			return nil, fmt.Errorf("googletagmanager-connector: invalid user id: %s", userID)
		}

		mail := uParts[1]
		for _, up := range ups.UserPermission {
			if up.EmailAddress != mail {
				continue
			}

			if revoke && up.ContainerAccess == nil {
				continue
			}

			if !revoke && up.ContainerAccess == nil {
				rv = append(rv, up.Path)
			}

			for _, ca := range up.ContainerAccess {
				if ca.ContainerId != containerID {
					continue
				}

				if revoke {
					if ca.Permission == permission {
						rv = append(rv, up.Path)
					}
				} else {
					if ca.Permission == permission {
						continue
					}

					rv = append(rv, up.Path)
				}
			}
		}

		if ups.NextPageToken == "" {
			break
		}

		pageToken = ups.NextPageToken
	}

	return rv, nil
}

func (c *containerBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"googletagmanager-connector: only users can be granted permissions on containers",
			zap.String("principal", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("googletagmanager-connector: only users can be granted permissions on containers")
	}

	container, userID, permission := entitlement.Resource, principal.Id.Resource, entitlement.Slug
	accID := container.ParentResourceId.Resource
	pPaths, err := c.FindRelevantPermissions(ctx, accID, container.Id.Resource, userID, permission, false)
	if err != nil {
		return nil, err
	}

	if len(pPaths) == 0 {
		l.Info(
			"googletagmanager-connector: permission already granted",
			zap.String("principal", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("permission", permission),
		)

		return nil, nil
	}

	for _, pPath := range pPaths {
		pg, err := c.client.Accounts.UserPermissions.Get(pPath).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("googletagmanager-connector: failed to get permission: %w", err)
		}

		// update existing permission
		pg.ContainerAccess = append(pg.ContainerAccess,
			&tagmanager.ContainerAccess{
				ContainerId: container.Id.Resource,
				Permission:  permission,
			},
		)

		// update in API
		_, err = c.client.Accounts.UserPermissions.Update(pPath, pg).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("googletagmanager-connector: failed to grant permission: %w", err)
		}
	}

	return nil, nil
}

func (c *containerBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	principal := grant.Principal
	entitlement := grant.Entitlement

	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"googletagmanager-connector: only users can have permissions on containers revoked",
			zap.String("principal", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("googletagmanager-connector: only users can have permissions on containers revoked")
	}

	container, userID, permission := entitlement.Resource, principal.Id.Resource, entitlement.Slug
	accID := container.ParentResourceId.Resource
	pPaths, err := c.FindRelevantPermissions(ctx, accID, container.Id.Resource, userID, permission, true)
	if err != nil {
		return nil, err
	}

	if len(pPaths) == 0 {
		l.Info(
			"googletagmanager-connector: permission already revoked",
			zap.String("principal", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("permission", permission),
		)

		return nil, nil
	}

	for _, pPath := range pPaths {
		pg, err := c.client.Accounts.UserPermissions.Get(pPath).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("googletagmanager-connector: failed to get permission: %w", err)
		}

		// no-access role is used to revoke permissions
		pg.ContainerAccess = append(pg.ContainerAccess, &tagmanager.ContainerAccess{
			ContainerId: container.Id.Resource,
			Permission:  NoAccessRole,
		})

		_, err = c.client.Accounts.UserPermissions.Update(pPath, pg).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("googletagmanager-connector: failed to revoke permission: %w", err)
		}
	}

	return nil, nil
}

func newContainerBuilder(client *tagmanager.Service) *containerBuilder {
	return &containerBuilder{
		client:       client,
		resourceType: containerResourceType,
	}
}

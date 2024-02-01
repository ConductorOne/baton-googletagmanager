package connector

import (
	"context"
	"fmt"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
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
		rs.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: roleResourceType.Id},
		),
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

// TODO: Implement and comment this method.
func (c *containerBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// TODO: Implement and comment this method.
func (c *containerBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newContainerBuilder(client *tagmanager.Service) *containerBuilder {
	return &containerBuilder{
		client:       client,
		resourceType: containerResourceType,
	}
}

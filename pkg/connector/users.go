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

type userBuilder struct {
	client       *tagmanager.Service
	resourceType *v2.ResourceType
}

func (u *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

func userResource(ctx context.Context, mail string, parent *v2.ResourceId) (*v2.Resource, error) {
	userTraitOptions := []rs.UserTraitOption{
		rs.WithStatus(v2.UserTrait_Status_STATUS_ENABLED),
		rs.WithEmail(mail, true),
		rs.WithUserLogin(mail),
	}

	userID := fmt.Sprintf("%s:%s", parent.Resource, mail)
	resource, err := rs.NewUserResource(
		mail,
		userResourceType,
		userID,
		userTraitOptions,
		rs.WithParentResourceID(parent),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (u *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentResourceID == nil {
		return nil, "", nil, nil
	}

	bag, page, err := parsePageToken(pToken.Token, &v2.ResourceId{ResourceType: userResourceType.Id})
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to parse page token: %w", err)
	}

	ulreq := u.client.Accounts.UserPermissions.List(parentResourceID.Resource).Context(ctx)

	if page != "" {
		ulreq = ulreq.PageToken(page)
	}

	ul, err := ulreq.Do()
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to list users: %w", err)
	}

	var rv []*v2.Resource
	for _, up := range ul.UserPermission {
		ur, err := userResource(ctx, up.EmailAddress, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, ur)
	}

	nextPage, err := bag.NextToken(ul.NextPageToken)
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to set next page token: %w", err)
	}

	return rv, nextPage, nil, nil
}

// Entitlements always returns an empty slice for users.
func (u *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (u *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newUserBuilder(client *tagmanager.Service) *userBuilder {
	return &userBuilder{
		client:       client,
		resourceType: userResourceType,
	}
}

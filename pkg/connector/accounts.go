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

const (
	RolePermissionUnspecifiedRole      = "accountPermissionUnspecified"
	AdminRole                          = "admin"
	NoAccessRole                       = "noAccess"
	UserRole                           = "user"
	ApproveRole                        = "approve"
	ContainerPermissionUnspecifiedRole = "containerPermissionUnspecified"
	EditRole                           = "edit"
	PublishRole                        = "publish"
	ReadRole                           = "read"
)

var (
	accountPermissions = []string{
		RolePermissionUnspecifiedRole,
		AdminRole,
		NoAccessRole,
		UserRole,
	}
	containerPermissions = []string{
		ApproveRole,
		ContainerPermissionUnspecifiedRole,
		EditRole,
		PublishRole,
		ReadRole,
	}
)

type accountBuilder struct {
	client       *tagmanager.Service
	resourceType *v2.ResourceType
	accountMap   map[string]struct{}
}

func (a *accountBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return accountResourceType
}

func accountResource(ctx context.Context, account *tagmanager.Account) (*v2.Resource, error) {
	resource, err := rs.NewResource(
		account.Name,
		accountResourceType,
		account.AccountId,
		rs.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: userResourceType.Id},
			&v2.ChildResourceType{ResourceTypeId: containerResourceType.Id},
		),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

// List returns all the accounts from the database as resource objects.
func (a *accountBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	bag, page, err := parsePageToken(pToken.Token, &v2.ResourceId{ResourceType: accountResourceType.Id})
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to parse page token: %w", err)
	}

	alreq := a.client.Accounts.List().Context(ctx)

	if page != "" {
		alreq = alreq.PageToken(page)
	}

	al, err := alreq.Do()
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to list accounts: %w", err)
	}

	var rv []*v2.Resource
	for _, acc := range al.Account {
		if _, ok := a.accountMap[acc.AccountId]; !ok && len(a.accountMap) > 0 {
			continue
		}

		ar, err := accountResource(ctx, acc)
		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, ar)
	}

	nextPage, err := bag.NextToken(al.NextPageToken)
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to set next page token: %w", err)
	}

	return rv, nextPage, nil, nil
}

// Entitlements returns slice of entititlements representing all possible permissions user can have on the account.
func (a *accountBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	for _, perm := range accountPermissions {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(userResourceType),
			ent.WithDisplayName(fmt.Sprintf("%s permission", perm)),
			ent.WithDescription(fmt.Sprintf("%s permission in GoogleTagManager under account %s", perm, resource.DisplayName)),
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

// Grants returns slice of grants representing all permissions user have granted on the account.
func (a *accountBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	bag, page, err := parsePageToken(pToken.Token, resource.Id)
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to parse page token: %w", err)
	}

	accID := resource.Id.Resource
	parentPath := fmt.Sprintf("accounts/%s", accID)
	upl := a.client.Accounts.UserPermissions.List(parentPath).Context(ctx)

	if page != "" {
		upl = upl.PageToken(page)
	}

	ul, err := upl.Do()
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to list user permissions: %w", err)
	}

	var rv []*v2.Grant
	for _, up := range ul.UserPermission {
		if up.AccountId != accID {
			return nil, "", nil, fmt.Errorf("googletagmanager-connector: found invalid account id: %s", up.AccountId)
		}

		if !slices.Contains(accountPermissions, up.AccountAccess.Permission) {
			l.Warn("found invalid permission during account grant creation", zap.String("permission", up.AccountAccess.Permission))

			continue
		}

		id := fmt.Sprintf("%s:%s", accID, up.EmailAddress)
		principalID, err := rs.NewResourceID(userResourceType, id)
		if err != nil {
			return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to create resource id: %w", err)
		}

		rv = append(rv, grant.NewGrant(resource, up.AccountAccess.Permission, principalID))
	}

	nextPage, err := bag.NextToken(ul.NextPageToken)
	if err != nil {
		return nil, "", nil, fmt.Errorf("googletagmanager-connector: failed to set next page token: %w", err)
	}

	return rv, nextPage, nil, nil
}

func newAccountBuilder(client *tagmanager.Service, accounts []string) *accountBuilder {
	accMap := make(map[string]struct{}, len(accounts))
	for _, acc := range accounts {
		accMap[acc] = struct{}{}
	}

	return &accountBuilder{
		client:       client,
		resourceType: accountResourceType,
		accountMap:   accMap,
	}
}

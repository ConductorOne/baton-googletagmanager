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
			&v2.ChildResourceType{ResourceTypeId: roleResourceType.Id},
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

// TODO: Implement and comment this method.
func (a *accountBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// TODO: Implement and comment this method.
func (a *accountBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
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

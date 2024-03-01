package connector

import (
	"context"
	"fmt"
	"io"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/api/option"
	"google.golang.org/api/tagmanager/v2"
)

type GoogleTagManager struct {
	accounts []string
	client   *tagmanager.Service
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (g *GoogleTagManager) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		newAccountBuilder(g.client, g.accounts),
		newContainerBuilder(g.client),
		newUserBuilder(g.client),
	}
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (g *GoogleTagManager) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (g *GoogleTagManager) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "GoogleTagManager",
		Description: "Connector syncing Google Tag Manager accounts, roles, users and containers to Baton",
	}, nil
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid.
func (d *GoogleTagManager) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, err := d.client.Accounts.List().Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("googletagmanager-connector: error validating credentials: %w", err)
	}

	return nil, nil
}

// New returns a new instance of the connector.
func New(ctx context.Context, ac uhttp.AuthCredentials, accounts []string) (*GoogleTagManager, error) {
	httpClient, err := ac.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("googletagmanager-connector: error creating http client: %w", err)
	}

	tagmanagerService, err := tagmanager.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("error creating tagmanager service: %w", err)
	}

	return &GoogleTagManager{
		client:   tagmanagerService,
		accounts: accounts,
	}, nil
}

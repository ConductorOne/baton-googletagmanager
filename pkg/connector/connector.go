package connector

import (
	"context"
	"fmt"
	"io"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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
		newRoleBuilder(g.client),
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
	return nil, nil
}

// New returns a new instance of the connector.
func New(ctx context.Context, credentials []byte, accounts []string) (*GoogleTagManager, error) {
	l := ctxzap.Extract(ctx)
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, l))
	if err != nil {
		return nil, fmt.Errorf("googletagmanager-connector: error creating http client: %w", err)
	}

	jwt, err := google.JWTConfigFromJSON(
		credentials,
		tagmanager.TagmanagerManageAccountsScope,
		tagmanager.TagmanagerManageUsersScope,
		tagmanager.TagmanagerEditContainersScope,
		tagmanager.TagmanagerEditContainerversionsScope,
		tagmanager.TagmanagerDeleteContainersScope,
		tagmanager.TagmanagerPublishScope,
	)
	if err != nil {
		return nil, fmt.Errorf("googletagmanager-connector: error creating JWT config: %w", err)
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	httpClient = &http.Client{
		Transport: &oauth2.Transport{
			Base:   httpClient.Transport,
			Source: jwt.TokenSource(ctx),
		},
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

package main

import (
	"context"
	"fmt"
	"os"

	cfg "github.com/conductorone/baton-googletagmanager/pkg/config"
	"github.com/conductorone/baton-googletagmanager/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/tagmanager/v2"
)

var version = "dev"

func main() {
	ctx := context.Background()

	_, cmd, err := config.DefineConfiguration(
		ctx,
		"baton-googletagmanager",
		getConnector,
		cfg.Config,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilder(&connector.GoogleTagManager{}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version

	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, cc *cfg.Googletagmanager) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)
	if err := cfg.ValidateConfig(cc); err != nil {
		return nil, err
	}

	var ac uhttp.AuthCredentials = &uhttp.NoAuth{}
	if cc.CredentialsJSONFilePath != "" {
		credentials, err := os.ReadFile(cc.CredentialsJSONFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading credentials JSON file: %w", err)
		}

		ac = uhttp.NewOAuth2JWT(
			credentials,
			[]string{
				tagmanager.TagmanagerManageAccountsScope,
				tagmanager.TagmanagerManageUsersScope,
				tagmanager.TagmanagerEditContainersScope,
				tagmanager.TagmanagerEditContainerversionsScope,
				tagmanager.TagmanagerDeleteContainersScope,
				tagmanager.TagmanagerPublishScope,
			},
			google.JWTConfigFromJSON,
		)
	}

	cb, err := connector.New(ctx, ac, cc.Accounts)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	c, err := connectorbuilder.NewConnector(ctx, cb)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}

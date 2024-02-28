package main

import (
	"context"
	"fmt"
	"os"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/tagmanager/v2"

	"github.com/conductorone/baton-googletagmanager/pkg/connector"
)

var version = "dev"

func main() {
	ctx := context.Background()

	cfg := &config{}
	cmd, err := cli.NewCmd(ctx, "baton-googletagmanager", cfg, validateConfig, getConnector)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version
	cmdFlags(cmd)

	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, cfg *config) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)

	var ac uhttp.AuthCredentials = &uhttp.NoAuth{}
	if cfg.CredentialsJSONFilePath != "" {
		credentials, err := os.ReadFile(cfg.CredentialsJSONFilePath)
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

	cb, err := connector.New(ctx, ac, cfg.Accounts)
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

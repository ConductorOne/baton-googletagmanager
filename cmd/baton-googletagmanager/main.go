package main

import (
	"context"
	"fmt"
	"io"
	"os"

	cfg "github.com/conductorone/baton-googletagmanager/pkg/config"
	"github.com/conductorone/baton-googletagmanager/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	googlecredentialguard "github.com/conductorone/google-credential-guard"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"golang.org/x/oauth2/jwt"
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
	if cc.CredentialsJsonFilePath != "" {
		var err error
		ac, err = loadCredentialAuth(cc.CredentialsJsonFilePath)
		if err != nil {
			return nil, err
		}
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

// loadCredentialAuth reads only the file chosen by the local connector operator.
// A hosted caller must never expose this path field to a remote tenant.
func loadCredentialAuth(path string) (uhttp.AuthCredentials, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening credentials JSON file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error stating credentials JSON file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > googlecredentialguard.MaxJSONBytes {
		return nil, fmt.Errorf("credentials JSON file must be regular and at most %d bytes", googlecredentialguard.MaxJSONBytes)
	}

	credentials, err := io.ReadAll(io.LimitReader(file, googlecredentialguard.MaxJSONBytes+1))
	if err != nil {
		return nil, fmt.Errorf("error reading credentials JSON file: %w", err)
	}
	key, err := googlecredentialguard.ParseServiceAccountJSON(credentials)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials JSON file: %w", err)
	}

	return uhttp.NewOAuth2JWT(nil, []string{
		tagmanager.TagmanagerManageAccountsScope,
		tagmanager.TagmanagerManageUsersScope,
		tagmanager.TagmanagerEditContainersScope,
		tagmanager.TagmanagerEditContainerversionsScope,
		tagmanager.TagmanagerDeleteContainersScope,
		tagmanager.TagmanagerPublishScope,
	}, func(_ []byte, scopes ...string) (*jwt.Config, error) {
		return key.JWTConfig(scopes...)
	}), nil
}

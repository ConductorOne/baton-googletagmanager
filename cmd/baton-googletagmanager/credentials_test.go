package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	guard "github.com/conductorone/google-credential-guard"
	"google.golang.org/api/tagmanager/v2"
)

func testServiceAccount(t *testing.T) map[string]any {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	// Console keys use PKCS#8.
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return map[string]any{
		"type": "service_account", "project_id": "example-project", "private_key_id": "key-id",
		"private_key": string(pemKey), "client_email": "example@example-project.iam.gserviceaccount.com",
		"client_id": "123456789", "auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": guard.TokenURL, "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/example",
		"universe_domain":      "googleapis.com",
	}
}

func writeCredential(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "key.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCredentialAuthAcceptsConsoleKeyAndPreservesScopes(t *testing.T) {
	ac, err := loadCredentialAuth(writeCredential(t, testServiceAccount(t)))
	if err != nil {
		t.Fatal(err)
	}
	jwtAuth, ok := ac.(*uhttp.OAuth2JWT)
	if !ok {
		t.Fatalf("auth type %T", ac)
	}
	cfg, err := jwtAuth.CreateJWTConfig(jwtAuth.Credentials, jwtAuth.Scopes...)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TokenURL != guard.TokenURL {
		t.Errorf("token URL = %q", cfg.TokenURL)
	}
	if cfg.Email != "example@example-project.iam.gserviceaccount.com" {
		t.Errorf("email = %q", cfg.Email)
	}
	want := []string{
		tagmanager.TagmanagerManageAccountsScope, tagmanager.TagmanagerManageUsersScope,
		tagmanager.TagmanagerEditContainersScope, tagmanager.TagmanagerEditContainerversionsScope,
		tagmanager.TagmanagerDeleteContainersScope, tagmanager.TagmanagerPublishScope,
	}
	if len(cfg.Scopes) != len(want) {
		t.Fatalf("scopes = %v", cfg.Scopes)
	}
	for i := range want {
		if cfg.Scopes[i] != want[i] {
			t.Errorf("scope[%d] = %q, want %q", i, cfg.Scopes[i], want[i])
		}
	}
}

func TestCredentialAuthRejectsUnsafeFiles(t *testing.T) {
	valid := testServiceAccount(t)
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"external file", func(m map[string]any) {
			m["type"] = "external_account"
			m["credential_source"] = map[string]any{"file": "/etc/passwd"}
		}},
		{"external url", func(m map[string]any) {
			m["type"] = "external_account"
			m["credential_source"] = map[string]any{"url": "http://127.0.0.1/"}
		}},
		{"external executable", func(m map[string]any) {
			m["type"] = "external_account"
			m["credential_source"] = map[string]any{"executable": map[string]any{"command": "id"}}
		}},
		{"hostile token uri", func(m map[string]any) { m["token_uri"] = "https://attacker.example/token" }},
		{"unknown field", func(m map[string]any) { m["future_google_field"] = "private-value" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := make(map[string]any, len(valid))
			for k, v := range valid {
				m[k] = v
			}
			tc.mutate(m)
			_, err := loadCredentialAuth(writeCredential(t, m))
			if err == nil {
				t.Fatal("unsafe credential accepted")
			}
			if tc.name == "unknown field" && (!strings.Contains(err.Error(), "future_google_field") || strings.Contains(err.Error(), "private-value")) {
				t.Fatalf("unknown field diagnostic was unsafe or unhelpful: %v", err)
			}
		})
	}
	t.Run("oversized", func(t *testing.T) {
		m := make(map[string]any, len(valid))
		for k, v := range valid {
			m[k] = v
		}
		m["client_id"] = strings.Repeat("x", guard.MaxJSONBytes)
		if _, err := loadCredentialAuth(writeCredential(t, m)); err == nil {
			t.Fatal("oversized credential accepted")
		}
	})
}

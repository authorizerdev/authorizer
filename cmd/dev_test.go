package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/server"
)

// devFlagSet mirrors the subset of root flags applyDevDefaults consults, so
// tests can mark a flag as explicitly set exactly the way cobra does.
func devFlagSet(t *testing.T, set map[string]string) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet("dev", pflag.ContinueOnError)
	var s string
	var b bool
	for _, name := range []string{"host", "database-type", "database-url", "url", "client-id", "client-secret",
		"admin-secret", "encryption-key", "jwt-type", "jwt-secret", "jwt-private-key", "jwt-public-key"} {
		fs.StringVar(&s, name, "", "")
	}
	fs.BoolVar(&b, "mcp-enabled", false, "")
	fs.BoolVar(&b, "enable-client-id-metadata-document", false, "")
	for k, v := range set {
		require.NoError(t, fs.Set(k, v))
	}
	return fs
}

func TestLoadOrCreateDevState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".authorizer")

	first, err := loadOrCreateDevState(dir)
	require.NoError(t, err)

	t.Run("generates every secret and a usable RSA key pair", func(t *testing.T) {
		for name, v := range map[string]string{
			"client_id": first.ClientID, "client_secret": first.ClientSecret,
			"admin_secret": first.AdminSecret, "encryption_key": first.EncryptionKey,
		} {
			assert.NotEmpty(t, v, name)
		}
		_, err := crypto.ParseRsaPrivateKeyFromPemStr(first.JWTPrivateKey)
		require.NoError(t, err)
		_, err = crypto.ParseRsaPublicKeyFromPemStr(first.JWTPublicKey)
		require.NoError(t, err)
	})

	t.Run("secrets file and dir are private, and git-ignored", func(t *testing.T) {
		fi, err := os.Stat(filepath.Join(dir, devStateFile))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
		di, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), di.Mode().Perm())
		gi, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Contains(t, string(gi), "*\n")
	})

	t.Run("second run reuses the same secrets", func(t *testing.T) {
		second, err := loadOrCreateDevState(dir)
		require.NoError(t, err)
		assert.Equal(t, first, second)
	})

	t.Run("a loosened secrets file is re-tightened on load", func(t *testing.T) {
		p := filepath.Join(dir, devStateFile)
		require.NoError(t, os.Chmod(p, 0o644))
		require.NoError(t, os.Chmod(dir, 0o755))
		_, err := loadOrCreateDevState(dir)
		require.NoError(t, err)
		fi, err := os.Stat(p)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
		di, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), di.Mode().Perm())
	})

	t.Run("an existing empty dir is adopted", func(t *testing.T) {
		empty := t.TempDir()
		_, err := loadOrCreateDevState(empty)
		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(empty, ".gitignore"))
	})

	t.Run("a non-empty dir that is not a dev dir is refused and left untouched", func(t *testing.T) {
		project := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(project, "go.mod"), []byte("module x\n"), 0o644))
		_, err := loadOrCreateDevState(project)
		assert.ErrorContains(t, err, "not empty")
		assert.NoFileExists(t, filepath.Join(project, ".gitignore"), "must not git-ignore someone's project")
		assert.NoFileExists(t, filepath.Join(project, devStateFile))
	})

	t.Run("an incomplete secrets file is an error, not a silent regenerate", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "bad")
		require.NoError(t, os.MkdirAll(bad, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(bad, devStateFile), []byte(`{"client_id":"x"}`), 0o600))
		_, err := loadOrCreateDevState(bad)
		assert.ErrorContains(t, err, "incomplete")
	})
}

func TestApplyDevDefaults(t *testing.T) {
	st := &devState{ClientID: "cid", ClientSecret: "csec", AdminSecret: "adm", EncryptionKey: "enc",
		JWTPrivateKey: "priv", JWTPublicKey: "pub"}

	t.Run("no flags: everything comes from the dev state", func(t *testing.T) {
		cfg := &config.Config{GRPCPort: 9091}
		srv := &server.Config{HTTPPort: 8080}
		applyDevDefaults(devFlagSet(t, nil), cfg, srv, st, "/tmp/d")

		assert.Equal(t, devHost, srv.Host)
		assert.Equal(t, "sqlite", cfg.DatabaseType)
		assert.Equal(t, filepath.Join("/tmp/d", "authorizer.db"), cfg.DatabaseURL)
		assert.Equal(t, "http://localhost:8080", cfg.AuthorizerURL)
		assert.Equal(t, "cid", cfg.ClientID)
		assert.Equal(t, "csec", cfg.ClientSecret)
		assert.Equal(t, "adm", cfg.AdminSecret)
		assert.Equal(t, "enc", cfg.EncryptionKey)
		assert.Equal(t, devJWTType, cfg.JWTType)
		assert.Equal(t, "priv", cfg.JWTPrivateKey)
		assert.Equal(t, "pub", cfg.JWTPublicKey)
		assert.True(t, cfg.MCPEnabled)
		assert.True(t, cfg.EnableClientIDMetadataDocument)
	})

	t.Run("url follows a non-default http port", func(t *testing.T) {
		cfg := &config.Config{}
		applyDevDefaults(devFlagSet(t, nil), cfg, &server.Config{HTTPPort: 9000}, st, "d")
		assert.Equal(t, "http://localhost:9000", cfg.AuthorizerURL)
	})

	t.Run("explicit flags always win", func(t *testing.T) {
		cfg := &config.Config{
			DatabaseURL: "postgres://x", AuthorizerURL: "https://auth.example.com",
			ClientID: "mine", AdminSecret: "my-admin", JWTType: "HS256", JWTSecret: "s",
		}
		srv := &server.Config{Host: "0.0.0.0", HTTPPort: 8080}
		fs := devFlagSet(t, map[string]string{
			"host": "0.0.0.0", "database-url": "postgres://x", "url": "https://auth.example.com",
			"client-id": "mine", "admin-secret": "my-admin", "jwt-type": "HS256",
			"mcp-enabled": "false", "enable-client-id-metadata-document": "false",
		})
		applyDevDefaults(fs, cfg, srv, st, "d")

		assert.Equal(t, "0.0.0.0", srv.Host)
		assert.Empty(t, cfg.DatabaseType, "database type is not forced to sqlite when only --database-url is set")
		assert.Equal(t, "postgres://x", cfg.DatabaseURL)
		assert.Equal(t, "https://auth.example.com", cfg.AuthorizerURL)
		assert.Equal(t, "mine", cfg.ClientID)
		assert.Equal(t, "csec", cfg.ClientSecret, "an unset half of the client pair still gets the generated value")
		assert.Equal(t, "my-admin", cfg.AdminSecret)
		assert.Equal(t, "HS256", cfg.JWTType)
		assert.Empty(t, cfg.JWTPrivateKey, "generated RSA keys are not injected under a caller-chosen algorithm")
		assert.False(t, cfg.MCPEnabled)
		assert.False(t, cfg.EnableClientIDMetadataDocument)
	})
}

func TestBuildDevInfo(t *testing.T) {
	cfg := &config.Config{
		AuthorizerURL: "http://localhost:8080/", GRPCPort: 9091, MCPEnabled: true,
		ClientID: "cid", ClientSecret: "csec", AdminSecret: "adm",
		DatabaseType: "sqlite", DatabaseURL: ".authorizer/authorizer.db",
	}
	info := buildDevInfo(cfg, &server.Config{Host: "0.0.0.0"}, ".authorizer")

	assert.Equal(t, "http://localhost:8080", info.URL)
	assert.Equal(t, "http://localhost:8080/app", info.LoginURL)
	assert.Equal(t, "http://localhost:8080/.well-known/openid-configuration", info.OpenIDConfigurationURL)
	assert.Equal(t, "http://localhost:8080/.well-known/jwks.json", info.JWKSURL)
	assert.Equal(t, "http://localhost:8080/mcp", info.MCPURL)
	assert.Equal(t, "localhost:9091", info.GRPCAddress, "a wildcard bind address is not dialable; report localhost")
	assert.True(t, filepath.IsAbs(info.StateDir))

	t.Run("mcp_url is omitted when MCP is off", func(t *testing.T) {
		off := *cfg
		off.MCPEnabled = false
		var buf bytes.Buffer
		require.NoError(t, writeDevInfo(&buf, buildDevInfo(&off, &server.Config{Host: devHost}, "d")))
		var m map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
		assert.NotContains(t, m, "mcp_url")
		assert.Equal(t, "127.0.0.1:9091", m["grpc_address"])
		assert.Equal(t, "adm", m["admin_secret"])
	})
}

// TestDevCommandWiring runs the real cobra command, so it catches regressions
// the unit tests above cannot: reading flags from the wrong FlagSet (inherited
// persistent flags must count as explicitly set) and skipping the root's
// PersistentPreRunE (Finalize + config validation).
func TestDevCommandWiring(t *testing.T) {
	savedRoot, savedDev := rootArgs, devArgs
	t.Cleanup(func() {
		rootArgs, devArgs = savedRoot, savedDev
		// Reset pflag's Changed bits so later tests in the package see a clean command.
		for _, fs := range []*pflag.FlagSet{RootCmd.PersistentFlags(), devCmd.Flags()} {
			fs.VisitAll(func(f *pflag.Flag) { f.Changed = false })
		}
		RootCmd.SetArgs(nil)
		RootCmd.SetOut(nil)
	})

	run := func(args ...string) (devInfo, error) {
		var out bytes.Buffer
		RootCmd.SetOut(&out)
		RootCmd.SetErr(&bytes.Buffer{})
		RootCmd.SetArgs(append([]string{"dev", "--print-config", "--dev-dir", filepath.Join(t.TempDir(), "d")}, args...))
		err := RootCmd.Execute()
		var info devInfo
		if err == nil {
			require.NoError(t, json.Unmarshal(out.Bytes(), &info))
		}
		return info, err
	}

	info, err := run("--host", "0.0.0.0", "--http-port", "9000", "--admin-secret", "explicit-admin")
	require.NoError(t, err)
	assert.Equal(t, "explicit-admin", info.AdminSecret, "an inherited persistent flag must override the dev default")
	assert.Equal(t, "http://localhost:9000", info.URL)
	assert.Equal(t, "http://localhost:9000/mcp", info.MCPURL)
	assert.NotEmpty(t, info.ClientID)

	_, err = run("--app-cookie-same-site", "bogus")
	assert.Error(t, err, "the root's PersistentPreRunE validation must still run under `dev`")
}

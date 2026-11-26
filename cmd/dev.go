package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/server"
)

const (
	// devStateFile holds the generated secrets. It is written once and reused,
	// so client credentials and signing keys stay stable across restarts.
	devStateFile = "dev-secrets.json"
	// devInfoFile is the machine-readable connection summary written on every
	// `authorizer dev` start — the same document --print-config prints.
	devInfoFile = "dev.json"
	devJWTType  = "RS256"
	devHost     = "127.0.0.1"
)

// devFrontendOrigins are the default ports of common frontend dev servers
// (Next.js / Create React App / Remix, Vite, Angular). The allowlist has no
// port wildcard, and the server default "*" is same-origin only for CSRF and
// redirect URIs, so without these a local app could not log in.
var devFrontendOrigins = []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:4200"}

var devArgs struct {
	dir         string
	printConfig bool
}

// devState is the generated, persisted material for a local dev instance.
type devState struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	AdminSecret   string `json:"admin_secret"`
	EncryptionKey string `json:"encryption_key"`
	JWTPrivateKey string `json:"jwt_private_key"`
	JWTPublicKey  string `json:"jwt_public_key"`
}

// devInfo is the connection summary printed by --print-config and written to
// <dir>/dev.json. Field names are a stable contract for scripts and agents.
type devInfo struct {
	URL                    string   `json:"url"`
	LoginURL               string   `json:"login_url"`
	DashboardURL           string   `json:"dashboard_url"`
	GraphQLURL             string   `json:"graphql_url"`
	RESTBaseURL            string   `json:"rest_base_url"`
	OpenAPIURL             string   `json:"openapi_url"`
	GRPCAddress            string   `json:"grpc_address"`
	OpenIDConfigurationURL string   `json:"openid_configuration_url"`
	JWKSURL                string   `json:"jwks_url"`
	MCPURL                 string   `json:"mcp_url,omitempty"`
	AllowedOrigins         []string `json:"allowed_origins"`
	ClientID               string   `json:"client_id"`
	ClientSecret           string   `json:"client_secret"`
	AdminSecret            string   `json:"admin_secret"`
	AdminSecretHeader      string   `json:"admin_secret_header"`
	DatabaseType           string   `json:"database_type"`
	DatabaseURL            string   `json:"database_url"`
	StateDir               string   `json:"state_dir"`
}

// devCmd runs a local server with zero required flags. It exists so a person
// or a coding agent can stand up a working instance and read back every value
// needed to integrate with it, without hand-picking secrets or keys.
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run a local development server with generated, persisted secrets",
	Long: "Runs Authorizer for local development without any required flags.\n\n" +
		"On first run it generates a client ID and secret, an admin secret, an encryption key " +
		"and an RS256 signing key pair, and saves them to <dir>/" + devStateFile + " (mode 0600). " +
		"Later runs reuse them, so credentials stay stable across restarts.\n\n" +
		"Defaults (each applies only when you do not pass the flag yourself): SQLite at " +
		"<dir>/authorizer.db, --host=" + devHost + ", --url=http://localhost:<http-port>, " +
		"--allowed-origins=<url>,http://localhost:3000,http://localhost:5173,http://localhost:4200, " +
		"--mcp-enabled and --enable-client-id-metadata-document. Every server flag is accepted and wins.\n\n" +
		"--print-config prints the connection details as JSON and exits without starting the server. " +
		"The same JSON is written to <dir>/" + devInfoFile + " on every start.\n\n" +
		"Not for production: the secrets sit in a local file.",
	PersistentPreRunE: func(c *cobra.Command, args []string) error {
		st, err := loadOrCreateDevState(devArgs.dir)
		if err != nil {
			return err
		}
		applyDevDefaults(c.Flags(), &rootArgs.config, &rootArgs.server, st, devArgs.dir)
		// A subcommand's PersistentPreRunE replaces the root's, so run the
		// root's validation explicitly, after the defaults are in place.
		return RootCmd.PersistentPreRunE(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		info := buildDevInfo(&rootArgs.config, &rootArgs.server, devArgs.dir)
		if devArgs.printConfig {
			return writeDevInfo(c.OutOrStdout(), info)
		}
		path := filepath.Join(devArgs.dir, devInfoFile)
		if err := writeDevInfoFile(path, info); err != nil {
			return err
		}
		// stderr: stdout carries the server's JSON log stream.
		fmt.Fprintf(os.Stderr, "authorizer dev: %s (connection details: %s)\n", info.URL, path)
		runRoot(c, args)
		return nil
	},
}

func init() {
	devCmd.Flags().StringVar(&devArgs.dir, "dev-dir", ".authorizer", "Directory for generated dev secrets, the SQLite database and dev.json")
	devCmd.Flags().BoolVar(&devArgs.printConfig, "print-config", false, "Print connection details as JSON and exit without starting the server")
	RootCmd.AddCommand(devCmd)
}

// loadOrCreateDevState returns the persisted dev secrets in dir, generating
// and saving them on first use.
//
// dir must be dedicated to `authorizer dev`: it is accepted only if it does
// not exist yet, is empty, or already holds dev-secrets.json. Anything else
// (e.g. --dev-dir=. inside a project) is refused, because the directory gets
// a .gitignore of `*` and mode 0700. Both are re-applied on every run.
func loadOrCreateDevState(dir string) (*devState, error) {
	path := filepath.Join(dir, devStateFile)
	if err := prepareDevDir(dir, path); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		// os.WriteFile only applies the mode on create; re-tighten in case it
		// was loosened since.
		if err := os.Chmod(path, 0o600); err != nil {
			return nil, fmt.Errorf("chmod %s: %w", path, err)
		}
		var st devState
		if err := json.Unmarshal(raw, &st); err != nil {
			return nil, fmt.Errorf("parse %s: %w (delete it to regenerate)", path, err)
		}
		if st.ClientID == "" || st.ClientSecret == "" || st.AdminSecret == "" ||
			st.EncryptionKey == "" || st.JWTPrivateKey == "" || st.JWTPublicKey == "" {
			return nil, fmt.Errorf("%s is incomplete (delete it to regenerate)", path)
		}
		return &st, nil
	case !errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	st, err := newDevState()
	if err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return nil, fmt.Errorf("write %s: %w", path, err)
	}
	return st, nil
}

func prepareDevDir(dir, statePath string) error {
	entries, err := os.ReadDir(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create dev dir %q: %w", dir, err)
		}
	case err != nil:
		return fmt.Errorf("read dev dir %q: %w", dir, err)
	case len(entries) > 0:
		if _, sErr := os.Stat(statePath); sErr != nil {
			return fmt.Errorf("--dev-dir %q is not empty and has no %s; use a dedicated directory (it is git-ignored and made private)", dir, devStateFile)
		}
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("chmod dev dir %q: %w", dir, err)
	}
	gitignore := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("# Generated by `authorizer dev`: local secrets, never commit.\n*\n"), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", gitignore, err)
	}
	return nil
}

func newDevState() (*devState, error) {
	var st devState
	for _, v := range []struct {
		dst   *string
		bytes int
	}{
		{&st.ClientID, 24},
		{&st.ClientSecret, 48},
		{&st.AdminSecret, 32},
		{&st.EncryptionKey, 32},
	} {
		s, err := crypto.NewRandomString(v.bytes)
		if err != nil {
			return nil, fmt.Errorf("generate dev secret: %w", err)
		}
		*v.dst = s
	}
	_, priv, pub, _, err := crypto.NewRSAKey(devJWTType, st.ClientID)
	if err != nil {
		return nil, fmt.Errorf("generate dev signing key: %w", err)
	}
	st.JWTPrivateKey, st.JWTPublicKey = priv, pub
	return &st, nil
}

// applyDevDefaults fills cfg and srv from st for every flag the caller did not
// set explicitly, so any server flag passed to `authorizer dev` wins.
func applyDevDefaults(flags *pflag.FlagSet, cfg *config.Config, srv *server.Config, st *devState, dir string) {
	unset := func(name string) bool { return !flags.Changed(name) }

	if unset("host") {
		srv.Host = devHost
	}
	if unset("database-type") && unset("database-url") {
		cfg.DatabaseType = "sqlite"
		cfg.DatabaseURL = filepath.Join(dir, "authorizer.db")
	}
	if unset("url") {
		port := srv.HTTPPort
		if port == 0 {
			port = defaultHTTPPort
		}
		cfg.AuthorizerURL = "http://localhost:" + strconv.Itoa(port)
	}
	if unset("allowed-origins") {
		// Explicit, so the server's own origin must be listed too: with an
		// explicit list the CSRF check no longer falls back to same-origin.
		cfg.AllowedOrigins = append([]string{cfg.CanonicalURL()}, devFrontendOrigins...)
	}
	if unset("client-id") {
		cfg.ClientID = st.ClientID
	}
	if unset("client-secret") {
		cfg.ClientSecret = st.ClientSecret
	}
	if unset("admin-secret") {
		cfg.AdminSecret = st.AdminSecret
	}
	if unset("encryption-key") {
		cfg.EncryptionKey = st.EncryptionKey
	}
	// The generated key pair is only meaningful for the generated algorithm;
	// any explicit JWT choice is left entirely to the caller.
	if unset("jwt-type") && unset("jwt-secret") && unset("jwt-private-key") && unset("jwt-public-key") {
		cfg.JWTType = devJWTType
		cfg.JWTPrivateKey = st.JWTPrivateKey
		cfg.JWTPublicKey = st.JWTPublicKey
	}
	if unset("mcp-enabled") {
		cfg.MCPEnabled = true
	}
	if unset("enable-client-id-metadata-document") {
		cfg.EnableClientIDMetadataDocument = true
	}
}

func buildDevInfo(cfg *config.Config, srv *server.Config, dir string) devInfo {
	base := cfg.CanonicalURL()
	grpcHost := srv.Host
	if grpcHost == "" || grpcHost == "0.0.0.0" || grpcHost == "::" {
		grpcHost = "localhost"
	}
	info := devInfo{
		URL:                    base,
		LoginURL:               base + "/app",
		DashboardURL:           base + "/dashboard",
		GraphQLURL:             base + "/graphql",
		RESTBaseURL:            base + "/v1",
		OpenAPIURL:             base + "/openapi.json",
		GRPCAddress:            net.JoinHostPort(grpcHost, strconv.Itoa(cfg.GRPCPort)),
		OpenIDConfigurationURL: base + "/.well-known/openid-configuration",
		JWKSURL:                base + "/.well-known/jwks.json",
		ClientID:               cfg.ClientID,
		ClientSecret:           cfg.ClientSecret,
		AdminSecret:            cfg.AdminSecret,
		AdminSecretHeader:      "X-Authorizer-Admin-Secret",
		AllowedOrigins:         cfg.AllowedOrigins,
		DatabaseType:           cfg.DatabaseType,
		DatabaseURL:            cfg.DatabaseURL,
		StateDir:               dir,
	}
	if cfg.MCPEnabled {
		info.MCPURL = cfg.MCPResource()
	}
	if abs, err := filepath.Abs(dir); err == nil {
		info.StateDir = abs
	}
	return info
}

func writeDevInfo(w io.Writer, info devInfo) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

func writeDevInfoFile(path string, info devInfo) error {
	var b strings.Builder
	if err := writeDevInfo(&b, info); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	// It holds secrets; keep it private even if it pre-existed with a looser mode.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}

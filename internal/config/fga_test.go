package config

import "testing"

func TestFGAStoreConfig(t *testing.T) {
	cases := []struct {
		name        string
		cfg         Config
		wantStore   string
		wantURL     string
		wantEnabled bool
	}{
		{
			name:        "sqlite main db reused, file URI added",
			cfg:         Config{DatabaseType: "sqlite", DatabaseURL: "data.db"},
			wantStore:   "sqlite",
			wantURL:     "file:data.db",
			wantEnabled: true,
		},
		{
			name:        "sqlite main db already a file URI",
			cfg:         Config{DatabaseType: "sqlite", DatabaseURL: "file:/var/lib/auth.db"},
			wantStore:   "sqlite",
			wantURL:     "file:/var/lib/auth.db",
			wantEnabled: true,
		},
		{
			name:        "postgres main db reused as-is",
			cfg:         Config{DatabaseType: "postgres", DatabaseURL: "postgres://u:p@h:5432/db"},
			wantStore:   "postgres",
			wantURL:     "postgres://u:p@h:5432/db",
			wantEnabled: true,
		},
		{
			name:        "mysql main db reused as-is",
			cfg:         Config{DatabaseType: "mysql", DatabaseURL: "u:p@tcp(h:3306)/db"},
			wantStore:   "mysql",
			wantURL:     "u:p@tcp(h:3306)/db",
			wantEnabled: true,
		},
		{
			name:        "mariadb maps to mysql store",
			cfg:         Config{DatabaseType: "mariadb", DatabaseURL: "u:p@tcp(h:3306)/db"},
			wantStore:   "mysql",
			wantURL:     "u:p@tcp(h:3306)/db",
			wantEnabled: true,
		},
		{
			name:        "mongodb main db => FGA disabled (no explicit store)",
			cfg:         Config{DatabaseType: "mongodb", DatabaseURL: "mongodb://h"},
			wantEnabled: false,
		},
		{
			name:        "dynamodb main db => FGA disabled",
			cfg:         Config{DatabaseType: "dynamodb", DatabaseURL: "http://h"},
			wantEnabled: false,
		},
		{
			name:        "explicit fga-store overrides a NoSQL main db",
			cfg:         Config{DatabaseType: "mongodb", DatabaseURL: "mongodb://h", FGAStore: "postgres", FGAStoreURL: "postgres://x/fga"},
			wantStore:   "postgres",
			wantURL:     "postgres://x/fga",
			wantEnabled: true,
		},
		{
			name:        "explicit sqlite override gets file URI",
			cfg:         Config{DatabaseType: "mongodb", FGAStore: "sqlite", FGAStoreURL: "fga.db"},
			wantStore:   "sqlite",
			wantURL:     "file:fga.db",
			wantEnabled: true,
		},
		{
			name:        "explicit memory store for tests",
			cfg:         Config{DatabaseType: "sqlite", DatabaseURL: "data.db", FGAStore: "memory"},
			wantStore:   "memory",
			wantURL:     "",
			wantEnabled: true,
		},
		{
			name:        "cockroachdb is NOT auto-mapped (needs explicit store)",
			cfg:         Config{DatabaseType: "cockroachdb", DatabaseURL: "postgres://h/db"},
			wantEnabled: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, url, enabled := tc.cfg.FGAStoreConfig()
			if enabled != tc.wantEnabled {
				t.Fatalf("enabled = %v, want %v", enabled, tc.wantEnabled)
			}
			if !enabled {
				return
			}
			if store != tc.wantStore {
				t.Errorf("store = %q, want %q", store, tc.wantStore)
			}
			if url != tc.wantURL {
				t.Errorf("url = %q, want %q", url, tc.wantURL)
			}
		})
	}
}

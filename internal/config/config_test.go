package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("Default().Validate() = %v", err)
	}
}

// The shipped example file must stay in sync with the built-in defaults.
func TestDefaultFileMatchesDefault(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "extra", "default_config_file", "server.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.Path = ""
	if want := Default(); !reflect.DeepEqual(cfg, want) {
		t.Errorf("default file = %+v\nwant %+v", cfg, want)
	}
}

func TestComposeFileIsValid(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "deploy", "compose", "server.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got := cfg.LoginServer.GameServerAddr(); got != "login:9413" {
		t.Errorf("GameServerAddr = %q, want login:9413", got)
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("Load(missing) = %+v, want Default()", cfg)
	}
	if cfg.Path != "" {
		t.Errorf("Path = %q, want empty for defaults", cfg.Path)
	}
}

func TestLoadSetsPath(t *testing.T) {
	path := writeFile(t, `{"gameservers": []}`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
}

// B11: JSON errors used to be ignored and gave a zero config.
func TestLoadReportsJSONErrors(t *testing.T) {
	tests := map[string]string{
		"syntax":         `{"loginserver": {"host": "x",}}`,
		"wrong type":     `{"loginserver": {"host": 5}}`,
		"unknown field":  `{"loginserver": {"hots": "x"}}`,
		"trailing data":  `{} {}`,
		"empty":          ``,
		"removed cache":  `{"gameservers": [{"id": 1, "cache": {}}]}`,
		"bad duration":   `{"gameservers": [{"id": 1, "scripts": {"eventTimeout": "fast"}}]}`,
		"number timeout": `{"gameservers": [{"id": 1, "scripts": {"eventTimeout": 50}}]}`,
		"unknown script": `{"gameservers": [{"id": 1, "scripts": {"folder": "x"}}]}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := writeFile(t, content)
			_, err := Load(path)
			if err == nil {
				t.Fatal("Load succeeded, want an error")
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error %q does not name the file", err)
			}
		})
	}
}

func TestLoadUnreadableFile(t *testing.T) {
	// A directory cannot be read as a file; this must not fall back to defaults.
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("Load(directory) succeeded, want an error")
	}
}

func TestParseAppliesDefaults(t *testing.T) {
	cfg, err := Parse([]byte(`{"loginserver": {"host": "h"}}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	l := cfg.LoginServer
	if l.ClientListen != DefaultClientListen || l.GameServerListen != DefaultGameServerListen {
		t.Errorf("listen = %q, %q; want defaults", l.ClientListen, l.GameServerListen)
	}
	if !reflect.DeepEqual(l.InternalNetworks, DefaultInternalNetworks) {
		t.Errorf("InternalNetworks = %v, want %v", l.InternalNetworks, DefaultInternalNetworks)
	}

	// Scripts settings left out get their defaults.
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}, {"id": 2, "scripts": {"dir": "/srv/lua", "devReload": true, "eventTimeout": "2s"}}]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := cfg.GameServers[0].Scripts, (Scripts{Dir: DefaultScriptsDir, EventTimeout: DefaultEventTimeout}); got != want {
		t.Errorf("default scripts = %+v, want %+v", got, want)
	}
	if got, want := cfg.GameServers[1].Scripts, (Scripts{Dir: "/srv/lua", DevReload: true, EventTimeout: Duration(2 * time.Second)}); got != want {
		t.Errorf("scripts = %+v, want %+v", got, want)
	}

	// An explicit empty list is kept: no client counts as internal.
	cfg, err = Parse([]byte(`{"loginserver": {"internalNetworks": []}}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if n := cfg.LoginServer.InternalNetworks; n == nil || len(n) != 0 {
		t.Errorf("InternalNetworks = %#v, want empty non-nil", n)
	}
}

func TestParseAcceptsComments(t *testing.T) {
	_, err := Parse([]byte(`{"_comment": "a", "loginserver": {"_comment": "b"}, "gameservers": [{"_comment": "c"}]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
		want   string // substring of the error; empty means valid
	}{
		{"default", func(*Config) {}, ""},
		{"empty host", func(c *Config) { c.LoginServer.Host = "" }, "loginserver.host"},
		{"bad client listen", func(c *Config) { c.LoginServer.ClientListen = "2106" }, "clientListen"},
		{"client listen port 0", func(c *Config) { c.LoginServer.ClientListen = ":0" }, "clientListen"},
		{"bad game server listen", func(c *Config) { c.LoginServer.GameServerListen = ":x" }, "gameServerListen"},
		{"bad network", func(c *Config) { c.LoginServer.InternalNetworks = []string{"10.0.0.0"} }, "internalNetworks"},
		{"login db no name", func(c *Config) { c.LoginServer.Database.Name = "" }, "loginserver.database"},
		{"login db no host", func(c *Config) { c.LoginServer.Database.Host = "" }, "loginserver.database"},
		{"login db bad port", func(c *Config) { c.LoginServer.Database.Port = 70000 }, "loginserver.database"},
		{"login db uri", func(c *Config) {
			c.LoginServer.Database = Database{Name: "x", URI: "mongodb://user:pw@db:27017"}
		}, ""},
		{"login db bad uri", func(c *Config) {
			c.LoginServer.Database = Database{Name: "x", URI: "http://db"}
		}, "loginserver.database"},
		{"id 0", func(c *Config) { c.GameServers[0].ID = 0 }, "gameservers[0].id"},
		{"id 256", func(c *Config) { c.GameServers[0].ID = 256 }, "gameservers[0].id"},
		{"duplicate id", func(c *Config) {
			c.GameServers = append(c.GameServers, c.GameServers[0])
		}, "used twice"},
		{"no name", func(c *Config) { c.GameServers[0].Name = " " }, "gameservers[0].name"},
		{"bad internal ip", func(c *Config) { c.GameServers[0].InternalIP = "localhost" }, "internalIP"},
		{"ipv6 external ip", func(c *Config) { c.GameServers[0].ExternalIP = "::1" }, "externalIP"},
		{"port 0", func(c *Config) { c.GameServers[0].Port = 0 }, "gameservers[0].port"},
		{"game db", func(c *Config) { c.GameServers[0].Database.Name = "" }, "gameservers[0].database"},
		{"no scripts dir", func(c *Config) { c.GameServers[0].Scripts.Dir = "" }, "gameservers[0].scripts.dir"},
		{"zero event timeout", func(c *Config) { c.GameServers[0].Scripts.EventTimeout = 0 }, "scripts.eventTimeout"},
		{"negative event timeout", func(c *Config) {
			c.GameServers[0].Scripts.EventTimeout = Duration(-time.Millisecond)
		}, "scripts.eventTimeout"},
		{"no game servers", func(c *Config) { c.GameServers = nil }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(&cfg)
			err := cfg.Validate()
			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nil", err)
			case tt.want != "" && err == nil:
				t.Fatalf("Validate() = nil, want error containing %q", tt.want)
			case tt.want != "" && !strings.Contains(err.Error(), tt.want):
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	cfg := Default()
	cfg.LoginServer.Host = ""
	cfg.GameServers[0].Port = 0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil")
	}
	for _, want := range []string{"loginserver.host", "gameservers[0].port"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q is missing %q", err, want)
		}
	}
}

func TestMongoURI(t *testing.T) {
	tests := []struct {
		db   Database
		want string
	}{
		{Database{Host: "127.0.0.1", Port: 27017}, "mongodb://127.0.0.1:27017"},
		{Database{Host: "mongo", Port: 1, User: "u", Password: "p@ss"}, "mongodb://u:p%40ss@mongo:1"},
		{Database{Host: "::1", Port: 27017}, "mongodb://[::1]:27017"},
		{Database{URI: "mongodb+srv://x", Host: "ignored"}, "mongodb+srv://x"},
	}
	for _, tt := range tests {
		if got := tt.db.MongoURI(); got != tt.want {
			t.Errorf("%+v.MongoURI() = %q, want %q", tt.db, got, tt.want)
		}
	}
}

func TestAddresses(t *testing.T) {
	l := LoginServer{Host: "login.example", GameServerListen: "0.0.0.0:9500"}
	if got, want := l.GameServerAddr(), "login.example:9500"; got != want {
		t.Errorf("GameServerAddr = %q, want %q", got, want)
	}
	if got, want := (GameServer{Port: 7777}).ListenAddr(), ":7777"; got != want {
		t.Errorf("ListenAddr = %q, want %q", got, want)
	}
}

func TestGameServerByID(t *testing.T) {
	cfg := Default()
	cfg.GameServers = append(cfg.GameServers, GameServer{ID: 5, Name: "Five"})
	if gs, ok := cfg.GameServerByID(5); !ok || gs.Name != "Five" {
		t.Errorf("GameServerByID(5) = %+v, %v", gs, ok)
	}
	if _, ok := cfg.GameServerByID(2); ok {
		t.Error("GameServerByID(2) found a server")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(".l2go", "config", "server.json")) {
		t.Errorf("DefaultPath = %q", path)
	}
}

func TestDurationText(t *testing.T) {
	var d Duration
	if err := d.UnmarshalText([]byte("1m30s")); err != nil || time.Duration(d) != 90*time.Second {
		t.Errorf("UnmarshalText(1m30s) = %v, %v", time.Duration(d), err)
	}
	if err := d.UnmarshalText([]byte("soon")); err == nil {
		t.Error("UnmarshalText(soon) succeeded")
	}
	if b, err := DefaultEventTimeout.MarshalText(); err != nil || string(b) != "50ms" {
		t.Errorf("MarshalText = %q, %v; want 50ms", b, err)
	}
}

func TestErrorsWrap(t *testing.T) {
	_, err := Load(t.TempDir())
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("Load(directory) error %v does not wrap *os.PathError", err)
	}
}

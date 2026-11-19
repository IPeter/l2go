// Package config loads and validates the server configuration file.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Defaults for fields that may be left out of the file.
const (
	DefaultClientListen     = ":2106"
	DefaultGameServerListen = ":9413"
	DefaultScriptsDir       = "scripts"
	DefaultEventTimeout     = Duration(50 * time.Millisecond)
)

// DefaultInternalNetworks are the client networks that get a game server's
// internal IP in the server list.
var DefaultInternalNetworks = []string{"127.0.0.0/8"}

// Config is the whole configuration file. Login and game servers read the
// same file.
type Config struct {
	// Comment is free text for people reading the file. It is ignored.
	Comment     string       `json:"_comment,omitempty"`
	LoginServer LoginServer  `json:"loginserver"`
	GameServers []GameServer `json:"gameservers"`

	// Path is the file the config was loaded from, or empty for the
	// built-in defaults.
	Path string `json:"-"`
}

// LoginServer configures the login server.
type LoginServer struct {
	Comment string `json:"_comment,omitempty"`
	// Host is where game servers reach the login server.
	Host string `json:"host"`
	// ClientListen is the listen address for game clients.
	ClientListen string `json:"clientListen"`
	// GameServerListen is the listen address for game server links.
	GameServerListen string `json:"gameServerListen"`
	// AutoCreate creates an account on the first login with a new username.
	AutoCreate bool `json:"autoCreate"`
	// InternalNetworks lists client networks (CIDR) that get a game server's
	// InternalIP in the server list. Everyone else gets its ExternalIP.
	InternalNetworks []string `json:"internalNetworks"`
	Database         Database `json:"database"`
}

// GameServer configures one game server.
type GameServer struct {
	Comment string `json:"_comment,omitempty"`
	// ID identifies the server in the server list and on the command line
	// (1-255).
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Secret will authenticate the game server to the login server.
	Secret string `json:"secret"` //nolint:gosec // read from the config file, never sent
	// InternalIP is announced to clients from InternalNetworks, ExternalIP
	// to everyone else. Both must be IPv4.
	InternalIP string `json:"internalIP"`
	ExternalIP string `json:"externalIP"`
	// Port is both the listen port and the port announced to clients.
	Port     int      `json:"port"`
	Database Database `json:"database"`
	Options  Options  `json:"options"`
	Scripts  Scripts  `json:"scripts"`
}

// Scripts configures the Lua scripts of a game server.
type Scripts struct {
	Comment string `json:"_comment,omitempty"`
	// Dir is the scripts folder. A relative path is relative to the working
	// directory.
	Dir string `json:"dir"`
	// DevReload reloads the scripts when a file in Dir changes.
	DevReload bool `json:"devReload"`
	// EventTimeout bounds the script handlers of one event, for example
	// "50ms".
	EventTimeout Duration `json:"eventTimeout"`
}

// Duration is a time.Duration written as a string such as "50ms" or "2s".
type Duration time.Duration

// MarshalText writes d like time.Duration.String.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// UnmarshalText parses a time.ParseDuration string.
func (d *Duration) UnmarshalText(b []byte) error {
	v, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

// Options are game server settings shown in the server list.
type Options struct {
	MaxPlayers uint16 `json:"maxPlayers"`
	// Testing servers only accept accounts above player access level.
	Testing bool `json:"testing"`
}

// Database is a MongoDB connection. Either URI, or Host and Port (plus
// optional User and Password) must be set.
type Database struct {
	URI      string `json:"uri,omitempty"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"` //nolint:gosec // read from the config file, never sent
}

// MongoURI returns URI if set, or builds one from the other fields.
func (d Database) MongoURI() string {
	if d.URI != "" {
		return d.URI
	}
	u := url.URL{Scheme: "mongodb", Host: net.JoinHostPort(d.Host, strconv.Itoa(d.Port))}
	if d.User != "" {
		u.User = url.UserPassword(d.User, d.Password)
	}
	return u.String()
}

// GameServerAddr returns the address game servers dial to reach the login
// server: Host with the port of GameServerListen.
func (l LoginServer) GameServerAddr() string {
	_, port, _ := net.SplitHostPort(l.GameServerListen)
	return net.JoinHostPort(l.Host, port)
}

// ListenAddr returns the address the game server listens on for clients.
func (g GameServer) ListenAddr() string {
	return net.JoinHostPort("", strconv.Itoa(g.Port))
}

// GameServerByID returns the game server with the given ID.
func (c Config) GameServerByID(id int) (GameServer, bool) {
	for _, gs := range c.GameServers {
		if gs.ID == id {
			return gs, true
		}
	}
	return GameServer{}, false
}

// Default returns the built-in configuration, used when no file exists.
func Default() Config {
	return Config{
		LoginServer: LoginServer{
			Host:             "127.0.0.1",
			ClientListen:     DefaultClientListen,
			GameServerListen: DefaultGameServerListen,
			AutoCreate:       true,
			InternalNetworks: append([]string(nil), DefaultInternalNetworks...),
			Database: Database{
				Name: "l2go-login",
				Host: "127.0.0.1",
				Port: 27017,
			},
		},
		GameServers: []GameServer{{
			ID:         1,
			Name:       "Bartz",
			Secret:     "CHANGE_ME_PLEASE",
			InternalIP: "127.0.0.1",
			ExternalIP: "192.168.1.2",
			Port:       7777,
			Database: Database{
				Name: "l2go-server",
				Host: "127.0.0.1",
				Port: 27017,
			},
			Options: Options{MaxPlayers: 10000},
			Scripts: Scripts{Dir: DefaultScriptsDir, EventTimeout: DefaultEventTimeout},
		}},
	}
}

// DefaultPath returns ~/.l2go/config/server.json.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: find home directory: %w", err)
	}
	return filepath.Join(home, ".l2go", "config", "server.json"), nil
}

// Load reads the config file at path. If the file does not exist it returns
// Default() with an empty Path. Any other problem, including invalid JSON
// and unknown fields, is an error. Load does not call Validate.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return Config{}, fmt.Errorf("config: %s: %w", path, err)
	}
	cfg.Path = path
	return cfg, nil
}

// Parse decodes a config file. Fields left out of the file stay at their
// zero value, except the listen addresses, internal networks, scripts
// folder and event timeout, which get their defaults.
func Parse(data []byte) (Config, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("parse: unexpected data after the top-level object")
	}
	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	l := &c.LoginServer
	if l.ClientListen == "" {
		l.ClientListen = DefaultClientListen
	}
	if l.GameServerListen == "" {
		l.GameServerListen = DefaultGameServerListen
	}
	if l.InternalNetworks == nil {
		l.InternalNetworks = append([]string(nil), DefaultInternalNetworks...)
	}
	for i := range c.GameServers {
		s := &c.GameServers[i].Scripts
		if s.Dir == "" {
			s.Dir = DefaultScriptsDir
		}
		if s.EventTimeout == 0 {
			s.EventTimeout = DefaultEventTimeout
		}
	}
}

// Validate checks the whole configuration and reports every problem found.
func (c Config) Validate() error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	l := c.LoginServer
	if l.Host == "" {
		add("loginserver.host is empty")
	}
	if err := checkListen(l.ClientListen); err != nil {
		add("loginserver.clientListen: %w", err)
	}
	if err := checkListen(l.GameServerListen); err != nil {
		add("loginserver.gameServerListen: %w", err)
	}
	for _, n := range l.InternalNetworks {
		if _, err := netip.ParsePrefix(n); err != nil {
			add("loginserver.internalNetworks: %w", err)
		}
	}
	if err := l.Database.validate(); err != nil {
		add("loginserver.database: %w", err)
	}

	seen := make(map[int]bool)
	for i, gs := range c.GameServers {
		where := fmt.Sprintf("gameservers[%d]", i)
		if gs.ID < 1 || gs.ID > 255 {
			add("%s.id: %d is not in 1-255", where, gs.ID)
		} else if seen[gs.ID] {
			add("%s.id: %d is used twice", where, gs.ID)
		}
		seen[gs.ID] = true
		if strings.TrimSpace(gs.Name) == "" {
			add("%s.name is empty", where)
		}
		if err := checkIPv4(gs.InternalIP); err != nil {
			add("%s.internalIP: %w", where, err)
		}
		if err := checkIPv4(gs.ExternalIP); err != nil {
			add("%s.externalIP: %w", where, err)
		}
		if err := checkPort(gs.Port); err != nil {
			add("%s.port: %w", where, err)
		}
		if err := gs.Database.validate(); err != nil {
			add("%s.database: %w", where, err)
		}
		if strings.TrimSpace(gs.Scripts.Dir) == "" {
			add("%s.scripts.dir is empty", where)
		}
		if gs.Scripts.EventTimeout <= 0 {
			add("%s.scripts.eventTimeout: %v is not positive", where, time.Duration(gs.Scripts.EventTimeout))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("config: invalid:\n%w", err)
	}
	return nil
}

func (d Database) validate() error {
	if d.Name == "" {
		return errors.New("name is empty")
	}
	if d.URI != "" {
		if !strings.HasPrefix(d.URI, "mongodb://") && !strings.HasPrefix(d.URI, "mongodb+srv://") {
			return errors.New("uri must start with mongodb:// or mongodb+srv://")
		}
		return nil
	}
	if d.Host == "" {
		return errors.New("host is empty (set host and port, or uri)")
	}
	return checkPort(d.Port)
}

func checkPort(p int) error {
	if p < 1 || p > 65535 {
		return fmt.Errorf("port %d is not in 1-65535", p)
	}
	return nil
}

func checkListen(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port %q is not a number", port)
	}
	return checkPort(p)
}

func checkIPv4(s string) error {
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return err
	}
	if !ip.Unmap().Is4() {
		return fmt.Errorf("%s is not an IPv4 address", s)
	}
	return nil
}

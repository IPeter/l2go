package config

import (
	"strings"
	"testing"
)

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := env[name]
		return v, ok
	}
}

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name string
		in   string
		env  map[string]string
		want string
	}{
		{"set", `"${IP}"`, map[string]string{"IP": "192.168.1.87"}, `"192.168.1.87"`},
		{"default when unset", `"${IP:-127.0.0.1}"`, nil, `"127.0.0.1"`},
		{"default when empty", `"${IP:-127.0.0.1}"`, map[string]string{"IP": ""}, `"127.0.0.1"`},
		{"set wins over default", `"${IP:-127.0.0.1}"`, map[string]string{"IP": "10.0.0.5"}, `"10.0.0.5"`},
		{"empty default", `"${X:-}"`, nil, `""`},
		{"several", `["${A}", "${B:-b}"]`, map[string]string{"A": "a"}, `["a", "b"]`},
		{"value escaped for JSON", `"${S}"`, map[string]string{"S": `quo"te\back`}, `"quo\"te\\back"`},
		{"plain dollar untouched", `"pa$$word $X"`, nil, `"pa$$word $X"`},
		{"no references", `{"a": 1}`, nil, `{"a": 1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandEnv([]byte(tt.in), lookupFrom(tt.env))
			if err != nil {
				t.Fatalf("expandEnv: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("expandEnv(%s) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandEnvMissing(t *testing.T) {
	_, err := expandEnv([]byte(`["${A}", "${B}", "${C:-c}"]`), lookupFrom(nil))
	if err == nil {
		t.Fatal("expandEnv succeeded, want an error for unset variables")
	}
	for _, name := range []string{"A", "B"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
	if strings.Contains(err.Error(), "C") {
		t.Errorf("error %q names C, which has a default", err)
	}
}

func TestParseUsesEnv(t *testing.T) {
	data := []byte(`{
	  "loginserver": {"host": "login", "database": {"name": "l", "host": "mongo", "port": 27017}},
	  "gameservers": [{
	    "id": 1, "name": "Bartz", "secret": "s", "port": 7777,
	    "internalIP": "${L2GO_PUBLIC_IP:-127.0.0.1}",
	    "externalIP": "${L2GO_PUBLIC_IP:-127.0.0.1}",
	    "database": {"name": "g", "host": "mongo", "port": 27017}
	  }]
	}`)

	cfg, err := parse(data, lookupFrom(map[string]string{"L2GO_PUBLIC_IP": "192.168.1.87"}))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	gs := cfg.GameServers[0]
	if gs.InternalIP != "192.168.1.87" || gs.ExternalIP != "192.168.1.87" {
		t.Errorf("IPs = %s / %s, want 192.168.1.87", gs.InternalIP, gs.ExternalIP)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}

	cfg, err = parse(data, lookupFrom(nil))
	if err != nil {
		t.Fatalf("parse without env: %v", err)
	}
	if got := cfg.GameServers[0].ExternalIP; got != "127.0.0.1" {
		t.Errorf("ExternalIP without env = %s, want the default 127.0.0.1", got)
	}
}

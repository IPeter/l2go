package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// envRef matches ${NAME} and ${NAME:-default}.
var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

// expandEnv replaces the environment references in a config file. The
// values are escaped for use inside a JSON string.
func expandEnv(data []byte, lookup func(string) (string, bool)) ([]byte, error) {
	var missing []string
	out := envRef.ReplaceAllFunc(data, func(ref []byte) []byte {
		m := envRef.FindSubmatch(ref)
		name, hasDefault, def := string(m[1]), len(m[2]) > 0, string(m[3])
		v, ok := lookup(name)
		if !ok || v == "" {
			if !hasDefault {
				missing = append(missing, name)
				return ref
			}
			v = def
		}
		quoted, _ := json.Marshal(v) // a string always marshals
		return quoted[1 : len(quoted)-1]
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("environment variable not set: %s", strings.Join(missing, ", "))
	}
	return out, nil
}

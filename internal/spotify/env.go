package spotify

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Credentials struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

func LoadCredentialsFromEnv() (Credentials, error) {
	cid, err := ReadSecretEnvOrFile("SPOTIFY_CLIENT_ID")
	if err != nil {
		return Credentials{}, err
	}
	sec, err := ReadSecretEnvOrFile("SPOTIFY_CLIENT_SECRET")
	if err != nil {
		return Credentials{}, err
	}
	rt, err := ReadSecretEnvOrFile("SPOTIFY_REFRESH_TOKEN")
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{ClientID: cid, ClientSecret: sec, RefreshToken: rt}, nil
}

func ReadSecretEnvOrFile(key string) (string, error) {
	// _FILE override is handy in some environments.
	if fp := os.Getenv(key + "_FILE"); fp != "" {
		b, err := os.ReadFile(expandHome(fp))
		if err != nil {
			return "", fmt.Errorf("read %s_FILE: %w", key, err)
		}
		v := strings.TrimSpace(string(b))
		if v == "" {
			return "", fmt.Errorf("%s_FILE is empty", key)
		}
		return v, nil
	}

	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("missing %s", key)
	}

	// If it looks like a path, treat it as a file pointer.
	if strings.Contains(v, "/") || strings.HasPrefix(v, "~") {
		b, err := os.ReadFile(expandHome(v))
		if err != nil {
			return "", fmt.Errorf("read %s file: %w", key, err)
		}
		vv := strings.TrimSpace(string(b))
		if vv == "" {
			return "", fmt.Errorf("%s file is empty: %s", key, v)
		}
		return vv, nil
	}

	return v, nil
}

func expandHome(p string) string {
	if p == "" {
		return p
	}
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

var ErrNoActivePlayback = errors.New("no active playback")

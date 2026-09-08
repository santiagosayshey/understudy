package main

import (
	"flag"
	"os"
)

// common holds the settings every command that reads the configuration or
// talks to Plex needs. Each flag falls back to UNDERSTUDY_<NAME>.
type common struct {
	config    string
	portraits string
	plexURL   string
	plexToken string
}

func (c *common) bind(fs *flag.FlagSet) {
	fs.StringVar(&c.config, "config", envOr("UNDERSTUDY_CONFIG", "/config/configuration.yml"), "path to configuration.yml")
	fs.StringVar(&c.portraits, "portraits", envOr("UNDERSTUDY_PORTRAITS", "/portraits"), "directory the configuration's images live in")
	fs.StringVar(&c.plexURL, "plex-url", envOr("UNDERSTUDY_PLEX_URL", ""), "Plex server address, or a proxy in front of it")
	fs.StringVar(&c.plexToken, "plex-token", envOr("UNDERSTUDY_PLEX_TOKEN", ""), "Plex token; omit when the address injects it")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

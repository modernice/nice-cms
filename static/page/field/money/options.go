package money

import "github.com/bojanz/currency"

// Config is the Money field configuration.
type Config struct {
	Localization map[string]currency.Amount
}

// Option configures a Money field.
type Option func(*Config)

// Configure configures a Money field.
func Configure(opts ...Option) Config {
	cfg := Config{Localization: make(map[string]currency.Amount)}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Localize returns an Option that localizes a Money field.
func Localize(val currency.Amount, locales ...string) Option {
	return func(cfg *Config) {
		for _, locale := range locales {
			cfg.Localization[locale] = val
		}
	}
}

package meta

// Config is the Meta field configuration.
type Config struct {
	Localization map[string]Data
}

// Option is a Meta option.
type Option func(*Config)

// Configure configures a Meta field.
func Configure(opts ...Option) Config {
	cfg := Config{Localization: make(map[string]Data)}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Localize returns an Option that localizes a Meta field.
func Localize(val Data, locales ...string) Option {
	return func(cfg *Config) {
		for _, locale := range locales {
			cfg.Localization[locale] = val
		}
	}
}

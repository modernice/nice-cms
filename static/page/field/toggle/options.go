package toggle

// Config is the Toggle field configuration.
type Config struct {
	Localization map[string]bool
}

// Option is a Toggle option.
type Option func(*Config)

// Configure configures a Toggle field.
func Configure(opts ...Option) Config {
	cfg := Config{Localization: make(map[string]bool)}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Localize returns an Option that localizes a Toggle field.
func Localize(val bool, locales ...string) Option {
	return func(cfg *Config) {
		for _, locale := range locales {
			cfg.Localization[locale] = val
		}
	}
}

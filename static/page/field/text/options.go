package text

// Config is the Text field configuration.
type Config struct {
	Localization map[string]string
}

// Option is a Text option.
type Option func(*Config)

// Configure configures a Text field.
func Configure(opts ...Option) Config {
	cfg := Config{Localization: make(map[string]string)}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Localize returns an Option that localizes a Text field.
func Localize(val string, locales ...string) Option {
	return func(cfg *Config) {
		for _, locale := range locales {
			cfg.Localization[locale] = val
		}
	}
}

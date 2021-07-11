package float

// Config is the Float field configuration.
type Config struct {
	Localization map[string]float64
}

// Option is a Float option.
type Option func(*Config)

// Configure configures a Float field.
func Configure(opts ...Option) Config {
	cfg := Config{Localization: make(map[string]float64)}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Localize returns an Option that localizes a Float field.
func Localize(val float64, locales ...string) Option {
	return func(cfg *Config) {
		for _, locale := range locales {
			cfg.Localization[locale] = val
		}
	}
}

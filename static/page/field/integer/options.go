package integer

// Config is the Int field configuration.
type Config struct {
	Localization map[string]int
}

// Option is an Int option.
type Option func(*Config)

// Configure configures an Int field.
func Configure(opts ...Option) Config {
	cfg := Config{Localization: make(map[string]int)}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Localize return an Option that localizes an Int field.
func Localize(val int, locales ...string) Option {
	return func(cfg *Config) {
		for _, locale := range locales {
			cfg.Localization[locale] = val
		}
	}
}

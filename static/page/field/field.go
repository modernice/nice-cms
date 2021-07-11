package field

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bojanz/currency"
	"github.com/modernice/cms/static/page/field/float"
	"github.com/modernice/cms/static/page/field/integer"
	"github.com/modernice/cms/static/page/field/meta"
	"github.com/modernice/cms/static/page/field/money"
	"github.com/modernice/cms/static/page/field/text"
	"github.com/modernice/cms/static/page/field/toggle"
)

// Built-in field types
const (
	Text   = Type("text")
	Toggle = Type("toggle")
	Int    = Type("integer")
	Float  = Type("float")
	Money  = Type("money")
	Meta   = Type("meta")
)

// Type is a field type.
type Type string

// Field is a field of a page.
type Field struct {
	Name    string
	Type    Type
	Values  map[string]string
	Guarded bool
}

// Option is a Field option.
type Option func(*Field)

// Localize returns an Option that localizes a Field. val is used as the value
// for the given locales.
func Localize(val string, locales ...string) Option {
	return func(f *Field) {
		for _, locale := range locales {
			f.Values[locale] = val
		}
	}
}

// Guarded returns an Option that makes a Field guarded. A guarded Field cannot
// be removed from a Page.
func Guarded() Option {
	return func(f *Field) {
		f.Guarded = true
	}
}

// New returns a new Field with the given name, Type and default value.
func New(name string, typ Type, defaultValue string, opts ...Option) Field {
	f := Field{
		Name:   name,
		Type:   typ,
		Values: make(map[string]string),
	}
	for _, opt := range opts {
		opt(&f)
	}
	f.Values[""] = defaultValue
	return f
}

// Value returns the value for the given locale as a string.
func (f Field) Value(locale string) string {
	if val, ok := f.Values[locale]; ok {
		return val
	}
	return f.Values[""]
}

// NewText returns a Text field.
func NewText(name, defaultValue string, opts ...text.Option) Field {
	cfg := text.Configure(opts...)
	var fieldOpts []Option
	for locale, val := range cfg.Localization {
		fieldOpts = append(fieldOpts, Localize(val, locale))
	}
	f := New(name, Text, defaultValue, fieldOpts...)
	return f
}

// NewToggle returns a Toggle field.
func NewToggle(name string, defaultValue bool, opts ...toggle.Option) Field {
	cfg := toggle.Configure(opts...)
	var fieldOpts []Option
	for locale, val := range cfg.Localization {
		fieldOpts = append(fieldOpts, Localize(boolToString(val), locale))
	}
	f := New(name, Toggle, boolToString(defaultValue), fieldOpts...)
	return f
}

func boolToString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// NewInt returns an Int field.
func NewInt(name string, defaultValue int, opts ...integer.Option) Field {
	cfg := integer.Configure(opts...)
	var fieldOpts []Option
	for locale, val := range cfg.Localization {
		fieldOpts = append(fieldOpts, Localize(strconv.Itoa(val), locale))
	}
	return New(name, Int, strconv.Itoa(defaultValue), fieldOpts...)
}

// NewFloat returns a Float field.
func NewFloat(name string, defaultValue float64, opts ...float.Option) Field {
	cfg := float.Configure(opts...)
	var fieldOpts []Option
	for locale, val := range cfg.Localization {
		fieldOpts = append(fieldOpts, Localize(floatToString(val), locale))
	}
	return New(name, Float, floatToString(defaultValue), fieldOpts...)
}

func floatToString(v float64) string {
	return fmt.Sprintf("%f", v)
}

// NewMoney returns a Money field.
func NewMoney(name string, defaultValue currency.Amount, opts ...money.Option) Field {
	cfg := money.Configure(opts...)
	var fieldOpts []Option
	for locale, val := range cfg.Localization {
		fieldOpts = append(fieldOpts, Localize(moneyToString(val), locale))
	}
	return New(name, Money, moneyToString(defaultValue), fieldOpts...)
}

var defaultMoneyFormatter = currency.NewFormatter(currency.NewLocale("en"))

func moneyToString(m currency.Amount) string {
	return defaultMoneyFormatter.Format(m)
}

// NewMeta returns a Meta field.
func NewMeta(name string, defaultValue meta.Data, opts ...meta.Option) Field {
	cfg := meta.Configure(opts...)
	var fieldOpts []Option
	for locale, val := range cfg.Localization {
		fieldOpts = append(fieldOpts, Localize(metaToString(val), locale))
	}
	return New(name, Meta, metaToString(defaultValue), fieldOpts...)
}

func metaToString(data meta.Data) string {
	b, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Errorf("marshal metadata: %w", err))
	}
	return string(b)
}

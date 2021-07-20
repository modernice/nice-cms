package field

import (
	"fmt"
	"strconv"

	"github.com/bojanz/currency"
	"github.com/modernice/nice-cms/internal/money"
	"github.com/modernice/nice-cms/static/page/metadata"
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
func NewText(name, defaultValue string, opts ...Option) Field {
	return New(name, Text, defaultValue, opts...)
}

// NewToggle returns a Toggle field.
func NewToggle(name string, defaultValue bool, opts ...Option) Field {
	return New(name, Toggle, boolToString(defaultValue), opts...)
}

func boolToString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// NewInt returns an Int field.
func NewInt(name string, defaultValue int, opts ...Option) Field {
	return New(name, Int, strconv.Itoa(defaultValue), opts...)
}

// NewFloat returns a Float field.
func NewFloat(name string, defaultValue float64, opts ...Option) Field {
	return New(name, Float, floatToString(defaultValue), opts...)
}

func floatToString(v float64) string {
	return fmt.Sprintf("%f", v)
}

// NewMoney returns a Money field.
func NewMoney(name string, defaultValue currency.Amount, opts ...Option) Field {
	return New(name, Money, money.Format(defaultValue), opts...)
}

// NewMeta returns a Meta field.
func NewMeta(name string, defaultValue metadata.Data, opts ...Option) Field {
	str, err := defaultValue.JSON()
	if err != nil {
		panic(err)
	}
	return New(name, Meta, str, opts...)
}

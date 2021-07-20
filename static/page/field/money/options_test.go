package money_test

import (
	"fmt"
	"testing"

	"github.com/bojanz/currency"
	"github.com/modernice/nice-cms/static/page/field"
	"github.com/modernice/nice-cms/static/page/field/money"
)

func TestLocalize(t *testing.T) {
	amount, _ := currency.NewAmount("30.25", "USD")
	otherAmount, _ := currency.NewAmount("49.95", "EUR")
	f := field.NewMoney("foo", amount, money.Localize(otherAmount, "de", "ch"))

	formatter := currency.NewFormatter(currency.NewLocale("en"))

	tests := map[string]string{
		"":    formatter.Format(amount),
		" ":   formatter.Format(amount),
		"en":  formatter.Format(amount),
		"abc": formatter.Format(amount),
		"de":  formatter.Format(otherAmount),
		"ch":  formatter.Format(otherAmount),
	}

	for locale, want := range tests {
		got := f.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v\n\n%s", locale, want, got, fmt.Sprintf("%#v", f))
		}
	}
}

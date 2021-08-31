package money_test

import (
	"fmt"
	"testing"

	"github.com/modernice/nice-cms/static/page/field"
	moneyfield "github.com/modernice/nice-cms/static/page/field/money"
	"github.com/radical-app/money"
	"github.com/radical-app/money/moneyfmt"
)

func TestLocalize(t *testing.T) {
	amount := money.USD(3025)
	otherAmount := money.EUR(4995)
	f := field.NewMoney("foo", amount, moneyfield.Localize(otherAmount, "de", "ch"))

	tests := map[string]string{
		"":    moneyfmt.MustDisplay(amount, "en"),
		" ":   moneyfmt.MustDisplay(amount, "en"),
		"en":  moneyfmt.MustDisplay(amount, "en"),
		"abc": moneyfmt.MustDisplay(amount, "abc"),
		"de":  moneyfmt.MustDisplay(otherAmount, "de"),
		"ch":  moneyfmt.MustDisplay(otherAmount, "ch"),
	}

	for locale, want := range tests {
		got := f.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v\n\n%s", locale, want, got, fmt.Sprintf("%#v", f))
		}
	}
}

package money

import "github.com/bojanz/currency"

var DefaultFormatter = currency.NewFormatter(currency.NewLocale("en"))

func Format(amount currency.Amount) string {
	return DefaultFormatter.Format(amount)
}

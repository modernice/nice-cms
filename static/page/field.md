# Field

A `Field` is a field of a [page](./page.md). Fields can be of different types
that are identified by a (unique) string.

## Field config

The `FieldConfig` provides (de)serialization of field values for the different
field types. A `Page` uses the field configuration to parse string values to
(compact) byte slices and format those byte slices into strings for
representation.

### Configure a field

```go
package example

func configureField() {
	var cfg *page.FieldConfig

	cfg.Configure(
		"text",
		page.WithFieldFormatterFunc(func(val []byte) string { return string(val) }), // configure formatter for text fields
	)

	cfg.Configure(
		"toggle",
		page.WithFieldFormatterFunc(func(val []byte) string {
			if val == '1' { // single quotes
				return true
			}
			return false
		}),

		// A FieldParser takes the user input in string format and tries to return
		// the byte value for that value. In this example, the toggle accepts only
		// "1" and "0" as values, representing "on" and "off".
		page.WithFieldParserFunc(func(val string) ([]byte, error) {
			switch val {
			case "1":
				return []byte{'1'}, nil
			case "0":
				return []byte{'0'}, nil
			default:
				return nil, fmt.Errorf("invalid value %q, expected one of %v", val, "1", "0")
			}
		}),
	)
}
```

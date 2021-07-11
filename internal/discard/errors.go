package discard

func Errors(errs <-chan error, use ...func(error)) {
	for err := range errs {
		for _, fn := range use {
			fn(err)
		}
	}
}

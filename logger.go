package httpclient

type logger interface {
	print(string)
}

type loggerFunc func(string)

func (fn loggerFunc) print(msg string) {
	fn(msg)
}

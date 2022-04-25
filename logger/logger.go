package logger

type Logger interface {
	Info(...any)
	Debug(...any)
	Error(...any)
}

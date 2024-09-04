package base

type Base interface {
}

type Service struct {
	Base Base
}

func New() *Service {
	return &Service{}
}

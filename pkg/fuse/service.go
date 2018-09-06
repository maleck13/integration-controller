package fuse

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) AddAMQPConnection(user, pass, messageHost string) error {
	return nil
}

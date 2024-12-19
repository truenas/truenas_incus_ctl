package core

type MockSession struct {
    a int
}

func (s *MockSession) Login() error {
    return nil
}

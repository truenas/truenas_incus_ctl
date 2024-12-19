package core

type Session interface {
    Login() error
}

func GetApi() Session {
    if (true) {
        return &RealSession{}
    }
    return &MockSession{}
}

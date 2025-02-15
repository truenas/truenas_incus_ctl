package core

import (
	"errors"
	"strings"
	"encoding/json"
	"truenas/truenas_incus_ctl/truenas_api"
)

type RealSession struct {
	HostUrl string
	ApiKey string
	client *truenas_api.Client
}

func (s *RealSession) Login() error {
	if s.client != nil {
		_ = s.Close()
	}

	if s.HostUrl == "" || s.ApiKey == "" {
		return errors.New("--url and --api-key were not provided")
	}

	client, err := truenas_api.NewClient(s.HostUrl, strings.HasPrefix(s.HostUrl, "wss://"))
	if err != nil {
		return errors.New("Failed to create client: " + err.Error())
	}

	err = client.Login("", "", s.ApiKey)
	if err != nil {
		client.Close()
		return errors.New("Client login failed: " + err.Error())
	}

	s.client = client
	return nil
}

func (s *RealSession) CallRaw(method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
	return s.client.Call(method, timeoutStr, params)
}

func (s *RealSession) Close() error {
	var err error
	if s.client != nil {
		err = s.client.Close()
		s.client = nil
	}
	return err
}

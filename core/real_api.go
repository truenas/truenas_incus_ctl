package core

import (
	"errors"
	"os"
	"strings"
	"encoding/json"
	"truenas/truenas-admin/truenas_api"
)

type RealSession struct {
	HostUrl string
	ApiKey string
	KeyFileName string
	client *truenas_api.Client
}

func (s *RealSession) Login() error {
	if s.client != nil {
		_ = s.Close()
	}

	var err error
	if s.HostUrl == "" || s.ApiKey == "" {
		s.HostUrl, s.ApiKey, err = loadHostAndKey(s.KeyFileName)
		if err != nil {
			return err
		}
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

func loadHostAndKey(fileName string) (string, string, error) {
	if fileName == "" {
		return "", "", errors.New("Could not open key file: no filename was given")
	}

	contents, err := os.ReadFile(fileName)
	if err != nil {
		return "", "", errors.New("Could not open key file: \"" + fileName + "\" does not exist")
	}

	lines := strings.Split(string(contents), "\n")
	if len(lines) < 2 {
		return "", "", errors.New("Failed to parse key file: the first line must be the server URL, and the second line must be the API key")
	}

	return lines[0], lines[1], nil
}

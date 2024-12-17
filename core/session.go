package core

import (
    "os"
    "fmt"
    "strings"
    "truenas/admin-tool/truenas_api"
)

type Session struct {
    hostUrl string
    apiKey string
}

func Login() (*Session, error) {
    var err error
    s := Session{}
    s.hostUrl, s.apiKey, err = loadHostAndKey()
    if err != nil {
        return nil, err
    }

    client, err := truenas_api.NewClient(s.hostUrl, true)
    if err != nil {
        fmt.Println("Failed to create client:", err)
        return nil, err
    }

    err = client.Login("", "", s.apiKey)
    if err != nil {
        fmt.Println("client.Login failed:", err)
        return nil, err
    }

    return &s, nil
}

func loadHostAndKey() (string, string, error) {
    fileName := "key.txt"
    contents, err := os.ReadFile(fileName)
    if err != nil {
        fmt.Println("Could not open", fileName)
        return "", "", err
    }

    lines := strings.Split(string(contents), "\n")
    if len(lines) < 2 {
        fmt.Println(lines[0])
        fmt.Println(
            "Failed to parse login config\n" +
            "The first line must be the server URL, and the second line must be the API key",
        )
        return "", "", err
    }

    return lines[0], lines[1], nil
}

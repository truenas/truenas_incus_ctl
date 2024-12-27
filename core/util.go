package core

import (
    "errors"
    "strings"
)

func EncloseWith(original string, ends string) (string, error) {
    if strings.Index(original, ends) >= 0 {
        return "", errors.New("string already contains '" + ends + "'")
    }
    return ends + original + ends, nil
}

//go:build !linux

package core

import "fmt"

func WaitForCreatedDeletedFiles(directory string, onFileEvent func(string, bool, bool) bool) error {
	return fmt.Errorf("inotify not supported on this platform")
}

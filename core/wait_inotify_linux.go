//go:build linux

package core

import (
	"fmt"
	"syscall"
)

func WaitForCreatedDeletedFiles(directory string, onFileEvent func(string, bool, bool) bool) error {
	fdInotify, err := syscall.InotifyInit()
	if err != nil {
		return fmt.Errorf("syscall.InotifyInit: %v", err)
	}
	defer syscall.Close(fdInotify)

	flagsInterested := uint32(syscall.IN_CREATE | syscall.IN_DELETE | syscall.IN_DELETE_SELF)
	watchDesc, err := syscall.InotifyAddWatch(fdInotify, directory, flagsInterested)
	if err != nil {
		return fmt.Errorf("syscall.InotifyAddWatch: %v", err)
	}
	defer syscall.InotifyRmWatch(fdInotify, uint32(watchDesc)) // why is the type uint32 here?

	var prevName string
	wasCreate := false
	wasDelete := false
	buf := make([]byte, 4096)

	for true {
		if onFileEvent(prevName, wasCreate, wasDelete) {
			break
		}
		prevName = ""
		wasCreate = false

		nRead := 0
		nRead, err = syscall.Read(fdInotify, buf)
		if err != nil {
			return fmt.Errorf("syscall.Read fdInotify: %v (read %d bytes)", err, nRead)
		}

		nameLen := int(buf[12]) | (int(buf[13]) << 8) | (int(buf[14]) << 16) | (int(buf[15]) << 24)
		if nameLen < 0 {
			return fmt.Errorf("inotify event: invalid name length %d", nameLen)
		}
		if nameLen == 0 {
			//fmt.Println("name was empty")
			continue
		}

		name := string(buf[16 : 16+nameLen])
		for bytePos, codePoint := range name {
			if codePoint == 0 {
				if bytePos == 0 {
					name = ""
					break
				}
				name = name[0:bytePos]
				break
			}
		}

		if len(name) == 0 {
			continue
		}

		mask := uint32(buf[4]) | (uint32(buf[5]) << 8) | (uint32(buf[6]) << 16) | (uint32(buf[7]) << 24)
		wasCreate = (mask & syscall.IN_CREATE) != 0
		wasDelete = (mask & (syscall.IN_DELETE | syscall.IN_DELETE_SELF)) != 0
		prevName = name
	}

	return nil
}

package core

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

func DeleteSnakeKebab(dict map[string]string, key string) {
	delete(dict, key)
	delete(dict, strings.ReplaceAll(key, "-", "_"))
	delete(dict, strings.ReplaceAll(key, "_", "-"))
}

func IdentifyObject(obj string) (string, string) {
	if obj == "" {
		return "", ""
	} else if _, errNotNumber := strconv.Atoi(obj); errNotNumber == nil {
		return "id", obj
	} else if obj[0] == '/' {
		return "share", obj
	} else if obj[0] == '@' {
		return "snapshot_only", obj[1:]
	} else if pos := strings.Index(obj, "@"); pos >= 1 {
		if pos == len(obj)-1 {
			return "error", obj
		}
		return "snapshot", obj
	} else if pos := strings.LastIndex(obj, "/"); pos >= 1 {
		if pos == len(obj)-1 {
			return IdentifyObject(obj[0:len(obj)-1])
		}
		return "dataset", obj
	}
	return "pool", obj
}

func GetHostNameFromApiUrl(urlString string) string {
	hostname := urlString
	if strings.Contains(urlString, "://") {
		parsed, err := url.Parse(urlString)
		if err == nil {
			hostname = parsed.Hostname()
			port := parsed.Port()
			if port != "" {
				hostname = hostname + ":" + port
			}
		}
	}
	return hostname
}

func GetApiUrlFromHostName(hostname string) string {
	if strings.Contains(hostname, "://") {
		parsed, err := url.Parse(hostname)
		if err == nil {
			return parsed.String()
		}
	}
	return "wss://" + hostname + "/api/current"
}

func StripPort(hostname string) string {
	parts := strings.Split(hostname, ":")
	if len(parts) == 1 {
		return hostname
	}
	return strings.Join(parts[0:len(parts)-1], ":")
}

func ResolvedIpv4OrVerbatim(hostname string) string {
	dotCount := 0
	digitCount := 0
	for _, c := range hostname {
		if c == '.' {
			if digitCount == 0 {
				break
			}
			digitCount = 0
			dotCount++
		} else if c >= '0' && c <= '9' {
			digitCount++
		} else {
			break
		}
	}
	if dotCount == 3 && digitCount > 0 {
		return hostname
	}

	ipAddrs, err := net.LookupIP(hostname)
	if err != nil {
		return hostname
	}
	return ipAddrs[0].String()
}

func IsLocal() bool {
	st, err := os.Stat("/var/run/middleware/middlewared.sock")
	return err == nil && (st.Mode() & fs.ModeSocket) != 0
}

func StringRepeated(str string, count int) []string {
	if count <= 0 {
		return nil
	}
	arr := make([]string, count)
	for i := 0; i < count; i++ {
		arr[i] = str
	}
	return arr
}

func AppendIfMissing[T comparable](arr []T, value T) []T {
	for _, elem := range arr {
		if elem == value {
			return arr
		}
	}
	return append(arr, value)
}

func MakeErrorFromList(errorList []error) error {
	if len(errorList) == 0 {
		return nil
	}

	var combinedErrMsg strings.Builder
	for _, e := range errorList {
		combinedErrMsg.WriteString("\n")
		combinedErrMsg.WriteString(e.Error())
	}

	return errors.New(combinedErrMsg.String())
}

func GetKeysSorted[T any](dict map[string]T) []string {
	var keys []string
	size := len(dict)
	if size > 0 {
		keys = make([]string, 0, size)
		for k, _ := range dict {
			keys = append(keys, k)
		}
		slices.Sort(keys)
	}
	return keys
}

func WaitForFilesToAppear(directory string, onFileAppeared func(string, bool)bool) error {
	fdInotify, err := syscall.InotifyInit()
	if err != nil {
		return fmt.Errorf("syscall.InotifyInit: %v", err)
	}
	defer syscall.Close(fdInotify)

	flagsInterested := uint32(syscall.IN_CREATE)
	watchDesc, err := syscall.InotifyAddWatch(fdInotify, directory, flagsInterested)
	if err != nil {
		return fmt.Errorf("syscall.InotifyAddWatch: %v", err)
	}
	defer syscall.InotifyRmWatch(fdInotify, uint32(watchDesc)) // why is the type uint32 here?

	var prevName string
	wasCreate := false
	buf := make([]byte, 4096)

	for true {
		if onFileAppeared(prevName, wasCreate) {
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

		name := string(buf[16:16+nameLen])
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
		prevName = name
	}

	return nil
}

func MakeHashedString(input string, length int) string {
	var h []byte
	var maxBits int
	bitsNeeded := length * 5

	if bitsNeeded > 256 {
		h512 := sha512.Sum512([]byte(input))
		h = h512[:]
		maxBits = 512
	} else {
		h256 := sha256.Sum256([]byte(input))
		h = h256[:]
		maxBits = 256
	}

	var builder strings.Builder
	for pos := 0; pos < bitsNeeded && pos < maxBits - 4; pos += 5 {
		data1 := h[pos/8] << (pos%8)
		data2 := h[(pos+4)/8] >> (7-((pos+4)%8))
		v := uint32((data1 >> 3) | data2) & 0x1f
		inc := uint32(0x30)
		if v >= 10 {
			inc = 0x57
		}
		builder.WriteByte(byte(inc + v))
	}
	return builder.String()
}

func ParseSizeString(str string) (int64, error) {
	if str == "" || str[0] < '0' || str[0] > '9' {
		return 0, fmt.Errorf("size was not a number")
	}
	var multiplier int64
	var whole int64
	var frac int64
	nFracDigits := 0
	isFrac := false

outer_loop:
	for _, c := range str {
		for j, unit := range "KMGTP" {
			if c == unit || c == unit + 0x20 {
				if multiplier != 0 {
					return 0, fmt.Errorf("invalid size units in \"" + str + "\"")
				}
				multiplier = int64(1) << (10 * (j + 1))
				continue outer_loop
			}
		}
		if c == '.' {
			isFrac = true
		} else if c >= '0' && c <= '9' {
			if isFrac {
				frac = frac * int64(10) + int64(c - '0')
				nFracDigits++
			} else {
				whole = whole * int64(10) + int64(c - '0')
			}
		} else if c != 'B' && c != 'b' && c != 'I' && c != 'i' && c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return 0, fmt.Errorf("unrecognized character '" + string(c) + "' in \"" + str + "\"")
		}
	}

	if multiplier == 0 {
		multiplier = 1
	}
	fracMult := float64(frac) * math.Pow10(-nFracDigits) * float64(multiplier)
	return whole * multiplier + int64(fracMult), nil
}

func RunCommandRaw(prog string, args ...string) (string, string, error) {
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := exec.Command(prog, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func RunCommand(prog string, args ...string) (string, error) {
	out, warn, err := RunCommandRaw(prog, args...)
	var errMsg strings.Builder
	isError := false
	if warn != "" {
		errMsg.WriteString(warn)
		if warn[len(warn)-1] != '\n' {
			errMsg.WriteString("\n")
		}
		isError = true
	}
	if err != nil {
		errMsg.WriteString(err.Error())
		isError = true
	}
	if isError {
		return "", errors.New(errMsg.String())
	}
	return out, nil
}

func FlushString(str string) {
	os.Stdout.WriteString(str)
	os.Stdout.Sync()
}

type ReadAllWriteAll interface {
	ReadAll() ([]byte, error)
	WriteAll([]byte) error
}

type FileRawa struct {
	FileName string
}

type MemoryRawa struct {
	Current []byte
}

func (rw *FileRawa) ReadAll() ([]byte, error) {
	return os.ReadFile(rw.FileName)
}
func (rw *FileRawa) WriteAll(content []byte) error {
	return os.WriteFile(rw.FileName, content, 0666)
}

func (rw *MemoryRawa) ReadAll() ([]byte, error) {
	var buf []byte
	size := len(rw.Current)
	if size > 0 {
		buf = make([]byte, size)
		copy(buf, rw.Current)
	}
	return buf, nil
}
func (rw *MemoryRawa) WriteAll(content []byte) error {
	rw.Current = content
	return nil
}

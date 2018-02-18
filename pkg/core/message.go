package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	Error MessageType = iota + 1
	Log
	Call
	Hijack
	ReadLineRequest
	ReadLineResponse
)

type MessageType byte
type Message struct {
	Type MessageType
	Body string
}

func write(dst io.Writer, args ...interface{}) error {
	var err error
	for _, arg := range args {
		switch val := arg.(type) {
		// Write strings as length prefixed byte arrays
		case string:
			err = write(dst, uint64(len(val)))
			if err != nil {
				return err
			}
			_, err = dst.Write([]byte(val))
			if err != nil {
				return err
			}
			// Any other types are handle with binary
		default:
			err := binary.Write(dst, binary.BigEndian, arg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func read(src io.Reader, args ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf(`Recovered from panic: %s

Stack trace:
%s

WARNING: This panic was likely caused by an RPC method with improperly type or args mapping in cmd.go
`, r, string(debug.Stack()))
			if err != nil {
				for _, line := range strings.Split(msg, "\n") {
					log.Error(line)
				}
			} else {
				err = errors.New(msg)
			}
		}
	}()

	for _, arg := range args {
		switch val := arg.(type) {
		// Read strings as length prefixed byte arrays
		case *string:
			var n uint64
			if err = read(src, &n); err != nil {
				return
			}
			bs := make([]byte, int(n))
			if _, err = io.ReadAtLeast(src, bs, int(n)); err != nil {
				return
			}
			*val = string(bs)
			// Any other types are handle with binary
		default:
			if err = binary.Read(src, binary.BigEndian, val); err != nil {
				return
			}
		}
	}
	return nil
}

func Errorf(dst io.Writer, msg string, args ...interface{}) error {
	return write(dst, Error, fmt.Sprintf(msg, args...))
}

func Logf(dst io.Writer, msg string, args ...interface{}) error {
	return write(dst, Log, fmt.Sprintf(msg, args...))
}

func Send(dst io.Writer, msg Message) error {
	return write(dst, msg.Type, msg.Body)
}

func Receive(src io.Reader) (Message, error) {
	var msg Message
	err := read(src, &msg.Type, &msg.Body)
	return msg, err
}

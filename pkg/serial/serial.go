package serial

import (
	"errors"
	"time"
)

const DefaultSize = 8 // Default value for Config.Size

type StopBits byte
type Parity byte

const (
	Stop1     StopBits = 1
	Stop1Half StopBits = 15
	Stop2     StopBits = 2
)

const (
	ParityNone  Parity = 'N'
	ParityOdd   Parity = 'O'
	ParityEven  Parity = 'E'
	ParityMark  Parity = 'M' // parity bit is always 1
	ParitySpace Parity = 'S' // parity bit is always 0
)

// Config contains the information needed to open a serial port.
//
// Currently few options are implemented, but more may be added in the
// future (patches welcome), so it is recommended that you create a
// new config addressing the fields by name rather than by order.
//
// For example:
//
//    c0 := &serial.Config{Name: "COM45", Baud: 115200, ReadTimeout: time.Millisecond * 500}
// or
//    c1 := new(serial.Config)
//    c1.Name = "/dev/tty.usbserial"
//    c1.Baud = 115200
//    c1.ReadTimeout = time.Millisecond * 500
//
type Config struct {
	Name        string
	Baud        int
	ReadTimeout time.Duration // Total timeout

	// Size is the number of data bits. If 0, DefaultSize is used.
	Size byte

	// Parity is the bit to use and defaults to ParityNone (no parity bit).
	Parity Parity

	// Number of stop bits to use. Default is 1 (1 stop bit).
	StopBits StopBits

	// RTSFlowControl bool
	// DTRFlowControl bool
	// XONFlowControl bool

	// CRLFTranslate bool
}

// ErrBadSize is returned if Size is not supported.
var ErrBadSize error = errors.New("unsupported serial data size")

// ErrBadStopBits is returned if the specified StopBits setting not supported.
var ErrBadStopBits error = errors.New("unsupported stop bit setting")

// ErrBadParity is returned if the parity is not supported.
var ErrBadParity error = errors.New("unsupported parity setting")

// OpenPort opens a serial port with the specified configuration
func OpenPort(c *Config) (*Port, error) {
	size, par, stop := c.Size, c.Parity, c.StopBits
	if size == 0 {
		size = DefaultSize
	}
	if par == 0 {
		par = ParityNone
	}
	if stop == 0 {
		stop = Stop1
	}
	return openPort(c.Name, c.Baud, size, par, stop, c.ReadTimeout)
}

// Converts the timeout values for Linux / POSIX systems
func posixTimeoutValues(readTimeout time.Duration) (vmin uint8, vtime uint8) {
	const MAXUINT8 = 1<<8 - 1 // 255
	// set blocking / non-blocking read
	var minBytesToRead uint8 = 1
	var readTimeoutInDeci int64
	if readTimeout > 0 {
		// EOF on zero read
		minBytesToRead = 0
		// convert timeout to deciseconds as expected by VTIME
		readTimeoutInDeci = (readTimeout.Nanoseconds() / 1e6 / 100)
		// capping the timeout
		if readTimeoutInDeci < 1 {
			// min possible timeout 1 Deciseconds (0.1s)
			readTimeoutInDeci = 1
		} else if readTimeoutInDeci > MAXUINT8 {
			// max possible timeout is 255 deciseconds (25.5s)
			readTimeoutInDeci = MAXUINT8
		}
	}
	return minBytesToRead, uint8(readTimeoutInDeci)
}

func contains(a []byte, d byte) bool {
	for _, v := range a {
		if d == v {
			return true
		}
	}
	return false
}

func send(s *Port, d byte) error {
	_, err := s.Write([]byte("S"))
	if err != nil {
		return err
	}

	err = wait(s, 'R')
	if err != nil {
		return err
	}

	sendData := []byte{d}
	_, err = s.Write(sendData)
	if err != nil {
		return err
	}

	err = wait(s, 'O')
	if err != nil {
		return err
	}
	return nil
}

func wait(s *Port, b byte) error {
	for {
		buf := make([]byte, 128)
		n, err := s.Read(buf)
		if err != nil {
			return err
		}
		if contains(buf[:n], b) {
			break
		}
	}
	return nil
}

func Send(s *Port, d []byte) error {
	// send S(send) status
	_, err := s.Write([]byte("S"))
	if err != nil {
		return err
	}
	// send length of data
	_, err = s.Write([]byte{uint8(len(d))})
	if err != nil {
		return err
	}
	err = wait(s, 'O')
	if err != nil {
		return err
	}

	// send main data
	_, err = s.Write(d)

	err = wait(s, 'O')
	if err != nil {
		return err
	}
	return nil
}

func Receive(s *Port) ([]byte, error) {
	n, err := s.Write([]byte("R"))
	if err != nil {
		return nil, err
	}

	data := make([]byte, 0)
	for {
		buf := make([]byte, 128)
		n, err = s.Read(buf)
		if err != nil {
			return nil, err
		}

		data = append(data, buf[:n]...)
		if contains(buf, '\n') {
			break
		}
	}

	_, err = s.Write([]byte("O"))
	if err != nil {
		return nil, err
	}
	return data, nil
}

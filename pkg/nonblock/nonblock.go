package nonblock

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const Scheme = "nonblock"

var DefaultDeadline = time.Millisecond

func init() {
	if err := zap.RegisterSink(Scheme, NewSink); err != nil {
		panic("Failed to register zap sink for scheme " + Scheme + ": " + err.Error())
	}
}

func NewSink(u *url.URL) (zap.Sink, error) {
	if u == nil || u.Scheme != Scheme {
		return nil, errors.New("invalid url or scheme")
	}
	t, err := os.OpenFile("nonblock.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(t, "Opening output: %#v\n", u)
	var f *os.File
	switch u.Opaque {
	case "stdout":
		f, err = dup(os.Stdout)
	case "stderr":
		f, err = dup(os.Stderr)
	default:
		f, err = os.OpenFile(u.Opaque, os.O_WRONLY|os.O_APPEND|os.O_CREATE|syscall.O_NONBLOCK, 0666)
	}
	if err != nil {
		t.Close()
		return nil, err
	}

	d := DefaultDeadline
	if deadline := u.Query().Get("deadline"); deadline != "" {
		d, err = time.ParseDuration(deadline)
		if err != nil {
			return nil, err
		}
	}

	fifo, err := isFIFO(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	fmt.Fprintf(t, "%s is FIFO: %t\n", f.Name(), fifo)

	return &nonblock{f: f, t: t, d: d}, nil
}

type nonblock struct {
	sync.Mutex

	f *os.File
	t io.WriteCloser
	d time.Duration
}

func (n *nonblock) Write(p []byte) (int, error) {
	n.Lock()
	defer n.Unlock()

	now := time.Now().Round(time.Millisecond)
	if err := n.f.SetWriteDeadline(time.Now().Add(n.d)); err != nil {
		fmt.Fprintf(n.t, "%s %s Failed to set write deadline: %v\n", n.f.Name(), now, err)
	}
	i, err := n.f.Write(p)
	fmt.Fprintf(n.t, "%s %s Wrote %d of %d: %v\n", n.f.Name(), now, i, len(p), err)
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return len(p), nil
	}
	return i, err
}

func (n *nonblock) Sync() error {
	n.Lock()
	defer n.Unlock()

	return n.f.Sync()
}

func (n *nonblock) Close() error {
	n.Lock()
	defer n.Unlock()

	return errors.Join(n.f.Close(), n.t.Close())
}

func dup(f *os.File) (*os.File, error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return nil, err
	}
	var duperr error
	var newfd int
	if err := rc.Control(func(oldfd uintptr) { newfd, err = syscall.Dup(int(oldfd)) }); err != nil {
		return nil, err
	}
	if duperr != nil {
		return nil, err
	}
	if err := syscall.SetNonblock(newfd, true); err != nil {
		return nil, err
	}
	f = os.NewFile(uintptr(newfd), f.Name())
	if f == nil {
		return nil, errors.New("failed to reopen file")
	}
	return f, nil
}

func isFIFO(f *os.File) (bool, error) {
	fd, err := f.SyscallConn()
	if err != nil {
		return false, err
	}
	info := &syscall.Stat_t{}
	err = fd.Control(func(fd uintptr) {
		syscall.Fstat(int(fd), info)
	})
	if err != nil {
		return false, err
	}
	return info.Mode&syscall.S_IFIFO != 0, nil
}

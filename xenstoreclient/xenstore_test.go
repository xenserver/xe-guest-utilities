package xenstoreclient

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"
)

type mockFile struct {
	r io.Reader
	w io.Writer
	t *testing.T
}

func NewMockFile(t *testing.T) io.ReadWriteCloser {
	var b bytes.Buffer

	return &mockFile{
		r: &b,
		w: &b,
		t: t,
	}
}

func (f *mockFile) Read(p []byte) (n int, err error) {
	for i := 0; i < 1; i++ {
		n, err = f.r.Read(p)
		if err == io.EOF {
			fmt.Printf("Read sleep %#v second\n", i)
			time.Sleep(1 * time.Second)
		} else {
			fmt.Printf("Read=%#v err %#v\n", n, err)
			return
		}
	}
	return 0, io.EOF
}

func (f *mockFile) Write(b []byte) (n int, err error) {
	n, err = f.w.Write(b)
	fmt.Printf("Write=%#v err %#v\n", n, err)
	return
}

func (f *mockFile) Close() error {
	f.t.Logf("Close()")
	return nil
}

func TestXenStore(t *testing.T) {
	xs, err := newXenstore(0, NewMockFile(t))
	if err != nil {
		t.Errorf("newXenstore error: %#v\n", err)
	}
	defer xs.Close()

	if err := xs.Write("foo", "bar"); err != nil {
		t.Errorf("xs.Write error: %#v\n", err)
	}

	if _, err := xs.Read("foo"); err != nil {
		t.Errorf("xs.Read error: %#v\n", err)
	}
}

func TestXenStoreWatch(t *testing.T) {
	xs, err := newXenstore(0, NewMockFile(t))
	if err != nil {
		t.Errorf("newXenstore error: %#v\n", err)
	}
	defer xs.Close()

	ready := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		<-ready
		if err := xs.StopWatch(); err != nil {
			t.Errorf("xs.StopWatch error: %#v\n", err)
		}
		close(stopped)
	}()

	if out, err := xs.Watch([]string{"foo"}); err == nil {
		close(ready)
		if e, ok := <-out; ok {
			fmt.Println(e.Path)
		}
	} else {
		t.Errorf("xs.Watch(\"foo\") error: %#v\n", err)
	}
	<-stopped
}

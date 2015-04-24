package xenstoreclient

import (
	"bytes"
	"testing"
)

type mockFile struct {
	t *testing.T
}

func (f *mockFile) Read(b []byte) (n int, err error) {
	value := "i am value"
	p := &Packet{
		OpCode: XS_READ,
		Req:    0,
		TxID:   0,
		Length: uint32(len(value)),
		Value:  []byte(value),
	}
	var buf bytes.Buffer
	if err = p.Write(&buf); err != nil {
		return 0, err
	}
	copy(b, buf.Bytes())
	n = buf.Len()
	f.t.Logf("Read %d bytes", n)
	return n, nil
}

func (f *mockFile) Write(b []byte) (n int, err error) {
	buf := bytes.NewBuffer(b)
	if _, err := ReadPacket(buf); err != nil {
		return 0, err
	}
	n = len(b)
	f.t.Logf("Write %d bytes", n)
	return n, nil
}

func (f *mockFile) Close() error {
	f.t.Logf("Close()")
	return nil
}

func TestXenStore(t *testing.T) {
	xs, err := newXenstore(0, &mockFile{t})
	if err != nil {
		t.Errorf("newXenstore error: %#v\n", err)
	}
	defer xs.Close()

	if _, err := xs.Read("foo"); err != nil {
		t.Errorf("xs.Read error: %#v\n", err)
	}

	if err := xs.Write("foo", "bar"); err != nil {
		t.Errorf("xs.Read error: %#v\n", err)
	}
}

package xenstoreclient

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Permission int

const (
	PERM_NONE Permission = iota
	PERM_READ
	PERM_WRITE
	PERM_READWRITE
)

type Operation uint32

const (
	XS_READ              Operation = 2
	XS_GET_PERMS         Operation = 3
	XS_WATCH             Operation = 4
	XS_UNWATCH           Operation = 5
	XS_TRANSACTION_START Operation = 6
	XS_TRANSACTION_END   Operation = 7
	XS_WRITE             Operation = 11
	XS_MKDIR             Operation = 12
	XS_RM                Operation = 13
	XS_SET_PERMS         Operation = 14
	XS_WATCH_EVENT       Operation = 15
	XS_ERROR             Operation = 16
)

type Packet struct {
	OpCode Operation
	Req    uint32
	TxID   uint32
	Length uint32
	Value  []byte
}

type Event struct {
	Token string
	Data  []byte
}

type XenStoreClient interface {
	Close() error
	DO(packet *Packet) (*Packet, error)
	Read(path string) (string, error)
	Mkdir(path string) error
	Rm(path string) error
	Write(path string, value string) error
	GetPermission(path string) (map[int]Permission, error)
	Watch(path string) (<-chan Event, error)
	StopWatch() error
}

func ReadPacket(r io.Reader) (packet *Packet, err error) {

	packet = &Packet{}

	err = binary.Read(r, binary.LittleEndian, &packet.OpCode)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.LittleEndian, &packet.Req)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.LittleEndian, &packet.TxID)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.LittleEndian, &packet.Length)
	if err != nil {
		return nil, err
	}

	if packet.Length > 0 {
		packet.Value = make([]byte, packet.Length)
		_, err = io.ReadFull(r, packet.Value)
		if err != nil {
			return nil, err
		}
		if packet.OpCode == XS_ERROR {
			return nil, errors.New(strings.Split(string(packet.Value), "\x00")[0])
		}
	}

	return packet, nil
}

func (p *Packet) Write(w io.Writer) (err error) {
	var bw *bufio.Writer

	if w1, ok := w.(*bufio.Writer); ok {
		bw = w1
	} else {
		bw = bufio.NewWriter(w)
	}
	defer bw.Flush()

	err = binary.Write(bw, binary.LittleEndian, p.OpCode)
	if err != nil {
		return err
	}
	err = binary.Write(bw, binary.LittleEndian, p.Req)
	if err != nil {
		return err
	}
	err = binary.Write(bw, binary.LittleEndian, p.TxID)
	if err != nil {
		return err
	}
	err = binary.Write(bw, binary.LittleEndian, p.Length)
	if err != nil {
		return err
	}
	if p.Length > 0 {
		_, err = bw.Write(p.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

type XenStore struct {
	tx               uint32
	xbFile           io.ReadWriteCloser
	xbFileReader     *bufio.Reader
	muWatch          *sync.Mutex
	onceWatch        *sync.Once
	watchQueues      map[string]chan Event
	watchStopChan    chan struct{}
	watchStoppedChan chan struct{}
	nonWatchQueue    chan []byte
}

func NewXenstore(tx uint32) (XenStoreClient, error) {
	devPath, err := getDevPath()
	if err != nil {
		return nil, err
	}

	xbFile, err := os.OpenFile(devPath, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	return newXenstore(tx, xbFile)
}

func newXenstore(tx uint32, rwc io.ReadWriteCloser) (XenStoreClient, error) {
	return &XenStore{
		tx:               tx,
		xbFile:           rwc,
		xbFileReader:     bufio.NewReader(rwc),
		watchQueues:      make(map[string]chan Event, 0),
		nonWatchQueue:    nil,
		watchStopChan:    make(chan struct{}, 1),
		watchStoppedChan: make(chan struct{}, 1),
		onceWatch:        &sync.Once{},
		muWatch:          &sync.Mutex{},
	}, nil
}

func (xs *XenStore) Close() error {
	return xs.xbFile.Close()
}

func (xs *XenStore) DO(req *Packet) (resp *Packet, err error) {
	err = req.Write(xs.xbFile)
	if err != nil {
		return nil, err
	}

	var r io.Reader
	if xs.nonWatchQueue != nil {
		data := <-xs.nonWatchQueue
		r = bytes.NewReader(data)
	} else {
		r = xs.xbFileReader
	}

	resp, err = ReadPacket(r)
	return resp, err
}

func (xs *XenStore) Read(path string) (string, error) {
	v := []byte(path + "\x00")
	req := &Packet{
		OpCode: XS_READ,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	resp, err := xs.DO(req)
	if err != nil {
		return "", err
	}
	return string(resp.Value), nil
}

func (xs *XenStore) Mkdir(path string) error {
	v := []byte(path + "\x00")
	req := &Packet{
		OpCode: XS_WRITE,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	_, err := xs.DO(req)
	return err
}

func (xs *XenStore) Rm(path string) error {
	v := []byte(path + "\x00")
	req := &Packet{
		OpCode: XS_RM,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	_, err := xs.DO(req)
	return err
}

func (xs *XenStore) Write(path string, value string) error {
	v := []byte(path + "\x00" + value)
	req := &Packet{
		OpCode: XS_WRITE,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	_, err := xs.DO(req)
	return err
}

func (xs *XenStore) GetPermission(path string) (map[int]Permission, error) {
	perm := make(map[int]Permission, 0)

	v := []byte(path + "\x00")
	req := &Packet{
		OpCode: XS_GET_PERMS,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	resp, err := xs.DO(req)
	if err != nil {
		return nil, err
	}

	for _, e := range strings.Split(string(resp.Value[:len(resp.Value)-1]), "\x00") {
		k, err := strconv.Atoi(e[1:])
		if err != nil {
			return nil, err
		}
		var p Permission
		switch e[0] {
		case 'n':
			p = PERM_NONE
		case 'r':
			p = PERM_READ
		case 'w':
			p = PERM_WRITE
		case 'b':
			p = PERM_READWRITE
		}
		perm[k] = p
	}
	return perm, nil
}

func (xs *XenStore) Watch(path string) (<-chan Event, error) {
	watcher := func() {

		type XSData struct {
			*Packet
			Error error
		}

		xsDataChan := make(chan XSData, 100)
		go func(r io.Reader, out chan<- XSData) {
			for {
				p, err := ReadPacket(r)
				out <- XSData{Packet: p, Error: err}
			}
		}(xs.xbFileReader, xsDataChan)

		xs.nonWatchQueue = make(chan []byte, 100)
		for {
			select {
			case <-xs.watchStopChan:
				fmt.Printf("watch receive stop signal, quit.")
				xs.watchStopChan <- struct{}{}
				return
			case xsdata := <-xsDataChan:
				if xsdata.Error != nil {
					fmt.Printf("watch receive error: %#v", xsdata.Error)
					return
				}
				switch xsdata.Packet.OpCode {
				case XS_WATCH_EVENT:
					parts := strings.SplitN(string(xsdata.Value), "\x00", 2)
					path := parts[0]
					token := parts[1]
					data := []byte(parts[2])
					if c, ok := xs.watchQueues[path]; ok {
						c <- Event{token, data}
					}
				default:
					var b bytes.Buffer
					xsdata.Packet.Write(&b)
					xs.nonWatchQueue <- b.Bytes()
				}
			}
		}
	}
	xs.onceWatch.Do(watcher)
	xs.muWatch.Lock()
	defer xs.muWatch.Unlock()
	if _, ok := xs.watchQueues[path]; !ok {
		xs.watchQueues[path] = make(chan Event, 100)
	}
	return xs.watchQueues[path], nil
}

func (xs *XenStore) StopWatch() error {
	xs.watchStopChan <- struct{}{}
	<-xs.watchStoppedChan
	xs.nonWatchQueue = nil
	return nil
}

type CachedXenStore struct {
	xs         XenStoreClient
	writeCache map[string]string
	lastCommit map[string]time.Time
}

func NewCachedXenstore(tx uint32) (XenStoreClient, error) {
	xs, err := NewXenstore(tx)
	if err != nil {
		return nil, err
	}
	return &CachedXenStore{
		xs:         xs,
		writeCache: make(map[string]string, 0),
		lastCommit: make(map[string]time.Time, 0),
	}, nil
}

func (xs *CachedXenStore) Write(path string, value string) error {
	if v, ok := xs.writeCache[path]; ok && v == value {
		if t, ok := xs.lastCommit[path]; ok && t.After(time.Now().Add(-2*time.Minute)) {
			return nil
		}
	}
	err := xs.xs.Write(path, value)
	if err != nil {
		xs.writeCache[path] = value
		xs.lastCommit[path] = time.Now()
	}
	return err
}

func (xs *CachedXenStore) Close() error {
	return xs.xs.Close()
}

func (xs *CachedXenStore) DO(req *Packet) (resp *Packet, err error) {
	return xs.xs.DO(req)
}

func (xs *CachedXenStore) Read(path string) (string, error) {
	return xs.xs.Read(path)
}

func (xs *CachedXenStore) Mkdir(path string) error {
	return xs.xs.Mkdir(path)
}

func (xs *CachedXenStore) Rm(path string) error {
	return xs.xs.Rm(path)
}

func (xs *CachedXenStore) GetPermission(path string) (map[int]Permission, error) {
	return xs.xs.GetPermission(path)
}

func (xs *CachedXenStore) Watch(path string) (<-chan Event, error) {
	return xs.xs.Watch(path)
}

func (xs *CachedXenStore) StopWatch() error {
	return xs.xs.StopWatch()
}

func (xs *CachedXenStore) Clear() {
	xs.writeCache = make(map[string]string, 0)
	xs.lastCommit = make(map[string]time.Time, 0)
}

func getDevPath() (devPath string, err error) {
	devPaths := []string{
		"/proc/xen/xenbus",
		"/dev/xen/xenbus",
		"/kern/xen/xenbus",
	}
	for _, devPath = range devPaths {
		if _, err = os.Stat(devPath); err == nil {
			return devPath, err
		}
	}
	return "", fmt.Errorf("Cannot locate xenbus dev path in %v", devPaths)
}

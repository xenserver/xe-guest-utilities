package xenstoreclient

import (
	syslog "../syslog"
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
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
	XS_DEBUG                Operation = 0
	XS_DIRECTORY            Operation = 1
	XS_READ                 Operation = 2
	XS_GET_PERMS            Operation = 3
	XS_WATCH                Operation = 4
	XS_UNWATCH              Operation = 5
	XS_TRANSACTION_START    Operation = 6
	XS_TRANSACTION_END      Operation = 7
	XS_INTRODUCE            Operation = 8
	XS_RELEASE              Operation = 9
	XS_GET_DOMAIN_PATH      Operation = 10
	XS_WRITE                Operation = 11
	XS_MKDIR                Operation = 12
	XS_RM                   Operation = 13
	XS_SET_PERMS            Operation = 14
	XS_WATCH_EVENT          Operation = 15
	XS_ERROR                Operation = 16
	XS_IS_DOMAIN_INTRODUCED Operation = 17
	XS_RESUME               Operation = 18
	XS_SET_TARGET           Operation = 19
	XS_RESTRICT             Operation = 128
)

type Packet struct {
	OpCode Operation
	Req    uint32
	TxID   uint32
	Length uint32
	Value  []byte
}

type XenStoreClient interface {
	Close() error
	DO(packet *Packet) (*Packet, error)
	Read(path string) (string, error)
	List(path string) ([]string, error)
	Mkdir(path string) error
	Rm(path string) error
	Write(path string, value string) error
	GetPermission(path string) (map[int]Permission, error)
	Watch(path string, token string) error
	WatchEvent(key string) (token string, ok bool)
	UnWatch(path string, token string) error
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

type WatchQueueManager struct {
	watchQueues map[string]chan string
	rwlocker    *sync.RWMutex
}

func (wq *WatchQueueManager) RemoveByKey(key string) {
	wq.rwlocker.Lock()
	defer wq.rwlocker.Unlock()
	delete(wq.watchQueues, key)
	return
}

func (wq *WatchQueueManager) SetEventByKey(key string, token string) (ok bool) {
	wq.rwlocker.RLock()
	defer wq.rwlocker.RUnlock()
	wq.watchQueues[key] <- token
	return
}

func (wq *WatchQueueManager) GetEventByKey(key string) (token string, ok bool) {
	wq.rwlocker.RLock()
	defer wq.rwlocker.RUnlock()
	ec, ok := wq.watchQueues[key]
	if len(ec) != 0 {
		return <-ec, ok
	} else {
		ok = false
	}
	return
}

func (wq *WatchQueueManager) AddChanByKey(key string) {
	wq.rwlocker.Lock()
	defer wq.rwlocker.Unlock()
	wq.watchQueues[key] = make(chan string, 100)
}

type XenStore struct {
	tx                 uint32
	xbFile             io.ReadWriteCloser
	xbFileReader       *bufio.Reader
	muWatch            *sync.Mutex
	onceWatch          *sync.Once
	watchQueue         WatchQueueManager
	watchStopChan      chan struct{}
	watchStoppedChan   chan struct{}
	nonWatchQueue      chan []byte
	xbFileReaderLocker *sync.Mutex
	logger             *log.Logger
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

const (
	LoggerName string = "xenstore"
)

func newXenstore(tx uint32, rwc io.ReadWriteCloser) (XenStoreClient, error) {
	var loggerWriter io.Writer = os.Stderr
	var topic string = LoggerName
	if w, err := syslog.NewSyslogWriter(topic); err == nil {
		loggerWriter = w
		topic = ""
	} else {
		fmt.Fprintf(os.Stderr, "NewSyslogWriter(%s) error: %s, use stderr logging\n", topic, err)
		topic = LoggerName + ": "
	}

	logger := log.New(loggerWriter, topic, 0)

	return &XenStore{
		tx:           tx,
		xbFile:       rwc,
		xbFileReader: bufio.NewReader(rwc),
		watchQueue: WatchQueueManager{
			watchQueues: make(map[string]chan string, 0),
			rwlocker:    &sync.RWMutex{},
		},
		nonWatchQueue:      nil,
		watchStopChan:      make(chan struct{}, 1),
		watchStoppedChan:   make(chan struct{}, 1),
		onceWatch:          &sync.Once{},
		muWatch:            &sync.Mutex{},
		xbFileReaderLocker: &sync.Mutex{},
		logger:             logger,
	}, nil
}

func (xs *XenStore) Close() error {
	return xs.xbFile.Close()
}

func (xs *XenStore) DO(req *Packet) (resp *Packet, err error) {
	xs.xbFileReaderLocker.Lock()
	defer xs.xbFileReaderLocker.Unlock()
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

func (xs *XenStore) List(path string) ([]string, error) {
	v := []byte(path + "\x00")
	req := &Packet{
		OpCode: XS_DIRECTORY,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	resp, err := xs.DO(req)
	if err != nil {
		return []string{}, err
	}
	subItems := strings.Split(
		string(bytes.Trim(resp.Value, "\x00")), "\x00")

	return subItems, nil
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

func (xs *XenStore) UnWatch(path string, token string) (err error) {
	v := []byte(path + "\x00" + token + "\x00")
	req := &Packet{
		OpCode: XS_UNWATCH,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	_, err = xs.DO(req)
	if err != nil {
		return
	}
	xs.watchQueue.RemoveByKey(path)
	return nil
}

func (xs *XenStore) Watch(path string, token string) error {
	watcher := func() {
		xs.logger.Printf("Watch: Start")
		type XSData struct {
			*Packet
			Error error
		}

		xsDataChan := make(chan XSData, 100)
		xsReadStop := make(chan bool)
		go func(r io.Reader, out chan<- XSData, c <-chan bool) {
			for {
				// The read will return at once if no data in r
				p, err := ReadPacket(r)
				ticker := time.Tick(1 * time.Second)
				if err != nil {
					select {
					case <-c:
						return
					case <-ticker:
						continue
					}
				}
				out <- XSData{Packet: p, Error: err}
			}
		}(xs.xbFileReader, xsDataChan, xsReadStop)

		xs.nonWatchQueue = make(chan []byte, 100)
		for {
			select {
			case <-xs.watchStopChan:
				xs.logger.Printf("Watch: receive stop signal, quit.\n")
				xs.watchStoppedChan <- struct{}{}
				xsReadStop <- true
				return
			case xsdata := <-xsDataChan:
				if xsdata.Error != nil {
					xs.logger.Printf("Watch: receive error: %#v", xsdata.Error)
					return
				}
				switch xsdata.Packet.OpCode {
				case XS_WATCH_EVENT:
					parts := strings.SplitN(string(xsdata.Value), "\x00", 2)
					path := parts[0]
					token := parts[1]
					xs.logger.Printf("Get XS_WATCH_EVENT key:%s, token:%s\n", path, token)
					xs.watchQueue.SetEventByKey(path, token)
				default:
					xs.logger.Printf("Get non watch event %#v\n", xsdata.Packet.OpCode)
					var b bytes.Buffer
					xsdata.Packet.Write(&b)
					xs.nonWatchQueue <- b.Bytes()
				}
			}
		}
	}
	xs.muWatch.Lock()
	defer xs.muWatch.Unlock()
	v := []byte(path + "\x00" + token + "\x00")
	req := &Packet{
		OpCode: XS_WATCH,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	_, err := xs.DO(req)
	if err != nil {
		xs.logger.Printf("Watch failed with error %#v\n", err)
		return err
	}
	xs.watchQueue.AddChanByKey(path)
	go xs.onceWatch.Do(watcher)
	return nil
}

func (xs *XenStore) WatchEvent(key string) (token string, ok bool) {
	return xs.watchQueue.GetEventByKey(key)
}

func (xs *XenStore) StopWatch() error {
	xs.watchStopChan <- struct{}{}
	xs.nonWatchQueue = nil
	<-xs.watchStoppedChan
	return nil
}

type Content struct {
	value     string
	keepalive bool
}

type CachedXenStore struct {
	xs         XenStoreClient
	writeCache map[string]Content
}

func NewCachedXenstore(tx uint32) (XenStoreClient, error) {
	xs, err := NewXenstore(tx)
	if err != nil {
		return nil, err
	}
	return &CachedXenStore{
		xs:         xs,
		writeCache: make(map[string]Content, 0),
	}, nil
}

func (xs *CachedXenStore) Write(path string, value string) error {
	if v, ok := xs.writeCache[path]; ok && v.value == value {
		v.keepalive = true
		xs.writeCache[path] = v
		return nil
	}
	err := xs.xs.Write(path, value)
	if err == nil {
		xs.writeCache[path] = Content{value: value, keepalive: true}
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

func (xs *CachedXenStore) List(path string) ([]string, error) {
	return xs.xs.List(path)
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

func (xs *CachedXenStore) Watch(path string, token string) error {
	return xs.xs.Watch(path, token)
}

func (xs *CachedXenStore) WatchEvent(key string) (token string, ok bool) {
	return xs.xs.WatchEvent(key)
}

func (xs *CachedXenStore) UnWatch(path string, token string) error {
	return xs.xs.UnWatch(path, token)
}

func (xs *CachedXenStore) StopWatch() error {
	return xs.xs.StopWatch()
}

func (xs *CachedXenStore) Clear() {
	xs.writeCache = make(map[string]Content, 0)
}

func (xs *CachedXenStore) InvalidCacheFlush() error {
	for key, value := range xs.writeCache {
		if value.keepalive {
			value.keepalive = false
			xs.writeCache[key] = value
		} else {
			err := xs.Rm(key)
			if err != nil {
				return err
			} else {
				delete(xs.writeCache, key)
			}
		}
	}
	return nil
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

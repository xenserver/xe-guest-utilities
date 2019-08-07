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
)

type Perm int

const (
	PERM_NONE Perm = iota
	PERM_READ
	PERM_WRITE
	PERM_READWRITE
)

func (p *Perm) ToStr() string {
	switch *p {
	case PERM_NONE:
		return "n"
	case PERM_READ:
		return "r"
	case PERM_WRITE:
		return "w"
	case PERM_READWRITE:
		return "b"
	}
	return " "
}

type Permission struct {
	Id uint
	Pe Perm
}

func (p *Permission) ToStr() string {
	return p.Pe.ToStr() + strconv.FormatUint(uint64(p.Id), 10)
}

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

const (
	EVENT_PATH = iota
	EVENT_TOKEN
	EVENT_MAXNUM
)

type Event struct {
	Path  string
	Token string
}

type XenStoreClient interface {
	Close() error
	DO(packet *Packet) (*Packet, error)
	Read(path string) (string, error)
	List(path string) ([]string, error)
	Mkdir(path string) error
	Rm(path string) error
	Write(path string, value string) error
	GetPermission(path string) ([]Permission, error)
	SetPermission(path string, perms []Permission) error
	Watch(path []string) (chan Event, error)
	StopWatch() error
	GetDomainPath(domid string) (string, error)
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
		err = bw.Flush()
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
	onceWatch        *sync.Once
	outEvent         chan Event
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
		nonWatchQueue:    nil,
		watchStopChan:    make(chan struct{}, 1),
		watchStoppedChan: make(chan struct{}, 1),
		onceWatch:        &sync.Once{},
		outEvent:         make(chan Event, 100),
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

func (xs *XenStore) GetPermission(path string) ([]Permission, error) {
	perms := make([]Permission, 0)

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
		k, err := strconv.ParseUint(e[1:], 0, 0)
		if err != nil {
			return nil, err
		}
		var p Perm
		switch e[0] {
		case 'n':
			p = PERM_NONE
		case 'r':
			p = PERM_READ
		case 'w':
			p = PERM_WRITE
		case 'b':
			p = PERM_READWRITE
		default:
			return nil, errors.New("Invalid permision value")
		}
		perms = append(perms, Permission{uint(k), p})
	}
	return perms, nil
}

func (xs *XenStore) SetPermission(path string, perms []Permission) error {
	s := path + "\x00"
	for _, p := range perms {
		s += p.ToStr() + "\x00"
	}

	v := []byte(s)
	req := &Packet{
		OpCode: XS_SET_PERMS,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	_, err := xs.DO(req)
	return err
}

func (xs *XenStore) add_watch(path string) error {
	v := []byte(path + "\x00" + path + "\x00")
	req := &Packet{
		OpCode: XS_WATCH,
		Req:    0,
		TxID:   xs.tx,
		Length: uint32(len(v)),
		Value:  v,
	}
	_, err := xs.DO(req)
	return err
}

func (xs *XenStore) read_watch() {
	type XSData struct {
		*Packet
		Error error
	}

	xsDataChan := make(chan XSData, 100)
	readStoppedChan := make(chan struct{}, 1)
	xs.nonWatchQueue = make(chan []byte, 100)

	go func(r io.Reader, out chan<- XSData) {
		for {
			p, err := ReadPacket(r)
			out <- XSData{Packet: p, Error: err}
			if err != nil {
				readStoppedChan <- struct{}{}
				return
			}
		}
	}(xs.xbFileReader, xsDataChan)

	go func(xsDtaChan chan XSData) {
		for {
			select {
			case <-xs.watchStopChan:
				close(xs.outEvent)
				xs.Close()
				<-readStoppedChan
				xs.watchStoppedChan <- struct{}{}
				return
			case xsdata := <-xsDataChan:
				if xsdata.Error != nil {
					close(xs.outEvent)
					xs.watchStoppedChan <- struct{}{}
					return
				}
				switch xsdata.Packet.OpCode {
				case XS_WATCH_EVENT:
					parts := strings.SplitN(string(xsdata.Value), "\x00", 2)
					if len(parts) == EVENT_MAXNUM {
						xs.outEvent <- Event{parts[EVENT_PATH], parts[EVENT_TOKEN]}
					}
				default:
					var b bytes.Buffer
					xsdata.Packet.Write(&b)
					xs.nonWatchQueue <- b.Bytes()
				}
			}
		}
	}(xsDataChan)
}

func (xs *XenStore) Watch(path []string) (chan Event, error) {
	xs.onceWatch.Do(xs.read_watch)
	for _, p := range path {
		if err := xs.add_watch(p); err != nil {
			fmt.Fprintf(os.Stderr, "failed to add watch: %s\n", p)
			xs.StopWatch()
			return nil, err
		}
	}
	return xs.outEvent, nil
}

func (xs *XenStore) StopWatch() error {
	if xs.nonWatchQueue != nil {
		xs.watchStopChan <- struct{}{}
		<-xs.watchStoppedChan
		xs.nonWatchQueue = nil
	}
	return nil
}

func (xs *XenStore) GetDomainPath(domid string) (string, error) {
	v := []byte(domid + "\x00")
	req := &Packet{
		OpCode: XS_GET_DOMAIN_PATH,
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

func (xs *CachedXenStore) GetPermission(path string) ([]Permission, error) {
	return xs.xs.GetPermission(path)
}

func (xs *CachedXenStore) SetPermission(path string, perms []Permission) error {
	return xs.xs.SetPermission(path, perms)
}

func (xs *CachedXenStore) Watch(path []string) (chan Event, error) {
	return xs.xs.Watch(path)
}

func (xs *CachedXenStore) StopWatch() error {
	return xs.xs.StopWatch()
}

func (xs *CachedXenStore) GetDomainPath(domid string) (string, error) {
	return xs.xs.GetDomainPath(domid)
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

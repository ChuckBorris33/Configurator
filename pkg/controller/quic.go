package controller

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fxamacker/cbor"
)

type QuicCommand uint8

const (
	QuicCmdInvalid QuicCommand = iota
	QuicCmdGet
	QuicCmdSet
	QuicCmdLog
	QuicCmdCalImu
	QuicCmdBlackbox
)

const (
	QuicFlagNone = iota
	QuicFlagError
)

const (
	QuicValInvalid = iota
	QuicValInfo
	QuicValProfile
	QuicValDefaultProfile
	QuicValBlackboxRate
)

type quicPacket struct {
	cmd     QuicCommand
	flag    uint8
	len     uint16
	payload []byte
}

const quicHeaderLen = uint16(4)

var quicChannel = make(map[QuicCommand]chan quicPacket)

var QuicLog = make(chan string, 100)
var QuicBlackbox = make(chan interface{}, 100)

type Blackbox struct {
	VbatFilter float32 `cbor:"vbat_filter"`

	GyroRaw    [3]float32 `cbor:"gyro_raw"`
	GyroFilter [3]float32 `cbor:"gyro_filter"`
	GyroVector [3]float32 `cbor:"gyro_vector"`

	RxRaw    [4]float32 `cbor:"rx_raw"`
	RxFilter [4]float32 `cbor:"rx_filter"`
	RxAux    []uint     `cbor:"rx_aux"`

	AccelRaw    [3]float32 `cbor:"accel_raw"`
	AccelFilter [3]float32 `cbor:"accel_filter"`
}

func (c *Controller) ReadQUIC() error {
	header, err := c.readAtLeast(int(quicHeaderLen - 1))
	if err != nil {
		return err
	}
	length := int(quicHeaderLen)

	cmd := QuicCommand(header[0] & (0xff >> 3))
	flag := (header[0] >> 5)
	payloadLen := uint16(header[1])<<8 | uint16(header[2])
	size := int(quicHeaderLen + payloadLen)

	payload, err := c.readAtLeast(int(payloadLen))
	if err != nil {
		return err
	}

	if len(payload) != int(payloadLen) {
		log.Debugf("% x (%s)\n", payload, string(payload))
		log.Fatalf("<quic> invalid size (%d vs %d)", length, size)
	}

	packet := quicPacket{
		cmd:     cmd,
		flag:    flag,
		len:     payloadLen,
		payload: payload,
	}
	log.Debugf("<quic> received cmd: %d flag: %d len: %d\n", packet.cmd, packet.flag, packet.len)

	switch packet.cmd {
	case QuicCmdLog:
		val := new(string)
		if err := cbor.Unmarshal(packet.payload, val); err != nil {
			log.Fatal(err)
		}
		log.Debugf("<quic> log %s\n", *val)
		QuicLog <- *val
		break
	case QuicCmdBlackbox:
		val := new(Blackbox)
		if err := cbor.Unmarshal(packet.payload, val); err != nil {
			log.Fatal(err)
		}
		log.Debugf("<quic> blackbox %+v\n", *val)
		QuicBlackbox <- *val
		break
	default:
		select {
		case quicChannel[cmd] <- packet:
		default:
		}
	}
	return nil
}

func (c *Controller) SendQUIC(cmd QuicCommand, data []byte) (*quicPacket, error) {
	buf := []byte{
		'#',
		byte(cmd),
		byte((len(data) >> 8) & 0xFF),
		byte(len(data) & 0xFF),
	}
	buf = append(buf, data...)

	if quicChannel[cmd] == nil {
		quicChannel[cmd] = make(chan quicPacket)
	}
	c.writeChannel <- buf

	select {
	case p := <-quicChannel[cmd]:
		if p.flag == QuicFlagError {
			var msg string
			if err := cbor.Unmarshal(p.payload, &msg); err != nil {
				return nil, err
			}
			return nil, errors.New(msg)
		}
		return &p, nil
	case <-time.After(1 * time.Second):
		return nil, errors.New("quic recive timeout")
	}
}

func (c *Controller) GetQUIC(typ QuicCommand, v interface{}) error {
	req, err := cbor.Marshal(typ, cbor.EncOptions{})
	if err != nil {
		return err
	}

	p, err := c.SendQUIC(QuicCmdGet, req)
	if err != nil {
		return err
	}

	var inTyp QuicCommand
	dec := cbor.NewDecoder(bytes.NewReader(p.payload))
	if err := dec.Decode(&inTyp); err != nil {
		return err
	}
	if typ != inTyp {
		return fmt.Errorf("typ (%d) != inTyp (%d)", typ, inTyp)
	}

	if err := dec.Decode(v); err != nil {
		return err
	}

	return nil
}

func (c *Controller) SetQUIC(typ QuicCommand, v interface{}) error {
	req := new(bytes.Buffer)

	enc := cbor.NewEncoder(req, cbor.EncOptions{})
	if err := enc.Encode(&typ); err != nil {
		return err
	}
	if err := enc.Encode(v); err != nil {
		return err
	}

	p, err := c.SendQUIC(QuicCmdSet, req.Bytes())
	if err != nil {
		return err
	}

	var inTyp QuicCommand
	dec := cbor.NewDecoder(bytes.NewReader(p.payload))
	if err := dec.Decode(&inTyp); err != nil {
		return err
	}
	if typ != inTyp {
		return fmt.Errorf("typ (%d) != inTyp (%d)", typ, inTyp)
	}

	if err := dec.Decode(v); err != nil {
		return err
	}

	return nil
}

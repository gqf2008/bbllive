package rtmp

import (
	"bbllive/util"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	AMF_NUMBER      = 0x00
	AMF_BOOLEAN     = 0x01
	AMF_STRING      = 0x02
	AMF_OBJECT      = 0x03
	AMF_NULL        = 0x05
	AMF_ARRAY_NULL  = 0x06
	AMF_MIXED_ARRAY = 0x08
	AMF_END         = 0x09
	AMF_ARRAY       = 0x0a

	AMF_INT8     = 0x0100
	AMF_INT16    = 0x0101
	AMF_INT32    = 0x0102
	AMF_VARIANT_ = 0x0103
)

type Object interface{}

type Map map[string]Object

func newMap() Map {
	return make(Map, 0)
}

type AMF struct {
	out *bytes.Buffer
	in  *bytes.Buffer
}

func newEncoder() (amf *AMF) {
	amf = new(AMF)
	amf.out = new(bytes.Buffer)
	return amf
}

func newDecoder(b []byte) (amf *AMF) {
	amf = new(AMF)
	amf.in = bytes.NewBuffer(b)
	return amf
}

func (amf *AMF) readAll() (obj []Object, err error) {
	obj = make([]Object, 0)
	for amf.in.Len() > 0 {
		v, err := amf.readData()
		if err != nil {
			break
		}
		obj = append(obj, v)
	}
	return
}

func (amf *AMF) writeAll(obj []Object) error {
	for _, v := range obj {
		if t, ok := v.(string); ok {
			return amf.writeString(t)
		} else if t, ok := v.(float64); ok {
			return amf.writeNumber(t)
		} else if t, ok := v.(bool); ok {
			return amf.writeBool(t)
		} else if t, ok := v.(Map); ok {
			return amf.writeMap(t)
		} else if v == nil {
			return amf.writeNull()
		} else {

		}
	}
	return nil
}

func (amf *AMF) readData() (obj Object, err error) {
	buf := amf.in
	if buf.Len() == 0 {
		return nil, errors.New(fmt.Sprintf("no enough bytes, %v/%v", buf.Len(), 1))
	}
	t, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	err = buf.UnreadByte()
	if err != nil {
		return nil, err
	}
	switch t {
	case 00:
		return amf.readNumber()
	case 01:
		return amf.readBool()
	case 02:
		return amf.readString()
	case 03:
		return amf.readObject()
	case 05:
		return amf.readNull()
	case 06:
		buf.ReadByte()
		return "Undefined", nil
	case 8:
		return amf.readMixArray()
	case 9:
		buf.ReadByte()
		return "ObjectEnd", nil
	case 10:
		return amf.readArray()
	case 11:
		return amf.readDate()
	case 12:
		return amf.readLongString()
	case 15:
		return amf.readLongString()
	}
	return nil, errors.New(fmt.Sprintf("Unsupported type %v", t))
}

func readBytes(buf *bytes.Buffer, length int) (b []byte, err error) {
	b = make([]byte, length)
	i, err := buf.Read(b)
	if length != i {
		err = errors.New(fmt.Sprintf("not enough bytes,%v/%v", buf.Len(), length))
	}
	return
}
func (amf *AMF) readLongString() (str string, err error) {
	buf := amf.in
	buf.ReadByte()
	b, err := readBytes(buf, 4)
	l := util.BigEndian.Uint32(b)
	b, err = readBytes(buf, int(l))
	return string(b), err
}

func (amf *AMF) readDate() (t uint64, err error) {
	buf := amf.in
	buf.ReadByte()
	b, err := readBytes(buf, 8)
	t = util.BigEndian.Uint64(b)
	b, err = readBytes(buf, 2)
	return t, err
}

func (amf *AMF) readArray() (list []Object, err error) {
	buf := amf.in
	buf.ReadByte()
	list = make([]Object, 0)
	b, err := readBytes(buf, 4)
	size := int(util.BigEndian.Uint32(b))
	for i := 0; i < size; i++ {
		obj, err := amf.readData()
		if err != nil {
			break
		}
		list = append(list, obj)
	}
	return
}

func (amf *AMF) readString() (str string, err error) {
	buf := amf.in
	buf.ReadByte()
	b, err := readBytes(buf, 2)
	l := util.BigEndian.Uint16(b)
	b, err = readBytes(buf, int(l))
	return string(b), err
}

func (amf *AMF) readString1() (str string, err error) {
	buf := amf.in
	b, err := readBytes(buf, 2)
	l := util.BigEndian.Uint16(b)
	b, err = readBytes(buf, int(l))
	return string(b), err
}

func (amf *AMF) readNull() (Object, error) {
	amf.in.ReadByte()
	return nil, nil
}

func (amf *AMF) readNumber() (num float64, err error) {
	buf := amf.in
	buf.ReadByte()
	err = binary.Read(buf, binary.BigEndian, &num)
	return num, err
}

func (amf *AMF) readBool() (f bool, err error) {
	buf := amf.in
	buf.ReadByte()
	b, err := buf.ReadByte()
	if b == 1 {
		f = true
	}
	return f, err
}

func (amf *AMF) readObject() (m Map, err error) {
	buf := amf.in
	buf.ReadByte()
	m = make(Map, 0)
	for {
		k, err := amf.readString1()
		if err != nil {
			break
		}
		v, err := amf.readData()
		if err != nil {
			break
		}
		if k == "" && "ObjectEnd" == v {
			break
		}
		m[k] = v
	}
	return m, err
}

func (amf *AMF) readMixArray() (m Map, err error) {
	buf := amf.in
	m = make(Map, 0)
	b, err := readBytes(buf, 4)
	size := int(util.BigEndian.Uint32(b))
	for i := 0; i < size; i++ {
		k, err := amf.readString1()
		if err != nil {
			break
		}
		v, err := amf.readData()
		if err != nil {
			break
		}
		if k == "" && "ObjectEnd" == v {
			break
		}
		m[k] = v
	}
	return m, err
}

func (amf *AMF) writeString(value string) error {
	buf := amf.out
	v := []byte(value)
	err := buf.WriteByte(byte(AMF_STRING))
	if err != nil {
		return err
	}
	b := make([]byte, 2)
	util.BigEndian.PutUint16(b, uint16(len(v)))
	_, err = buf.Write(b)
	if err != nil {
		return err
	}
	_, err = buf.Write(v)
	return err
}

func (amf *AMF) writeNull() error {
	buf := amf.out
	return buf.WriteByte(byte(AMF_NULL))
}

func (amf *AMF) writeBool(b bool) error {
	buf := amf.out
	err := buf.WriteByte(byte(AMF_BOOLEAN))
	if err != nil {
		return err
	}
	if b {
		return buf.WriteByte(byte(1))
	}
	return buf.WriteByte(byte(0))
}

func (amf *AMF) writeNumber(b float64) error {
	buf := amf.out
	err := buf.WriteByte(byte(AMF_NUMBER))
	if err != nil {
		return err
	}
	return binary.Write(buf, binary.BigEndian, b)
}

func (amf *AMF) writeObject() error {
	buf := amf.out
	return buf.WriteByte(byte(AMF_OBJECT))
}

func (amf *AMF) writeObjectString(key, value string) error {
	buf := amf.out
	b := make([]byte, 2)
	util.BigEndian.PutUint16(b, uint16(len([]byte(key))))
	_, err := buf.Write(b)
	if err != nil {
		return err
	}
	_, err = buf.Write([]byte(key))
	if err != nil {
		return err
	}
	return amf.writeString(value)
}

func (amf *AMF) writeObjectBool(key string, f bool) error {
	buf := amf.out
	b := make([]byte, 2)
	util.BigEndian.PutUint16(b, uint16(len([]byte(key))))
	_, err := buf.Write(b)
	if err != nil {
		return err
	}
	_, err = buf.Write([]byte(key))
	if err != nil {
		return err
	}
	return amf.writeBool(f)
}

func (amf *AMF) writeObjectNumber(key string, value float64) error {
	buf := amf.out
	b := make([]byte, 2)
	util.BigEndian.PutUint16(b, uint16(len([]byte(key))))
	_, err := buf.Write(b)
	if err != nil {
		return err
	}
	_, err = buf.Write([]byte(key))
	if err != nil {
		return err
	}
	return amf.writeNumber(value)
}

func (amf *AMF) writeObjectEnd() error {
	buf := amf.out
	err := buf.WriteByte(byte(0))
	if err != nil {
		return err
	}
	err = buf.WriteByte(byte(0))
	if err != nil {
		return err
	}
	return buf.WriteByte(byte(AMF_END))
}

func (amf *AMF) writeMap(t Map) (err error) {
	amf.writeObject()
	for k, vv := range t {
		if vvv, ok := vv.(string); ok {
			err = amf.writeObjectString(k, vvv)
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(float64); ok {
			err = amf.writeObjectNumber(k, vvv)
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(bool); ok {
			err = amf.writeObjectBool(k, vvv)
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(int); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(int16); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(int32); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(int64); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(uint16); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(uint32); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(uint64); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(uint8); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(int8); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		} else if vvv, ok := vv.(byte); ok {
			err = amf.writeObjectNumber(k, float64(vvv))
			if err != nil {
				break
			}
		}
	}
	amf.writeObjectEnd()
	return
}

func (amf *AMF) Bytes() []byte {
	return amf.out.Bytes()
}

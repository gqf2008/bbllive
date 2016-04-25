package codec

import (
	"errors"
)

//哥伦布解码
func GetUev(buff []byte, start int) (value int, pos int) {
	l := len(buff)
	var nZeroNum uint = 0
	for start < l*8 {
		if (buff[start/8] & (0x80 >> uint(start%8))) > 0 {
			break
		}
		nZeroNum += 1
		start += 1
	}
	dwRet := 0
	start += 1
	var i uint
	for i = 0; i < nZeroNum; i++ {
		dwRet <<= 1
		if (buff[start/8] & (0x80 >> uint(start%8))) > 0 {
			dwRet += 1
		}
		start += 1
	}
	return (1 << nZeroNum) - 1 + dwRet, start
}

func GetSliceType(nalu []byte) int {
	_, p := GetUev(nalu, 8)
	v, _ := GetUev(nalu, p)
	return v
}

func IsIDR(nalu []byte) bool {
	return (nalu[0] & 0x1f) == 0x05
}
func IsISlice(nalu []byte) bool {
	if (nalu[0] & 0x1f) == 0x05 {
		return true
	}
	if (nalu[0] & 0x1f) == 0x01 {
		st := GetSliceType(nalu)
		if st == 2 || st == 4 || st == 7 || st == 9 {
			return true
		}
	}
	return false
}

func IsPSlice(nalu []byte) bool {
	if (nalu[0] & 0x1f) == 0x01 {
		st := GetSliceType(nalu)
		if st == 0 || st == 3 || st == 5 || st == 8 {
			return true
		}
	}
	return false
}

func IsBSlice(nalu []byte) bool {
	if (nalu[0] & 0x1f) == 0x01 {
		st := GetSliceType(nalu)
		if st == 1 || st == 6 {
			return true
		}
	}
	return false
}

func IsSPS(nalu []byte) bool {
	return (nalu[0] & 0x1f) == 0x07
}
func IsPPS(nalu []byte) bool {
	return (nalu[0] & 0x1f) == 0x08
}

func ReadH264Nalu(buf []byte) (packet []byte, reset []byte, err error) {
	l := len(buf)
	if l < 3 {
		err = errors.New("invalid h264 nal format")
		return
	}

	//0x00 00 00 01 09 00 00 00 01 05... 00 00 01 01...
	//0x000001
	start := 0
	end := 0
	if buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x01 {
		start = 3
	} else if buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x00 && buf[3] == 0x01 {
		start = 4
	} else {
		err = errors.New("invalid h264 nal format")
		return
	}

	for end = start; end < l; end++ {
		if buf[end] == 0x01 && buf[end-1] == 0x00 && buf[end-2] == 0x00 && buf[end-3] == 0x00 {
			end -= 3
			break
		} else if buf[end] == 0x01 && buf[end-1] == 0x00 && buf[end-2] == 0x00 {
			end -= 2
			break
		}
	}
	if end == l {
		return buf[start:], nil, nil
	}
	return buf[start:end], buf[end:], nil
}

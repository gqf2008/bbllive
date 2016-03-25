package rtmp

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math/rand"
)

const (
	SHA256_DIGEST_LENGTH = 32
	HANDSHAKE_SIZE       = 1536
)

var (
	GENUINE_FMS_KEY = []byte{
		0x47, 0x65, 0x6e, 0x75, 0x69, 0x6e, 0x65, 0x20,
		0x41, 0x64, 0x6f, 0x62, 0x65, 0x20, 0x46, 0x6c,
		0x61, 0x73, 0x68, 0x20, 0x4d, 0x65, 0x64, 0x69,
		0x61, 0x20, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
		0x20, 0x30, 0x30, 0x31, // Genuine Adobe Flash Media Server 001
		0xf0, 0xee, 0xc2, 0x4a, 0x80, 0x68, 0xbe, 0xe8,
		0x2e, 0x00, 0xd0, 0xd1, 0x02, 0x9e, 0x7e, 0x57,
		0x6e, 0xec, 0x5d, 0x2d, 0x29, 0x80, 0x6f, 0xab,
		0x93, 0xb8, 0xe6, 0x36, 0xcf, 0xeb, 0x31, 0xae,
	}
	GENUINE_FP_KEY = []byte{
		0x47, 0x65, 0x6E, 0x75, 0x69, 0x6E, 0x65, 0x20,
		0x41, 0x64, 0x6F, 0x62, 0x65, 0x20, 0x46, 0x6C,
		0x61, 0x73, 0x68, 0x20, 0x50, 0x6C, 0x61, 0x79,
		0x65, 0x72, 0x20, 0x30, 0x30, 0x31, /* Genuine Adobe Flash Player 001 */
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8,
		0x2E, 0x00, 0xD0, 0xD1, 0x02, 0x9E, 0x7E, 0x57,
		0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
)

func handshake1(rw *bufio.ReadWriter) bool {
	buf := bytes.NewBuffer(nil)
	_, err := io.CopyN(buf, rw, HANDSHAKE_SIZE+1)
	if err != nil {
		log.Error("handshake1", err)
		return false
	}
	b := buf.Bytes() //ReadBuf(rw, HANDSHAKE_SIZE+1)
	if b[0] != 0x3 {
		log.Warn("C0 error ", b[0])
		return false
	}
	if len(b[1:]) != HANDSHAKE_SIZE {
		log.Warn("C1 error ", len(b[1:]))
		return false
	}
	input := make([]byte, HANDSHAKE_SIZE)
	copy(input, b[1:])
	ver := input[4] & 0xff
	if ver == 0 {
		return simple_handshake(rw, input)
	}
	return complex_handshake(rw, input)
}

func simple_handshake(rw *bufio.ReadWriter, input []byte) bool {
	s1 := make([]byte, HANDSHAKE_SIZE-4)
	s2 := input
	uptime := uint32(136793223)
	buf := new(bytes.Buffer)
	buf.WriteByte(0x03)
	binary.Write(buf, binary.BigEndian, uptime)
	buf.Write(s1)
	buf.Write(s2)
	rw.Write(buf.Bytes())
	rw.Flush()
	ReadBuf(rw, HANDSHAKE_SIZE)
	return true
}

func client_simple_handshake(rw *bufio.ReadWriter) bool {
	c1 := make([]byte, HANDSHAKE_SIZE-4)
	uptime := uint32(136793223)
	buf := new(bytes.Buffer)
	buf.WriteByte(0x03)
	binary.Write(buf, binary.BigEndian, uptime)
	buf.Write(c1)
	rw.Write(buf.Bytes())
	rw.Flush()
	s0 := ReadBuf(rw, 1)
	if s0[0] != 0x03 {
		return false
	}
	s1 := ReadBuf(rw, HANDSHAKE_SIZE)
	c2 := s1
	rw.Write(c2)
	rw.Flush()
	ReadBuf(rw, HANDSHAKE_SIZE)
	return true
}

func complex_handshake(rw *bufio.ReadWriter, input []byte) bool {
	result, scheme, challenge, digest := validateClient(input)
	if !result {
		return result
	}
	log.Debugf("Validate Client %v scheme %v challenge %0X digest %0X", result, scheme, challenge, digest)
	s1 := create_s1()
	log.Debug("s1 length", len(s1))
	off := getDigestOffset(s1, scheme)
	log.Debug("s1 digest offset", off)
	buf := new(bytes.Buffer)
	buf.Write(s1[:off])
	buf.Write(s1[off+32:])
	tempHash, _ := HMACsha256(buf.Bytes(), GENUINE_FMS_KEY[:36])
	copy(s1[off:], tempHash)
	log.Debug("s1 length", len(s1))
	//compute the challenge digest
	tempHash, _ = HMACsha256(digest, GENUINE_FMS_KEY[:68])
	log.Debug("s2 length tempHash", len(tempHash))
	randBytes := create_s2()
	log.Debug("s2 length", len(randBytes))
	lastHash, _ := HMACsha256(randBytes, tempHash)
	log.Debug("s2 length lastHash", len(lastHash))
	log.Debug("s2 length", len(randBytes))
	buf = new(bytes.Buffer)
	buf.WriteByte(0x03)
	buf.Write(s1)
	buf.Write(randBytes)
	buf.Write(lastHash)
	log.Debug("send s0s1s2", buf.Len())
	rw.Write(buf.Bytes())
	rw.Flush()
	ReadBuf(rw, HANDSHAKE_SIZE)
	return true
}

func create_s2() []byte {
	rndBytes := make([]byte, HANDSHAKE_SIZE-32)
	for i, _ := range rndBytes {
		rndBytes[i] = byte(rand.Int() % 256)
	}
	return rndBytes
}

func create_s1() []byte {
	s1 := []byte{0, 0, 0, 0, 1, 2, 3, 4}
	rndBytes := make([]byte, HANDSHAKE_SIZE-8)
	for i, _ := range rndBytes {
		rndBytes[i] = byte(rand.Int() % 256)
	}
	s1 = append(s1, rndBytes...)
	return s1
}

func validateClient(input []byte) (result bool, scheme int, challenge []byte, digest []byte) {
	if result, scheme, challenge, digest = validateClientScheme(input, 1); result {
		return
	}
	if result, scheme, challenge, digest = validateClientScheme(input, 0); result {
		return
	}
	log.Warn("Unable to validate client")
	return
}

func validateClientScheme(pBuffer []byte, scheme int) (result bool, schem int, challenge []byte, digest []byte) {
	digest_offset := -1
	challenge_offset := -1
	if scheme == 0 {
		digest_offset = getDigestOffset0(pBuffer)
		challenge_offset = getDHOffset0(pBuffer)
	} else if scheme == 1 {
		digest_offset = getDigestOffset1(pBuffer)
		challenge_offset = getDHOffset1(pBuffer)
	}
	p1 := pBuffer[:digest_offset]
	digest = pBuffer[digest_offset : digest_offset+32]
	p2 := pBuffer[digest_offset+32:]
	buf := new(bytes.Buffer)
	buf.Write(p1)
	buf.Write(p2)
	p := buf.Bytes()
	log.Debugf("Scheme: {%v} client digest offset: {%v}", scheme, digest_offset)
	tempHash, _ := HMACsha256(p, GENUINE_FP_KEY[:30])
	log.Debugf("Temp: {%0X}", tempHash)
	log.Debugf("Dig : {%0X}", digest)
	result = bytes.Compare(digest, tempHash) == 0
	challenge = pBuffer[challenge_offset : challenge_offset+128]
	schem = scheme
	return
}

func HMACsha256(msgBytes []byte, key []byte) ([]byte, error) {
	h := hmac.New(sha256.New, key)
	_, err := h.Write(msgBytes)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
func getDigestOffset(pBuffer []byte, scheme int) int {
	if scheme == 1 {
		return getDigestOffset1(pBuffer)
	} else if scheme == 0 {
		return getDigestOffset0(pBuffer)
	}
	return -1
}

/**
 * Returns a digest byte offset.
 *
 * @param pBuffer source for digest data
 * @return digest offset
 */
func getDigestOffset0(pBuffer []byte) int {
	offset := int(pBuffer[8]&0xff) + int(pBuffer[9]&0xff) + int(pBuffer[10]&0xff) + int(pBuffer[11]&0xff)
	offset = (offset % 728) + 8 + 4
	if offset+32 >= 1536 {
		log.Warn("Invalid digest offset")
	}
	log.Debug("digest offset", offset)
	return offset
}

/**
 * Returns a digest byte offset.
 *
 * @param pBuffer source for digest data
 * @return digest offset
 */
func getDigestOffset1(pBuffer []byte) int {
	offset := int(pBuffer[772]&0xff) + int(pBuffer[773]&0xff) + int(pBuffer[774]&0xff) + int(pBuffer[775]&0xff)
	offset = (offset % 728) + 772 + 4
	if offset+32 >= 1536 {
		log.Warn("Invalid digest offset")
	}
	log.Debug("digest offset", offset)
	return offset
}

func getDHOffset(handshakeBytes []byte, scheme int) int {
	if scheme == 0 {
		return getDHOffset0(handshakeBytes)
	} else if scheme == 1 {
		return getDHOffset1(handshakeBytes)
	}
	return -1
}

/**
 * Returns the DH byte offset.
 *
 * @return dh offset
 */
func getDHOffset0(handshakeBytes []byte) int {
	offset := int(handshakeBytes[1532]) + int(handshakeBytes[1533]) + int(handshakeBytes[1534]) + int(handshakeBytes[1535])
	offset = (offset % 632) + 772
	if offset+128 >= 1536 {
		log.Warn("Invalid DH offset")
	}
	return offset
}

/**
 * Returns the DH byte offset.
 *
 * @return dh offset
 */
func getDHOffset1(handshakeBytes []byte) int {
	offset := int(handshakeBytes[768]) + int(handshakeBytes[769]) + int(handshakeBytes[770]) + int(handshakeBytes[771])
	offset = (offset % 632) + 8
	if offset+128 >= 1536 {
		log.Warn("Invalid DH offset")
	}
	return offset
}

func ReadBuf(r io.Reader, n int) (b []byte) {
	b = make([]byte, n)
	_, err := io.ReadFull(r, b)
	if err != nil {
		log.Error("ReadBuf", err)
	}
	return
}

package transport

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/saichler/k8sman/go/common"
	utils "github.com/saichler/utils/golang"
	"io"
	"net"
	"strings"
	"time"
)

const (
	MAX_SIZE     = 1024 * 1024 * 50
	LARGE_PACKET = 1024 * 1024 * 5
)

func long2Bytes(s int64) []byte {
	size := make([]byte, 8)
	size[7] = byte(s)
	size[6] = byte(s >> 8)
	size[5] = byte(s >> 16)
	size[4] = byte(s >> 24)
	size[3] = byte(s >> 32)
	size[2] = byte(s >> 40)
	size[1] = byte(s >> 48)
	size[0] = byte(s >> 56)
	return size
}

func bytes2Long(data []byte) int64 {
	v1 := int64(data[0]) << 56
	v2 := int64(data[1]) << 48
	v3 := int64(data[2]) << 40
	v4 := int64(data[3]) << 32
	v5 := int64(data[4]) << 24
	v6 := int64(data[5]) << 16
	v7 := int64(data[6]) << 8
	v8 := int64(data[7])
	return v1 + v2 + v3 + v4 + v5 + v6 + v7 + v8
}

func encode(dataToEncode []byte, key string) (string, error) {
	k := []byte(key)
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}

	l := len(dataToEncode)
	cipherdata := make([]byte, aes.BlockSize+l)

	iv := cipherdata[:aes.BlockSize]
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(cipherdata[aes.BlockSize:], dataToEncode)
	return base64.StdEncoding.EncodeToString(cipherdata), nil
}

func decode(stringToDecode, key string) ([]byte, error) {
	encData, err := base64.StdEncoding.DecodeString(stringToDecode)
	if err != nil {
		return nil, err
	}
	if len(encData) < aes.BlockSize {
		err = errors.New("Encrypted data does not have an IV spec!")
		return nil, err
	}
	k := []byte(key)
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	iv := encData[:aes.BlockSize]
	encData = encData[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	data := make([]byte, len(encData))
	cfb.XORKeyStream(data, encData)
	return data, nil
}

func readPacket(conn net.Conn) ([]byte, error) {
	sizebytes, err := readData(8, conn)
	if sizebytes == nil || err != nil {
		return nil, err
	}
	size := bytes2Long(sizebytes)
	if size > MAX_SIZE {
		return nil, errors.New("Max Size Exceeded!")
	}
	data, err := readData(int(size), conn)
	return data, err
}

func readData(size int, conn net.Conn) ([]byte, error) {
	data := make([]byte, size)
	n, e := conn.Read(data)
	if e != nil {
		return nil, errors.New("Failed to read date size:" + e.Error())
	}

	if n < size {
		if n == 0 {
			time.Sleep(time.Second)
		}
		data = data[0:n]
		left, e := readData(size-n, conn)
		if e != nil {
			return nil, errors.New("Failed to read packet size:" + e.Error())
		}
		data = append(data, left...)
	}
	return data, nil
}

func writePacket(data []byte, conn net.Conn) error {
	if conn == nil {
		return errors.New("No Connection Available")
	}
	_, e := conn.Write(long2Bytes(int64(len(data))))
	if e != nil {
		return e
	}
	_, e = conn.Write(data)
	return e
}

func Send(pb proto.Message, c *Port) error {
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	return c.Send(data)
}

func LocalIps() (map[string]string, error) {
	netIfs, err := net.Interfaces()
	if err != nil {
		return nil, errors.New(common.Con("Could not fetch local interfaces:", err.Error()))
	}
	result := make(map[string]string)
	for _, netIf := range netIfs {
		addrs, err := netIf.Addrs()
		if err != nil {
			utils.Error(common.Con("Failed to fetch addresses for net interface:", err.Error()))
			continue
		}
		for _, addr := range addrs {
			addrString := addr.String()
			index := strings.Index(addrString, "/")
			result[addrString[0:index]] = addrString[0:index]
		}
	}
	return result, nil
}

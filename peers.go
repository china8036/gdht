package gdht

import (
	"net"
	"time"
	"bytes"
	"qiniupkg.com/x/errors.v7"
	"io"
	"encoding/binary"
	"strings"
	"strconv"
	"log"
	"fmt"
)

const
(
	Choke         = 0 //没有有效载荷
	UnChoke       = 1 //没有有效载荷
	Interested    = 2 //没有有效载荷
	NotInterested = 3 //没有有效载荷
	Have          = 4
	BitField      = 5
	Request       = 6
	Piece         = 7
	Cancel        = 8
	// EXTENDED represents it is a extended message
	EXTENDED = 20
	// HANDSHAKE represents handshake bit
	HANDSHAKE = 0
	BLOCK     = 16384 //16个字节
)

//peers 协议
//握手：
// 开头19 紧跟 "BitTorrent protocol" 然后跟8个保留字节用于扩展
// 20 bytes sha1 infohash + 20 bytes peer id
//信息:
//接下来是长度前缀和信息的交替排序 0长度的信息表示为保持连接 并且将被忽略 保持连接信息一般没两分钟发送一次 但是当想要获取数据时候超时时间将会更短
//所有不是保持连接的信息开头发送一个字节的下列信息 0 - choke  1 - unchoke 2 - interested 3 - not interested 4 - have 5 - bitfield
//6 - request 7 - piece 8 - cancel
var handshakePrefix = []byte{
	19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114,
	111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 16, 0, 1,
} //设备控制符 BitTorrent protocol nil nil nil nil nil 数据链路转义 nil 标题开始
//最后八个保留字节 全部为0 用于扩展  八个保留字节中 从右到左第20个bit被用来检测是不是扩展协议 也就是倒数第三个字节里的00001000(ox10)16也就是十进制的16
//

func GetMetaInfo(ih, peer_id, address string) error {
	dial, err := net.DialTimeout("tcp", address, time.Second*15) //准备发起tcp请求
	if err != nil {
		log.Println(err)
		return err
	}
	conn := dial.(*net.TCPConn) //断言为tcp链接
	conn.SetLinger(0)
	defer conn.Close()
	conn.SetLinger(0)
	data := bytes.NewBuffer(nil)
	data.Grow(BLOCK)
	err = SendHandshake(conn, []byte(ih), []byte(peer_id)) //peers标准内容
	if err != nil {
		log.Println(err)
		return err
	}
	err = SendExtHandshake(conn) //扩展内容
	if err != nil {
		log.Println(err)
		return err
	}
	err = Read(conn, 68, data)
	if err != nil {
		log.Println(err)
		return err
	}
	err = OnHandshake(data.Next(68))
	if err != nil {
		log.Println(err)
		return err
	}
	err = SendExtHandshake(conn)
	if err != nil {
		log.Println(err)
		return err
	}
	for {
		length, err := ReadMessage(conn, data)
		if err != nil {
			return err
		}

		if length == 0 {
			continue
		}

		msgType, err := data.ReadByte()
		if err != nil {
			return err
		}
		log.Println(msgType)
	}
	return nil
}

// 握手信息
func SendHandshake(conn *net.TCPConn, infoHash, peerID []byte) error {
	data := make([]byte, 68)
	copy(data[:28], handshakePrefix) //BitTorrent protocol
	copy(data[28:48], infoHash)      //infohash
	copy(data[48:], peerID)          //自己的peerid

	conn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	fmt.Println(data)
	_, err := conn.Write(data)
	return err
}

// readMessage gets a message from the tcp connection.
func ReadMessage(conn *net.TCPConn, data *bytes.Buffer) (
	length int, err error) {

	if err = Read(conn, 4, data); err != nil {
		return
	}

	length = int(bytes2int(data.Next(4)))
	if length == 0 {
		return
	}

	if err = Read(conn, length, data); err != nil {
		return
	}
	return
}

// read reads size-length bytes from conn to data.
func Read(conn *net.TCPConn, size int, data *bytes.Buffer) error {
	conn.SetReadDeadline(time.Now().Add(time.Second * 15))

	n, err := io.CopyN(data, conn, int64(size))
	if err != nil || n != int64(size) {
		return errors.New("read error")
	}
	return nil
}

// onHandshake handles the handshake response.
func OnHandshake(data []byte) (err error) {
	if !(bytes.Equal(handshakePrefix[:20], data[:20]) && data[25]&0x10 != 0) { //
		err = errors.New("invalid handshake response")
	}
	return
}

// sendExtHandshake requests for the ut_metadata and metadata_size.
func SendExtHandshake(conn *net.TCPConn) error {
	data := append(
		[]byte{EXTENDED, HANDSHAKE},
		Encode(map[string]interface{}{
			"m": map[string]interface{}{"ut_metadata": 1},
		})...,
	)

	return sendMessage(conn, data)
}

// sendMessage sends data to the connection.
func sendMessage(conn *net.TCPConn, data []byte) error {
	length := int32(len(data))

	buffer := bytes.NewBuffer(nil)
	binary.Write(buffer, binary.BigEndian, length)

	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	_, err := conn.Write(append(buffer.Bytes(), data...))
	return err
}

// Encode encodes a string, int, dict or list value to a bencoded string.
func Encode(data interface{}) string {
	switch v := data.(type) {
	case string:
		return EncodeString(v)
	case int:
		return EncodeInt(v)
	case []interface{}:
		return EncodeList(v)
	case map[string]interface{}:
		return EncodeDict(v)
	default:
		panic("invalid type when encode")
	}
}

// EncodeString encodes a string value.
func EncodeString(data string) string {
	return strings.Join([]string{strconv.Itoa(len(data)), data}, ":")
}

// EncodeInt encodes a int value.
func EncodeInt(data int) string {
	return strings.Join([]string{"i", strconv.Itoa(data), "e"}, "")
}

// EncodeItem encodes an item of dict or list.
func encodeItem(data interface{}) (item string) {
	switch v := data.(type) {
	case string:
		item = EncodeString(v)
	case int:
		item = EncodeInt(v)
	case []interface{}:
		item = EncodeList(v)
	case map[string]interface{}:
		item = EncodeDict(v)
	default:
		panic("invalid type when encode item")
	}
	return
}

// EncodeList encodes a list value.
func EncodeList(data []interface{}) string {
	result := make([]string, len(data))

	for i, item := range data {
		result[i] = encodeItem(item)
	}

	return strings.Join([]string{"l", strings.Join(result, ""), "e"}, "")
}

// EncodeDict encodes a dict value.
func EncodeDict(data map[string]interface{}) string {
	result, i := make([]string, len(data)), 0

	for key, val := range data {
		result[i] = strings.Join(
			[]string{EncodeString(key), encodeItem(val)},
			"")
		i++
	}

	return strings.Join([]string{"d", strings.Join(result, ""), "e"}, "")
}

// bytes2int returns the int value it represents.
func bytes2int(data []byte) uint64 {
	n, val := len(data), uint64(0)
	if n > 8 {
		panic("data too long")
	}

	for i, b := range data {
		val += uint64(b) << uint64((n-i-1)*8)
	}
	return val
}

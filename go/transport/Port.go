package transport

import (
	"errors"
	"github.com/google/uuid"
	utils "github.com/saichler/utils/golang"
	"net"
	"strconv"
	"sync"
)

type Port struct {
	key          string
	uuid         string
	zside        string
	rx           *DataQueue
	tx           *DataQueue
	nx           *DataQueue
	writeMutex   *sync.Cond
	conn         net.Conn
	active       bool
	dataListener DataListener
	name         string
}

func newPort(con net.Conn, key string, dataListener DataListener, maxInputQueueSize, maxOutputQueueSize int) *Port {
	port := &Port{}
	port.uuid = uuid.New().String()
	port.conn = con
	port.name = con.RemoteAddr().String()
	port.rx = NewDataQueue("RX Queue "+port.name, maxInputQueueSize)
	port.tx = NewDataQueue("TX Queue "+port.name, maxOutputQueueSize)
	port.nx = NewDataQueue("NX Queue "+port.name, maxInputQueueSize)
	port.active = true
	port.key = key
	port.dataListener = dataListener
	port.writeMutex = sync.NewCond(&sync.Mutex{})
	return port
}

func ConnectTo(host, key, secret string, destPort int, dataListener DataListener, maxIn, maxOut, notifiers int) (*Port, error) {
	conn, err := net.Dial("tcp", host+":"+strconv.Itoa(destPort))
	if err != nil {
		return nil, err
	}

	data, err := encode([]byte(secret), key)
	if err != nil {
		return nil, err
	}

	err = writePacket([]byte(data), conn)
	if err != nil {
		return nil, err
	}

	inData, err := readPacket(conn)
	if string(inData) != "OK" {
		return nil, errors.New("Failed to connect, incorrect Key/Secret")
	}

	port := newPort(conn, key, dataListener, maxIn, maxOut)

	data, err = encode([]byte(secret), port.uuid)
	if err != nil {
		return nil, err
	}

	err = writePacket([]byte(data), conn)
	if err != nil {
		return nil, err
	}

	inData, err = readPacket(conn)
	port.zside = string(inData)

	go port.read()
	go port.write()
	go port.process()
	if notifiers <= 0 {
		go port.notifier()
	} else {
		for i := 0; i < notifiers; i++ {
			go port.notifier()
		}
	}
	return port, nil
}

func (port *Port) read() {
	for port.active {
		packet, err := readPacket(port.conn)
		if err != nil {
			utils.Error(err)
			break
		}
		if packet != nil {
			if len(packet) == 2 && string(packet) == "WC" {
				port.writeMutex.L.Lock()
				port.writeMutex.Broadcast()
				port.writeMutex.L.Unlock()
				continue
			} else if len(packet) >= LARGE_PACKET {
				/*
					p.writeMutex.L.Lock()
					writePacket([]byte("WC"), p.conn)
					p.writeMutex.L.Unlock()
				*/
			}
			if port.active {
				port.rx.Add(packet)
			}
		} else {
			break
		}
	}
	utils.Info("Connection Read for ", port.name, " ended.")
	port.Shutdown()
}

func (port *Port) Shutdown() {
	port.active = false
	if port.conn != nil {
		port.conn.Close()
	}
	port.rx.Shutdown()
	port.tx.Shutdown()
	port.nx.Shutdown()
	port.writeMutex.Broadcast()
}

func (port *Port) write() {
	for port.active {
		packet := port.tx.Next()
		if packet != nil {
			port.writeMutex.L.Lock()
			if port.active {
				writePacket(packet, port.conn)
			}
			if len(packet) >= LARGE_PACKET {
				//c.writeMutex.Wait()
			}
			port.writeMutex.L.Unlock()
		} else {
			break
		}
	}
	utils.Info("Connection Write for ", port.name, " ended.")
	port.Shutdown()
}

func (port *Port) Send(data []byte) error {
	if port.active {
		encData, err := encode(data, port.key)
		if err != nil {
			return err
		}
		port.tx.Add([]byte(encData))
	}
	return nil
}

func (port *Port) process() {
	for port.active {
		packet := port.rx.Next()
		if packet != nil {
			encString := string(packet)
			data, err := decode(encString, port.key)
			if err != nil {
				break
			}
			port.nx.Add(data)
		}
	}
	utils.Info("Message Processing for ", port.name, " Ended")
	port.Shutdown()
}

func (port *Port) notifier() {
	for port.active {
		data := port.nx.Next()
		if data != nil && port.dataListener != nil {
			port.dataListener.DataReceived(data, port)
		}
	}
}

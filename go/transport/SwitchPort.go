package transport

import (
	"errors"
	"github.com/google/uuid"
	"github.com/saichler/k8sman/go/common"
	utils "github.com/saichler/utils/golang"
	"net"
	"strconv"
	"sync"
)

type SwitchPort struct {
	uuid    string
	socket  net.Listener
	mtx     *sync.Mutex
	_switch *Switch
}

func newSwitchPort(_switch *Switch) *SwitchPort {
	switchPort := &SwitchPort{}
	switchPort._switch = _switch
	switchPort.uuid = uuid.New().String()
	return switchPort
}

func (switchPort *SwitchPort) bind() error {
	socket, e := net.Listen("tcp", ":"+strconv.Itoa(switchPort._switch.port))

	if e != nil {
		panic(common.Con("Unable to bind to port ", switchPort._switch.port, e.Error()))
	}
	utils.Info(common.Con("Bind Successfully to port ", switchPort._switch.port))
	switchPort.socket = socket
	switchPort.mtx = &sync.Mutex{}
	return nil
}

func (switchPort *SwitchPort) start() error {
	if switchPort._switch.port == 0 {
		return errors.New("Switch Port does not have a port defined")
	}
	if switchPort._switch.secret == "" {
		return errors.New("Switch Port does not have a secret")
	}
	if switchPort._switch.key == "" {
		return errors.New("Switch Port does not have a key")
	}
	err := switchPort.bind()
	if err != nil {
		return err
	}
	for switchPort._switch.active {
		utils.Info("Waiting for connections...")
		conn, e := switchPort.socket.Accept()
		if e != nil {
			utils.Error("Failed to accept socket connection:", err)
			continue
		}
		utils.Info("Accepted socket connection...")
		switchPort.connectPort(conn)
	}
	return nil
}

func (switchPort *SwitchPort) connectPort(conn net.Conn) {
	port, err := switchPort.incoming(conn)
	if err != nil {
		utils.Error("Failed to connect to ", conn.RemoteAddr().String(), " ", err)
	}
	switchPort._switch.addPort(port)
}

func (switchPort *SwitchPort) incoming(conn net.Conn) (*Port, error) {
	port := newPort(conn, switchPort._switch.key, switchPort._switch, switchPort._switch.maxPortQueue, switchPort._switch.maxPortQueue)
	initData, err := readPacket(port.conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	data, err := decode(string(initData), switchPort._switch.key)
	if err != nil {
		conn.Close()
		return nil, err
	}

	if string(data) != switchPort._switch.secret {
		conn.Close()
		return nil, errors.New("Incorrect Secret/Key, aborting connection")
	}

	err = writePacket([]byte("OK"), conn)
	if err != nil {
		conn.Close()
		return nil, errors.New("Failed to write response")
	}

	initData, err = readPacket(port.conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	data, err = decode(string(initData), switchPort._switch.key)
	if err != nil {
		conn.Close()
		return nil, err
	}

	portUuid := string(data)
	port.uuid = portUuid

	err = writePacket([]byte(switchPort.uuid), conn)
	if err != nil {
		conn.Close()
		return nil, errors.New("Failed to write response")
	}

	go port.read()
	go port.write()
	go port.process()
	if switchPort._switch.notifiersPerPort <= 0 {
		go port.notifier()
	} else {
		for i := 0; i < switchPort._switch.notifiersPerPort; i++ {
			go port.notifier()
		}
	}
	return port, nil
}

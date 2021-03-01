package transport

import (
	"github.com/golang/protobuf/proto"
	"github.com/saichler/k8sman/go/pb"
	utils "github.com/saichler/utils/golang"
	"sync"
)

type Switch struct {
	secret           string
	key              string
	port             int
	maxPortQueue     int
	switchTable      map[string]*Port
	switchPort       *SwitchPort
	active           bool
	notifiersPerPort int
	mtx              *sync.Mutex
}

func NewSwitch(secret, key string, listenPort, maxQueuePerPort, notifiersPerPort int) (*Switch, error) {
	sw := &Switch{}
	sw.mtx = &sync.Mutex{}
	sw.switchTable = make(map[string]*Port)
	sw.secret = secret
	sw.key = key
	sw.port = listenPort
	sw.maxPortQueue = maxQueuePerPort
	sw.notifiersPerPort = notifiersPerPort
	sw.active = true
	return sw, nil
}

func (_switch *Switch) DataReceived(data []byte, port *Port) {
	packet := &pb.Packet{}
	err := proto.Unmarshal(data, packet)
	if err != nil {
		utils.Error("Unable to marshal packet:", err)
		return
	}

}

func (_switch *Switch) addPort(p *Port) {
	_switch.mtx.Lock()
	defer _switch.mtx.Unlock()
	ep, ok := _switch.switchTable[p.uuid]
	if ok {
		ep.Shutdown()
	}
	_switch.switchTable[p.uuid] = p
}

func (_switch *Switch) delPort(p *Port) {

}

func (_switch *Switch) switchPacket(packet *pb.Packet, data []byte, port *Port) {
	_switch.mtx.Lock()
	if packet.DestUuid != "" {
		destPort, ok := _switch.switchTable[packet.DestUuid]
		if ok {
			err := destPort.Send(data)
			if err != nil {
				utils.Error("Failed to send to destination:", err)
			}
		}
	}

}

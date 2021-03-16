package transport

import (
	"github.com/golang/protobuf/proto"
	"github.com/saichler/k8sman/go/pb"
	utils "github.com/saichler/utils/golang"
	"sync"
)

type Switch struct {
	secret              string
	key                 string
	port                int
	maxPortQueue        int
	localSwitchTable    map[string]*Port
	externalSwitchTable map[string]*Port
	switchPort          *SwitchPort
	active              bool
	notifiersPerPort    int
	mtx                 *sync.Mutex
	ias                 *IpAddressSegment
	multiCasted         map[string]map[int32]int32
	multiCastedSync     *sync.Mutex
}

func NewSwitch(secret, key string, listenPort, maxQueuePerPort, notifiersPerPort int) (*Switch, error) {
	sw := &Switch{}
	sw.ias = NewIpAddressSegment()
	sw.mtx = &sync.Mutex{}
	sw.localSwitchTable = make(map[string]*Port)
	sw.externalSwitchTable = make(map[string]*Port)
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
	_switch.switchPacket(packet, data, port)
}

func (_switch *Switch) addPort(p *Port) {
	_switch.mtx.Lock()
	defer _switch.mtx.Unlock()
	isLocal := _switch.ias.IsLocal(p.conn.LocalAddr().String())
	if isLocal {
		ep, ok := _switch.localSwitchTable[p.uuid]
		if ok {
			ep.Shutdown()
		}
		_switch.localSwitchTable[p.uuid] = p
	} else {
		ep, ok := _switch.externalSwitchTable[p.uuid]
		if ok {
			ep.Shutdown()
		}
		_switch.externalSwitchTable[p.uuid] = p
	}
}

func (_switch *Switch) delPort(p *Port) {

}

func (_switch *Switch) switchPacket(packet *pb.Packet, data []byte, port *Port) {
	if packet.DestUuid != "" {
		_switch.unicast(packet, data, port)
		return
	} else if packet.DestGroup != "" {
		_switch.multiCast(packet, data, port)
		return
	}
}

func (_switch *Switch) multiCast(packet *pb.Packet, data []byte, port *Port) {
	_switch.multiCastedSync.Lock()
	portMap, ok := _switch.multiCasted[packet.SourceUuid]
	if !ok {
		_switch.multiCasted[packet.SourceUuid] = make(map[int32]int32)
		portMap = _switch.multiCasted[packet.SourceUuid]
	}
	_, exist := portMap[packet.Sequence]
	if !exist {
		portMap[packet.Sequence] = packet.Sequence
	}
	_switch.multiCastedSync.Unlock()
	if exist {
		return
	}

	destination := _switch.multicastDestinations(packet.SwitchUuid)
	if packet.SwitchUuid == "" {
		packet.SwitchUuid = _switch.switchPort.uuid
		d, err := proto.Marshal(packet)
		if err != nil {
			utils.Error("Failed to marshal packet in multicast", err)
			return
		}
		data = d
	}
	for _, port := range destination {
		err := port.Send(data)
		if err != nil {
			utils.Error("Failed to send multicast to port:", port.uuid, ":", err)
		}
	}
}

func (_switch *Switch) unicast(packet *pb.Packet, data []byte, port *Port) {
	if packet.DestUuid == _switch.switchPort.uuid {
		_switch.myDataReceived(packet, port)
		return
	}
	_switch.mtx.Lock()
	destPortLocal, okLocal := _switch.localSwitchTable[packet.DestUuid]
	destPortExternal, okExternal := _switch.externalSwitchTable[packet.DestUuid]
	_switch.mtx.Unlock()
	if okLocal {
		err := destPortLocal.Send(data)
		if err != nil {
			utils.Error("Failed to send to local destination:", err)
		}
		return
	}

	if okExternal {
		err := destPortExternal.Send(data)
		if err != nil {
			utils.Error("Failed to send to external destination:", err)
		}
		return
	}

	utils.Error("No switch option available for destination:", packet.DestUuid)
}

func (_switch *Switch) multicastDestinations(switchUuid string) map[string]*Port {
	_switch.mtx.Lock()
	defer _switch.mtx.Unlock()
	result := make(map[string]*Port)
	for uuid, port := range _switch.localSwitchTable {
		result[uuid] = port
	}
	for uuid, port := range _switch.externalSwitchTable {
		if port.uuid != switchUuid {
			result[uuid] = port
		}
	}
	return result
}

func (_switch *Switch) myDataReceived(p *pb.Packet, port *Port) {

}

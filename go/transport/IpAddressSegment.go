package transport

import (
	"errors"
	"github.com/saichler/k8sman/go/common"
	utils "github.com/saichler/utils/golang"
	"net"
	"strings"
)

type IpAddressSegment struct {
	ip2InterfaceName map[string]string
	subnet2Local     map[string]bool
}

func NewIpAddressSegment() *IpAddressSegment {
	ias := &IpAddressSegment{}
	lip, err := LocalIps()
	if err != nil {
		utils.Error(err)
	}
	ias.ip2InterfaceName = lip
	ias.initSegment()
	return ias
}

func (ias *IpAddressSegment) initSegment() {
	ias.subnet2Local = make(map[string]bool)
	for ip, name := range ias.ip2InterfaceName {
		if name[0:3] == "eth" || name[0:3] == "ens" {
			ias.subnet2Local[getPrefix(ip)] = false
		} else {
			ias.subnet2Local[getPrefix(ip)] = true
		}
	}
}

func (ias *IpAddressSegment) IsLocal(ip string) bool {
	return ias.subnet2Local[getPrefix(ip)]
}

func getPrefix(ip string) string {
	index := strings.Index(ip, ":")
	if index == -1 {
		index := strings.LastIndex(ip, ".")
		return ip[0:index]
	}
	return ip
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
			result[addrString[0:index]] = netIf.Name
		}
	}
	return result, nil
}

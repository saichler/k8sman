package main

import (
	"fmt"
	"github.com/saichler/k8sman/go/transport"
)

func main() {
	ips, err := transport.LocalIps()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Local Ips:")
	for ip, _ := range ips {
		fmt.Println(ip)
	}
}

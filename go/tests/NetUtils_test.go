package tests

import (
	"fmt"
	"github.com/saichler/k8sman/go/transport"
	"testing"
)

func TestLocalIps(t *testing.T) {
	fmt.Println(transport.LocalIps())
}

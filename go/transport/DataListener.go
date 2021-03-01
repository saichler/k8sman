package transport

type DataListener interface {
	DataReceived([]byte, *Port)
}

package transport

import (
	utils "github.com/saichler/utils/golang"
	"sync"
)

type DataQueue struct {
	name    string
	queue   [][]byte
	mtx     *sync.Cond
	maxSize int
	active  bool
}

func NewDataQueue(name string, maxSize int) *DataQueue {
	dataQueue := &DataQueue{}
	dataQueue.mtx = sync.NewCond(&sync.Mutex{})
	dataQueue.queue = make([][]byte, 0)
	dataQueue.maxSize = maxSize
	dataQueue.active = true
	dataQueue.name = name
	return dataQueue
}

func (dataQueue *DataQueue) Add(packet []byte) {
	dataQueue.mtx.L.Lock()
	defer dataQueue.mtx.L.Unlock()

	for len(dataQueue.queue) >= dataQueue.maxSize && dataQueue.active {
		dataQueue.mtx.Broadcast()
		dataQueue.mtx.Wait()
	}
	if dataQueue.active {
		dataQueue.queue = append(dataQueue.queue, packet)
	} else {
		dataQueue.queue = dataQueue.queue[0:0]
	}
	dataQueue.mtx.Broadcast()
}

func (dataQueue *DataQueue) Next() []byte {
	for dataQueue.active {
		dataQueue.mtx.L.Lock()
		if len(dataQueue.queue) == 0 {
			dataQueue.mtx.Broadcast()
			dataQueue.mtx.Wait()
		}
		if len(dataQueue.queue) > 0 {
			data := dataQueue.queue[0]
			dataQueue.queue = dataQueue.queue[1:]
			dataQueue.mtx.Broadcast()
			dataQueue.mtx.L.Unlock()
			return data
		}
		dataQueue.mtx.L.Unlock()
	}
	utils.Info("Data Queue", dataQueue.name, " has stopped.")
	return nil
}

func (dataQueue *DataQueue) Active() bool {
	return dataQueue.active
}

func (dataQueue *DataQueue) Shutdown() {
	dataQueue.active = false
	dataQueue.mtx.Broadcast()
}

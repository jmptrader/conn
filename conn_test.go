package conn

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

const (
	AppPort           = ":8210"
	InitialConnReadTO = time.Second * 60
	MsgHeadLen        = 5
	SendChanSize      = 5
	LoremIpsum        = `Lorem ipsum dolor sit amet, eros invenire an duo, ei malis maiorum eos. 
	Praesent conclusionemque usu no, eros libris qualisque usu an. Commune democritum 
	vituperatoribus ex his, vel labitur consectetuer ut, est an enim debet. Mea labitur 
	feugiat consequuntur ex. Vel sint duis disputationi id, his civibus conceptam ei. 
	Natum malis complectitur ei usu, sea id mazim sententiae, purto wisi cu duo`
)

var test *testing.T
var delegate = &DelegateImpl{}
var message []byte
var twoMessages []byte
var emptyMsg []byte //message buf with just head

func init() {
	libytes := []byte(LoremIpsum)
	message = append(message, []byte{byte(1)}...)
	lenBytes := make([]byte, 4, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(libytes)))
	message = append(message, lenBytes...)
	message = append(message, libytes...)

	msgLen := len(message)
	twoMessages = make([]byte, 2*msgLen, 2*msgLen)
	for i, v := range message {
		twoMessages[i] = v
		twoMessages[msgLen+i] = v
	}

	emptyLenBytes := make([]byte, 4, 4)
	binary.BigEndian.PutUint32(emptyLenBytes, uint32(0))
	emptyMsg = append([]byte{byte(1)}, emptyLenBytes...)
}

func TestServer(t *testing.T) {
	t.Log("message Len:", len(message), "bodyLen:", len(LoremIpsum))
	test = t

	go startServer(t)

	time.Sleep(time.Second * 2)
	t.Log("done waiting")

	//create client
	conn, _ := net.Dial("tcp", AppPort)

	//write message in one write pass
	//to simulate normal networking status
	conn.Write(message)

	//write two messages in one write pass
	//to simulate good networking status
	conn.Write(twoMessages)

	//write two message in several passes,
	//to simulate slow networking
	seg1 := 3
	seg2 := 200
	seg3 := 500
	seg4 := 501

	conn.Write(twoMessages[0:seg1])
	time.Sleep(time.Second * 1)
	conn.Write(twoMessages[seg1:seg2])
	time.Sleep(time.Second * 1)
	conn.Write(twoMessages[seg2:seg3])
	time.Sleep(time.Second * 1)
	conn.Write(twoMessages[seg3:seg4])
	time.Sleep(time.Second * 1)
	conn.Write(twoMessages[seg4:])
	time.Sleep(time.Second * 1)

	//send empty message with just head set
	conn.Write(emptyMsg)
	time.Sleep(time.Second * 1)
}

type DelegateImpl struct {
	c *Conn
}

//let's satisfy conn delegate

func (this *DelegateImpl) OnMessage(head []byte, msgBuf []byte) {
	msgType := head[0]
	msgLen := binary.BigEndian.Uint32(head[1:5])
	test.Log(msgType, msgLen)
	test.Log(string(msgBuf))
}

func (this *DelegateImpl) CalMsgLen(head []byte) int {
	msgType := head[0]
	msgLen := binary.BigEndian.Uint32(head[1:5])
	test.Log(msgType, msgLen)

	return int(msgLen)
}

func (this *DelegateImpl) OnConnTimeout() {
	test.Log("timeout")
}

func (this *DelegateImpl) OnConnEnd() {
	test.Log("conn ended")
}

func (this *DelegateImpl) OnWriteErr(err error) {
	test.Log("write error")
}

//delegate interface methods end

func startServer(t *testing.T) {
	server := NewServer(AppPort,
		func(inConn net.Conn) bool {
			return true
		},
		func(conn *Conn) {
			delegate.c = conn
			conn.SetDelegate(delegate)
			toTime := time.Now().Add(InitialConnReadTO)
			conn.SetReadTimeout(toTime)
		},
		MsgHeadLen,
		SendChanSize)

	err := server.Serve()
	if err != nil {
		t.Fatal("%v", err)
	}
}

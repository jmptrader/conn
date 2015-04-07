package conn

import (
	"io"
	"net"
	"time"
)

type Conn struct {
	inConn   net.Conn
	headLen  int
	sendChan chan []byte
	delegate Delegate
	needStop bool
}

// Delegate interface of connection
type Delegate interface {

	//called when read operation timeout
	//and needStop is not set ture
	OnConnTimeout()

	//called when read operation reads
	//io.EOF err
	OnConnEnd()

	//called when done with reading a full message
	OnMessage(head []byte, msg []byte)

	//called when done with reading the head part
	//of a message, to calculate the body length of the
	//coming message. this method will be called right
	//before method OnMessage
	CalMsgLen(head []byte) int

	//called when there is a write error
	OnWriteErr(err error)
}

func (this *Conn) start() {
	go this.read()
	go this.write()
}

func (this *Conn) read() {
	defer this.inConn.Close()

	delegate := this.delegate
	headLen := this.headLen

	//make buffers to hold incoming data.
	var headBuf, msgBuf []byte
	readBuf := make([]byte, 1024)

	headDone, msgDone := false, false
	msgLen := 0

	//read the incoming connection into the buffer.
	for {
		readLen, err := this.inConn.Read(readBuf)
		if err != nil {
			if err == io.EOF {
				delegate.OnConnEnd()
			} else {
				opErr, ok := err.(*net.OpError)              //convert to concrete type
				if ok && opErr.Timeout() && !this.needStop { //read time out
					delegate.OnConnTimeout()
				}
			}

			if !this.needStop {
				this.Close()
			}
			return
		}

		nextReadPoint := 0
		for {
			toReadCnt := readLen - nextReadPoint
			if toReadCnt == 0 {
				break
			}
			if !headDone { //read head data
				leftHeadCnt := headLen - len(headBuf)
				if toReadCnt >= leftHeadCnt {
					toReadCnt = leftHeadCnt
					headDone = true
				}
				headBuf = append(headBuf, readBuf[nextReadPoint:nextReadPoint+toReadCnt]...)
				nextReadPoint += toReadCnt

				if headDone {
					msgLen = delegate.CalMsgLen(headBuf)
					goto ReadMsg
				}
				break
			}

		ReadMsg:
			toReadCnt = readLen - nextReadPoint
			leftMsgCnt := msgLen - len(msgBuf)
			if toReadCnt >= leftMsgCnt {
				toReadCnt = leftMsgCnt
				msgDone = true
			}
			//fmt.Println("toReadCnt:", toReadCnt, msgDone)
			msgBuf = append(msgBuf, readBuf[nextReadPoint:nextReadPoint+toReadCnt]...)
			nextReadPoint += toReadCnt

			if msgDone {
				delegate.OnMessage(headBuf, msgBuf)

				headDone, msgDone = false, false
				headBuf, msgBuf = headBuf[:0], msgBuf[:0]
			}
		} //read buf loop
	} //read io loop
}

func (this *Conn) write() {
	for {
		msgBuf := <-this.sendChan
		if this.needStop {
			return
		}
		_, err := this.inConn.Write(msgBuf)
		if this.needStop {
			return
		}
		if err != nil {
			if err != io.EOF && !this.needStop {
				this.delegate.OnWriteErr(err)
			}
		}
	}
}

func (this *Conn) Send(msgBuf []byte) bool {
	if len(this.sendChan) == cap(this.sendChan) {
		//if channel is full, then return false
		//indicating a failed send
		return false
	}

	this.sendChan <- msgBuf
	return true
}

func (this *Conn) Close() {
	if this.needStop {
		return
	}

	this.needStop = true

	close(this.sendChan)

	//timeout after 1 seconds
	toTime := time.Now().Add(time.Second * 1)
	this.inConn.SetDeadline(toTime)
}

func (this *Conn) SetDelegate(delegate Delegate) {
	this.delegate = delegate
}

func (this *Conn) SetReadTimeout(time time.Time) {
	this.inConn.SetReadDeadline(time)
}

func (this *Conn) SetWriteTimeout(time time.Time) {
	this.inConn.SetWriteDeadline(time)
}

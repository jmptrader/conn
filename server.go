package conn

import (
	"net"
)

type ShouldAcceptConn func(inConn net.Conn) bool
type ConnCreated func(conn *Conn)

type Server struct {
	port            string
	headLen         int
	sendChanSize    int
	accFunc         ShouldAcceptConn
	connCreatedFunc ConnCreated
}

func NewServer(port string, accFunc ShouldAcceptConn, connCreatedFun ConnCreated, //
	headLen int, sendChanSize int) *Server {
	return &Server{port: port, accFunc: accFunc, connCreatedFunc: connCreatedFun, //
		headLen: headLen, sendChanSize: sendChanSize}
}

func (this *Server) Serve() error {
	l, err := net.Listen("tcp", this.port)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		inConn, err := l.Accept()
		if err != nil {
			return err
		}

		accept := this.accFunc(inConn)
		if !accept {
			continue
		}

		//create new conn
		sendChan := make(chan []byte, this.sendChanSize)
		conn := &Conn{inConn: inConn, headLen: this.headLen, //
			sendChan: sendChan}

		//config new connection
		this.connCreatedFunc(conn)

		//start real work
		conn.start()
	}
}

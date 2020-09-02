package local

import (
	"io/ioutil"
	"net"
	"strconv"
	"testing"
	"time"

	"../../core"
	"../../log"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/proxy"

	_ "../../proxy/identical"
	_ "../../proxy/masker"
	_ "../../proxy/socks"
)

const (
	targetAdress = "127.0.0.1:7894"
)

var (
	request  = []byte("1 + 1 = ?")
	response = []byte("1 + 1 = 2")
)

func TestRunningLocally(t *testing.T) {
	log.SetCurLogLevel(log.ErrorLevel)

	// init client node and server node
	go startNode(t, "server_a_config.json")
	go startNode(t, "server_b_config.json")
	go startNode(t, "client_config.json")
	time.Sleep(10e9)

	// init target server
	go startServer(t)
	time.Sleep(2e9)

	// create a local socks5 proxy client
	// first, get client node's listening port
	config, err := core.LoadConfig("client_config.json")
	if err != nil {
		t.Fatalf("Err in loading config: %v.", err)
	}

	// then create the socks5 client
	socks5Client, err := proxy.SOCKS5("tcp", "127.0.0.1:"+strconv.Itoa(int(config.Port)), nil, proxy.Direct)
	if err != nil {
		t.Fatalf("Err in creating socks5 client: %v.", err)
	}

	// finally dial the target server
	// send the request and receive the response
	conn, err := socks5Client.Dial("tcp", targetAdress)
	_, err = conn.Write(request)
	if err != nil {
		t.Fatalf("Socks5 client: err in sending request: %v", err)
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}
	t.Logf("Socks5 client: sending resquest: %s", string(request))

	receivedResponse, err := ioutil.ReadAll(conn)
	if err != nil {
		t.Errorf("Socks5 client: err in reading response: %v", err)
	}
	if cmp.Equal(receivedResponse, response) == false {
		t.Errorf("Socks5 client: err in reading response, want %s but get %s", string(response), string(receivedResponse))
	}
	t.Logf("End...")
	conn.Close()
}

func startServer(t *testing.T) {
	// init server
	var ln net.Listener
	var conn net.Conn
	var err error
	var tryTimes int = 10
	for i := 0; i <= tryTimes; i++ {
		ln, err = net.Listen("tcp", targetAdress)
		if err != nil {
			continue
		}

		conn, err = ln.Accept()
		if err == nil {
			break
		}

		ln.Close()

		if i == tryTimes {
			t.Fatalf("Can not create the target server")
			return
		}
	}

	// read request
	buffer := make([]byte, 512)
	nBytes, err := conn.Read(buffer)
	if err != nil {
		t.Errorf("Target server: err in reading request: %v", err)
	}

	if cmp.Equal(buffer[:nBytes], request) == false {
		t.Fatalf("Target server: err in reading request, want %s but get %s", string(request), string(buffer[:nBytes]))
	}
	t.Logf("Target server: receiving request: %s", string(request))

	// send response
	_, err = conn.Write(response)
	if err != nil {
		t.Errorf("Target server: err in sending response: %v", err)
	}
	t.Logf("Target server: sending response: %s", string(response))

	conn.Close()
	ln.Close()
}

func startNode(t *testing.T, configFile string) {
	config, err := core.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Err in loading config: %v.", err)
	}

	node, err := core.NewNode(config)
	if err != nil {
		t.Fatalf("Err in creating a new node: %v.", err)
	}

	err = node.Start()
	if err != nil {
		t.Fatalf("Err in starting node: %v.", err)
	}
}

package Network

import (
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"fmt"
	log "github.com/corgi-kx/logcustom"
	"io/ioutil"
	"net"
	"strconv"
	"sync"
)

var lock sync.Mutex

// var replymessage []byte
var (
	//记录客户端收到的Reply正确与否
	isReplyValid bool
	//记录客户端收到的Reply的序号
	clientReplyCount map[string]int
)

// 处理客户端tcp连接
func HandleClientTcp(conn net.Conn) {
	//m := make(map[string]int)

	b, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}

	lock.Lock() // 加锁
	clientReplyCount[string(b)]++
	lock.Unlock() // 解锁

	test := clientReplyCount[strconv.Itoa(Constant.RequestID)]
	if test > 1 {
		isReplyValid = true
	}

	//if string(b) == strconv.Itoa(requestID) {
	//	lock.Lock() // 加锁
	//	m[string(b)]++
	//	lock.Unlock() // 解锁
	//} else {
	//	lock.Lock() // 加锁
	//	m[string(b)] = 1
	//	lock.Unlock() // 解锁
	//}
	//
	//log.Info("received b is: ", string(b))
	//if m[string(b)] > nodeCount/3*2 {
	//	end := time.Now().UnixNano()
	//	during := (end - startTime) / 1000000
	//	log.Info("时延:", during,"ms")
	//	Delay = append(Delay, during)
	//	var sum int64 = 0
	//	for _, v := range Delay {
	//		sum += v
	//	}
	//	Avg = sum / int64(len(Delay))
	//	log.Info("Avg:", Avg, "ms")
	//	fmt.Println("Reply valid, preparing next stage.")
	//	isReplyValid = true
	//	m[string(b)] = 0
	//} else{}
}

// 客户端使用的tcp监听
func ClientTcpListen() {
	listen, err := net.Listen("tcp", "1007")
	if err != nil {
		log.Panic(err)
	}
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		go HandleClientTcp(conn)
	}

}

// 节点使用的tcp监听
func (p *PBFT) TcpListen() {
	listen, err := net.Listen("tcp", p.node.addr)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Printf("节点开启监听，地址：%s\n", p.node.addr)
	fmt.Printf("Node :%s Start \n", p.node.addr)
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err := ioutil.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		go p.handleRequest(b) //所有节点都会开启TCP监听，所有命令都包含一个包头。
		//当有对应的信息通过TCP被节点监听到，该节点就会执行该信息包含的相应命令。
	}

}

// 使用tcp发送消息
func TCPDial(context []byte, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Info("connect error", err)
		return
	}

	_, err = conn.Write(context)
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}

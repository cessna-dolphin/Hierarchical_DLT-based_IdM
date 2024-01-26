package Network

import (
	"Hierarchical_IdM/BlockchainLayer/Blockchain"
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"Hierarchical_IdM/BlockchainLayer/Utils"
	"encoding/hex"
	"encoding/json"
	"fmt"
	log "github.com/corgi-kx/logcustom"
	"strconv"
	"sync"
	"time"
)

// 本地消息池（模拟持久化层），只有确认提交成功后才会存入此池
var localMessagePool []Utils.Message

type node struct {
	//节点ID
	nodeID string
	//节点监听地址
	addr string
	//RSA私钥
	rsaPrivKey []byte
	//RSA公钥
	rsaPubKey []byte
}

type PBFT struct {
	//节点信息
	node node

	//每笔请求自增序号
	MainSequenceID int
	SideSequenceID int

	//锁
	lock sync.Mutex

	//当前侧链区块中交易数量，用于确定动态密钥位于侧脸中的具体位置
	DynaKeyPos map[int]int
	//ID交易缓存，用于模拟ID交易的生成
	IDtxPool chan *Utils.Tx
	//动态密钥交易缓存，用于模拟动态密钥交易的生成
	DynaKeytxPool chan *Utils.Tx
	//主节点ID交易管道
	leaderIDTxSetChan chan *Utils.TxSet
	//主节点交易缓存池
	leaderIDTxPool []*Utils.Tx
	//主节点动态密钥交易管道
	leaderDynaKeyTxSetChan chan *Utils.TxSet
	//主节点动态密钥交易缓存池
	leaderDynaKeyTxPool []*Utils.Tx

	//临时消息池，消息摘要对应消息本体
	messagePool map[string]Utils.Request
	//主链摘要池，用于暂时获取摘要
	DigestPoolMain map[int64]string
	//侧链摘要池，用于暂时获取摘要
	DigestPoolSide map[int64]string
	//临时准备消息池，用于存放未被commit的准备消息
	preparePool map[string]Utils.Prepare
	//存放收到的prepare数量(至少需要收到并确认2f个)，根据摘要来对应
	prePareConfirmCount map[string]map[string]bool
	//存放收到的commit数量（至少需要收到并确认2f+1个），根据摘要来对应
	commitConfirmCount map[string]map[string]bool
	//该笔消息是否已进行Commit广播
	isCommitBroadcast map[string]bool
	//该笔消息是否已对客户端进行Reply
	isReply map[string]bool
	//存放收到的View-Change数量（至少需要收到2f个）
	ViewChangeConfirmCount map[string]map[string]bool
	//New-View消息是否已经广播
	isNewViewBroadcast map[string]bool
	//时延
	start int64
	//
	delay []int64
	//
	delayAvg int64
	//客户端时延统计
	ClientDelay int64

	pubkeys map[string][]byte
}

//var TxRx int
//var isPrimaryNode bool
//var TxInBlock int //测试用，暂存一个区块中交易的数量

// 以nodeID和节点监听地址构造PBFT
func NewIdM(nodeID, addr string) *PBFT {
	p := new(PBFT)
	p.node.nodeID = nodeID
	p.node.addr = addr
	p.node.rsaPrivKey = Utils.GetPrivKey(nodeID)
	p.node.rsaPubKey = Utils.GetPubKey(nodeID)
	p.pubkeys = make(map[string][]byte)

	p.MainSequenceID = 0
	p.SideSequenceID = 0

	p.DynaKeyPos = make(map[int]int)
	p.leaderIDTxSetChan = make(chan *Utils.TxSet, 500)
	p.IDtxPool = make(chan *Utils.Tx, 50)
	p.leaderDynaKeyTxSetChan = make(chan *Utils.TxSet, 500)
	p.DynaKeytxPool = make(chan *Utils.Tx, 10)

	p.messagePool = make(map[string]Utils.Request)
	p.DigestPoolMain = make(map[int64]string)
	p.DigestPoolSide = make(map[int64]string)
	p.preparePool = make(map[string]Utils.Prepare)

	p.prePareConfirmCount = make(map[string]map[string]bool)
	p.commitConfirmCount = make(map[string]map[string]bool)
	p.isCommitBroadcast = make(map[string]bool)
	p.isReply = make(map[string]bool)

	return p
}

func (p *PBFT) handleRequest(data []byte) {
	//切割消息，根据消息命令调用不同的功能
	cmd, content := Utils.SplitMessage(data)
	switch Constant.Command(cmd) {
	case Constant.CTxTrans:
		p.handleTxTrans(content)
	case Constant.CRequest:
		p.handleClientRequest(content)
	case Constant.CPrePrepare:
		p.handlePrePrepare(content)
	case Constant.CPrepare:
		p.handlePrepare(content)
	case Constant.CCommit:
		p.handleCommit(content)
		//case cViewChange:
		//	p.handleViewChange(content)
		//case cNewView:
		//	p.handleNewView(content)
	}
}

// 节点处理来自其他节点的交易信息，放入管道
// 收到后判断自己是不是主节点，只有主节点才能进一步操作
func (p *PBFT) handleTxTrans(content []byte) {
	if p.node.nodeID != "1008" {
		//非主节点，直接退出
		return
	}
	txSet := new(Utils.TxSet)
	err := json.Unmarshal(content, txSet)
	if err != nil {
		log.Panic(err)
	}
	p.lock.Lock()
	//处理收到的交易，根据收到交易的具体形式
	switch txSet.TxS[0].Data.Category {
	case "ID": //如果是ID交易，则存储到ID交易缓存中
		log.Info("Received ID transactions.")
		if len(p.leaderIDTxSetChan) < 100 {
			log.Info("ID transactions saved in primary node ID transaction set")
			p.leaderIDTxSetChan <- txSet
		}
	case "DynaKey": //如果是动态密钥交易，则存储到动态密钥交易缓存中
		log.Info("Received dynamic key transactions.")
		if len(p.leaderDynaKeyTxSetChan) < 100 {
			log.Info("Dynamic key transactions saved in primary node dynamic key transaction set.")
			p.leaderDynaKeyTxSetChan <- txSet
		}
	}
	p.lock.Unlock()
}

// 负责从管道中取出交易并发送
func (p *PBFT) sendTxTrans() {
	txsInBlock := 500 //规定每个区块中交易的数量，主侧链中这一设置相同
	defer p.lock.Unlock()
	for {
		if len(p.leaderIDTxPool) <= txsInBlock {
			//log.Info("Available spaces in primary node ID transaction pool, receiving new ID transactions from ID transaction set.")
			// 当ID交易池中，交易数量未达到阈值，则从ID交易管道接收新交易
			select {
			case txMainSet := <-p.leaderIDTxSetChan:
				//log.Info("Primary node ID transaction set valid.")
				for i, _ := range txMainSet.TxS {
					p.leaderIDTxPool = append(p.leaderIDTxPool, &txMainSet.TxS[i])
				}
				//log.Info("Primary node ID transaction pool received ID transactions of total: ", len(p.leaderIDTxPool))
			default:
				//log.Info("Primary node ID transaction set invalid. Continue.")
				//do nothing
			}
		} else {
			//否则，将ID交易池中交易打包形成新的ID区块，并准备开启主链上的共识
			p.lock.Lock()
			Utils.SortTxs(p.leaderIDTxPool)
			//交易打包发送
			bc := new(Blockchain.BlockChain)
			bc = Blockchain.MainBlockchainObject(Constant.ListenPort)
			block := Utils.NewBlock(Constant.CurMainHeight+1, bc.Tip, Utils.TxsPointer2Array(p.leaderIDTxPool[:txsInBlock]))
			//TxInBlock = len(p.leaderIDTxPool[:txsInBlock])
			r := new(Utils.Request)
			r.Timestamp = time.Now().UnixNano()
			r.ClientAddr = "1007"
			r.Message.ID = Utils.GetRandom()
			r.Message.Content = "New ID Block"
			r.Message.ABlock = *block
			br, err := json.Marshal(r)
			//清空缓存
			log.Info("New ID block generated.")
			p.leaderIDTxPool = p.leaderIDTxPool[txsInBlock:]
			log.Info("Erased packaged transactions in primary node ID transaction pool")

			p.lock.Unlock()
			if err != nil {
				log.Panic(err)
			}
			//发起交易
			Constant.WaitCommit.Wait()
			Constant.WaitCommit.Add(1) //waitGroup确保同一时间只有一个区块能够进行共识
			p.handleClientRequest(br)
			log.Info("Consensus for new ID block started.")
			log.Info("_____________________________________")
			log.Info("New ID block info:")
			log.Info("ID block height: ", Constant.CurMainHeight+1)
			log.Info("ID block tx nums: ", len(r.Message.ABlock.Data.Data))
			log.Info("_____________________________________")
		}
		if len(p.leaderDynaKeyTxPool) <= txsInBlock {
			//log.Info("Available spaces in primary node dynamic key transaction pool, receiving new dynamic key transactions from dynamic key transaction set.")
			// 当动态密钥交易池中，交易数量未达到阈值，则从动态密钥交易管道接收新交易
			select {
			case txSideSet := <-p.leaderDynaKeyTxSetChan:
				//log.Info("Primary node dynamic key transaction set valid")
				for i, _ := range txSideSet.TxS {
					p.leaderDynaKeyTxPool = append(p.leaderDynaKeyTxPool, &txSideSet.TxS[i])
				}
				//log.Info("Primary node dynamic key transaction pool received dynamic key transactions of total: ", len(p.leaderDynaKeyTxPool))
			default:
				//log.Info("Primary node dynamic key transaction set invalid. Continue.")
				//do nothing
			}
		} else {
			//否则，将动态密钥交易池中交易打包，并准备开启侧链上的共识
			p.lock.Lock()
			Utils.SortTxs(p.leaderDynaKeyTxPool)
			//交易打包发送
			bc := new(Blockchain.BlockChain)
			bc = Blockchain.SideBlockchainObject(Constant.ListenPort)
			block := Utils.NewBlock(Constant.CurSideHeight+1, bc.Tip, Utils.TxsPointer2Array(p.leaderDynaKeyTxPool[:txsInBlock]))
			//TxInBlock = len(p.leaderDynaKeyTxPool[:txsInBlock])
			r := new(Utils.Request)
			r.Timestamp = time.Now().UnixNano()
			r.ClientAddr = "1007"
			r.Message.ID = Utils.GetRandom()
			r.Message.Content = "New Dynamic Key Block"
			r.Message.ABlock = *block
			br, err := json.Marshal(r)
			//清空缓存
			log.Info("New dynamic key block generated.")
			p.leaderDynaKeyTxPool = p.leaderDynaKeyTxPool[txsInBlock:]
			log.Info("Erased packaged transactions in primary node dynamic key transaction pool.")
			p.lock.Unlock()
			if err != nil {
				log.Panic(err)
			}
			//发起交易
			Constant.WaitCommit.Wait()
			Constant.WaitCommit.Add(1) //Waitgroup机制确保同一时间只有一个区块进行共识
			p.handleClientRequest(br)
			log.Info("Consensus for new dynamic key block started.")
			log.Info("_____________________________________")
			log.Info("New Dynamic Key block info:")
			log.Info("Dynamic Key block height: ", Constant.CurSideHeight+1)
			log.Info("Dynamic Key block tx nums: ", len(r.Message.ABlock.Data.Data))
			log.Info("_____________________________________")
		}
	}
}

// 处理客户端发来的请求
// 通过PBFT共识算法共识区块
func (p *PBFT) handleClientRequest(content []byte) {
	//fmt.Println("主节点已接收到客户端发来的request ...")
	time.Sleep(3 * time.Second) //固定时延添加
	if p.node.nodeID == "1008" {
		//fmt.Println("has received the request from client ...")
		log.Info("has received the consensus request from client")
		//使用json解析出Request结构体
		r := new(Utils.Request)
		err := json.Unmarshal(content, r)
		if err != nil {
			log.Panic(err)
		}
		//根据收到的request描述，在主链或侧链上开启共识
		switch r.Message.Content {
		case "New ID Block": //ID区块，主链上共识
			log.Info("Received new ID block consensus request.")
			p.MainSequenceIDAdd()         //主链消息序号自增
			digest := Utils.GetDigest(*r) //获取消息摘要
			p.DigestPoolMain[Constant.CurMainHeight] = digest
			log.Info("Cached the ID block request message locally.")
			fmt.Println("Cached the ID block request message locally.")
			p.messagePool[digest] = *r //存入临时消息池
			//主节点对消息摘要进行签名
			digestByte, _ := hex.DecodeString(digest)
			signInfo := Utils.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			//拼接成PrePrepare，准备发往follower节点
			pp := Utils.PrePrepare{RequestMessage: *r, Digest: digest, SequenceID: p.MainSequenceID, Sign: signInfo}
			if pp.Digest != Utils.GetDigest(pp.RequestMessage) {
				log.Panic("Digest and message unmatched.")
			}
			b, err := json.Marshal(pp)
			if err != nil {
				log.Panic(err)
			}
			log.Info("PrePrepare for new ID block broadcast.")
			fmt.Println("PrePrepare broadcast ...")
			//生成时延
			//time.Sleep(time.Duration(getTransmitDelay(Constant.CommunicationMethod, p.parameters)) * time.Millisecond)
			//进行PrePrepare广播
			//p.broadcast(cPrePrepare, b)
			log.Info("PrePrepare broadcast started.")
			p.Broadcast(Constant.CPrePrepare, b)
			log.Info("PrePrepare broadcast completed.")
			fmt.Println("")
			fmt.Println("---------------------PrePrepare------------------------")
		case "New Dynamic Key Block":
			log.Info("Received new dynamic key block consensus request.")
			p.SideSequenceIDAdd()         //主链消息序号自增
			digest := Utils.GetDigest(*r) //获取消息摘要
			p.DigestPoolSide[Constant.CurSideHeight] = digest
			log.Info("Cached the dynamic block request message locally.")
			fmt.Println("Cached the dynamic block request message locally.")
			p.messagePool[digest] = *r //存入临时消息池
			//主节点对消息摘要进行签名
			digestByte, _ := hex.DecodeString(digest)
			signInfo := Utils.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			//拼接成PrePrepare，准备发往follower节点
			pp := Utils.PrePrepare{RequestMessage: *r, Digest: digest, SequenceID: p.SideSequenceID, Sign: signInfo}
			if pp.Digest != Utils.GetDigest(pp.RequestMessage) {
				log.Panic("Digest and message unmatched.")
			}
			b, err := json.Marshal(pp)
			if err != nil {
				log.Panic(err)
			}
			log.Info("PrePrepare for dynamic key block broadcast.")
			fmt.Println("PrePrepare broadcast ...")
			//生成时延
			//time.Sleep(time.Duration(getTransmitDelay(Constant.CommunicationMethod, p.parameters)) * time.Millisecond)
			//进行PrePrepare广播
			//p.broadcast(cPrePrepare, b)
			log.Info("PrePrepare broadcast started.")
			p.Broadcast(Constant.CPrePrepare, b)
			log.Info("PrePrepare broadcast completed.")
			fmt.Println("")
			fmt.Println("---------------------PrePrepare------------------------")
		}
	}
}

// 处理预准备消息
func (p *PBFT) handlePrePrepare(content []byte) {
	time.Sleep(3 * time.Second) //固定时延添加
	//fmt.Println("本节点已接收到主节点发来的PrePrepare ...")
	fmt.Println("---------------------PrePrepare------------------------")
	log.Info("---------------------PrePrepare------------------------")
	fmt.Println("has received the PrePrepare from the primary node")
	log.Info("Has received the PrePrepare from the primary node")
	//	//使用json解析出PrePrepare结构体
	pp := new(Utils.PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}
	//获取主节点的公钥，用于数字签名验证
	var primaryNodePubKey []byte
	if _, ok := p.pubkeys["1008"]; !ok {
		primaryNodePubKey = Utils.GetPubKey("1008")
		p.pubkeys["1008"] = Utils.GetPubKey("1008")
	} else {
		primaryNodePubKey = p.pubkeys["1008"]
	}

	digestByte, _ := hex.DecodeString(pp.Digest)
	if digest := Utils.GetDigest(pp.RequestMessage); digest != pp.Digest {
		fmt.Println("信息摘要对不上，拒绝进行prepare广播")
		log.Info("Unmatched digest, prepare broadcast refused.")
	} else if !Utils.RsaVerySignWithSha256(digestByte, pp.Sign, primaryNodePubKey) {
		fmt.Println("主节点签名验证失败！,拒绝进行prepare广播")
		log.Info("RSA sign validating failed, prepare broadcast refused.")
	} else {

		switch pp.RequestMessage.Message.Content {
		case "New ID Block": //若是ID区块，则在主链上继续共识流程
			log.Info("Received new ID block.")
			if p.MainSequenceID+1 != pp.SequenceID { //验证主链消息序列是否
				fmt.Println("Refuse to broadcast prepare, due to the unmatched main chain sequenceID and the digest sequence ID. ")
				log.Info("Refuse to broadcast prepare, due to the unmatched main chain sequenceID and the digest sequence ID.")
				log.Info(fmt.Sprintf("Current message sequenceID+1：%v  Received PrePrepare sequenceID：%v \n", p.MainSequenceID+1, pp.SequenceID))
				log.Info(fmt.Sprintf("Current message pool size：%v\n", len(p.messagePool)))
			}
			p.MainSequenceID = pp.SequenceID
			p.messagePool[pp.Digest] = pp.RequestMessage
			//节点使用私钥对其签名
			sign := Utils.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			//拼接成Prepare
			pre := Utils.Prepare{RequestMessage: pp.RequestMessage, Digest: pp.Digest, SequenceID: pp.SequenceID, NodeID: p.node.nodeID, Sign: sign}
			bPre, err := json.Marshal(pre)
			if err != nil {
				log.Panic(err)
			}

			//进行准备阶段的广播
			//fmt.Println("正在进行Prepare广播 ...")
			log.Info("Prepare for new ID block broadcast.")
			fmt.Println("Prepare broadcast...")
			//p.broadcast(cPrepare, bPre)
			log.Info("Prepare broadcast started.")
			p.Broadcast(Constant.CPrepare, bPre)

			log.Info("Prepare broadcast completed.")
			fmt.Println("---------------------Prepare------------------------")
		case "New Dynamic Key Block": //若是动态密钥区块，则在侧链上继续共识流程
			log.Info("Received new dynamic key block.")
			if p.SideSequenceID+1 != pp.SequenceID { //验证侧链消息序列是否
				fmt.Println("Refuse to broadcast prepare, due to the unmatched side chain sequenceID and the digest sequence ID. ")
				log.Info("Refuse to broadcast prepare, due to the unmatched side chain sequenceID and the digest sequence ID.")
				log.Info(fmt.Sprintf("Current message sequenceID+1：%v  Received PrePrepare sequenceID：%v \n", p.SideSequenceID+1, pp.SequenceID))
				log.Info(fmt.Sprintf("Current message pool size：%v\n", len(p.messagePool)))
			}
			p.SideSequenceID = pp.SequenceID
			p.messagePool[pp.Digest] = pp.RequestMessage
			//节点使用私钥对其签名
			sign := Utils.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			//拼接成Prepare
			pre := Utils.Prepare{RequestMessage: pp.RequestMessage, Digest: pp.Digest, SequenceID: pp.SequenceID, NodeID: p.node.nodeID, Sign: sign}
			bPre, err := json.Marshal(pre)
			if err != nil {
				log.Panic(err)
			}

			//进行准备阶段的广播
			//fmt.Println("正在进行Prepare广播 ...")
			log.Info("Prepare for new dynamic key block broadcast.")
			fmt.Println("Prepare broadcast...")
			//p.broadcast(cPrepare, bPre)
			log.Info("Prepare broadcast started.")
			p.Broadcast(Constant.CPrepare, bPre)

			log.Info("Prepare broadcast completed.")
			fmt.Println("---------------------Prepare------------------------")

		}
	}
}

// 处理准备消息
func (p *PBFT) handlePrepare(content []byte) {
	time.Sleep(3 * time.Second) //固定时延添加
	//使用json解析出Prepare结构体
	pre := new(Utils.Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Printf("本节点已接收到%s节点发来的Prepare ... \n", pre.NodeID)
	if p.node.nodeID == "1008" {
		fmt.Println("---------------------Prepare------------------------")
	}
	log.Info("Received prepare from node: ", pre.NodeID)
	fmt.Printf("has received the Prepare from the node %s  ... \n", pre.NodeID)

	p.lock.Lock()

	//获取消息源节点的公钥，用于数字签名验证
	var MessageNodePubKey []byte
	if _, ok := p.pubkeys[pre.NodeID]; !ok {
		MessageNodePubKey = Utils.GetPubKey(pre.NodeID)
		p.pubkeys[pre.NodeID] = Utils.GetPubKey(pre.NodeID)
	} else {
		MessageNodePubKey = p.pubkeys[pre.NodeID]
	}
	//MessageNodePubKey := getPubKey(pre.NodeID)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if _, ok := p.messagePool[pre.Digest]; !ok {
		fmt.Println("当前临时消息池无此摘要，拒绝执行commit广播")
		log.Info("No such digest in temp message pool, refuse commit broadcast.")
		//log.Info(fmt.Sprintf("当前消息序号：%v  收到Prepare消息序号：%v \n", p.MainSequenceID, pre.SequenceID))
	} else if !Utils.RsaVerySignWithSha256(digestByte, pre.Sign, MessageNodePubKey) {
		fmt.Println("节点签名验证失败！,拒绝执行commit广播")
		log.Info("RSA sign validating failed, refuse commit broadcast.")
	} else {
		switch pre.RequestMessage.Message.Content {
		case "New ID Block":
			log.Info("Received new ID block.")
			if p.MainSequenceID < pre.SequenceID {
				fmt.Println("Refuse to broadcast commit, due to the unmatched main chain sequence ID and the digest sequence ID")
				log.Info("Refuse to broadcast commit, due to the unmatched main chain sequence ID and the digest sequence ID")
			}
			p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
			count := 0
			for range p.prePareConfirmCount[pre.Digest] {
				count++
			}
			//因为主节点不会发送Prepare，所以不包含自己
			specifiedCount := 0
			if p.node.nodeID == "1008" {
				specifiedCount = (Constant.SPNum + Constant.UENum) / 3 * 2
			} else {
				specifiedCount = ((Constant.SPNum + Constant.UENum) / 3 * 2) - 1
			}
			//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
			//获取消息源节点的公钥，用于数字签名验证
			if count >= specifiedCount && !p.isCommitBroadcast[pre.Digest] {
				//fmt.Println("本节点已收到至少2f个节点(包括本地节点)发来的Interact信息 ...")
				fmt.Println("Prepare information from at least 2f nodes (including local nodes) has been received ...")
				//节点使用私钥对其签名
				sign := Utils.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
				c := Utils.Commit{RequestMessage: pre.RequestMessage, Digest: pre.Digest, SequenceID: pre.SequenceID, NodeID: p.node.nodeID, Sign: sign}
				bc, err := json.Marshal(c)
				if err != nil {
					log.Panic(err)
				}
				//进行提交信息的广播
				fmt.Println("commit broadcast")
				log.Info("Commit for new ID block broadcast.")
				//p.broadcast(cCommit, bc)
				p.Broadcast(Constant.CCommit, bc)

				log.Info("commit broadcast started.")
				p.isCommitBroadcast[pre.Digest] = true
				//fmt.Println("commit广播完成")
				log.Info("commit broadcast completed.")
				fmt.Println("---------------------Commit------------------------")
			}
		case "New Dynamic Key Block":
			log.Info("Received new dynamic key block.")
			if p.SideSequenceID < pre.SequenceID {
				fmt.Println("Refuse to broadcast commit, due to the unmatched side chain sequence ID and the digest sequence ID")
				log.Info("Refuse to broadcast commit, due to the unmatched side chain sequence ID and the digest sequence ID")
			}
			p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
			count := 0
			for range p.prePareConfirmCount[pre.Digest] {
				count++
			}
			//因为主节点不会发送Prepare，所以不包含自己
			specifiedCount := 0
			if p.node.nodeID == "1008" {
				specifiedCount = (Constant.SPNum + Constant.UENum) / 3 * 2
			} else {
				specifiedCount = ((Constant.SPNum + Constant.UENum) / 3 * 2) - 1
			}
			//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
			//获取消息源节点的公钥，用于数字签名验证
			if count >= specifiedCount && !p.isCommitBroadcast[pre.Digest] {
				//fmt.Println("本节点已收到至少2f个节点(包括本地节点)发来的Interact信息 ...")
				fmt.Println("Prepare information from at least 2f nodes (including local nodes) has been received ...")
				//节点使用私钥对其签名
				sign := Utils.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
				c := Utils.Commit{RequestMessage: pre.RequestMessage, Digest: pre.Digest, SequenceID: pre.SequenceID, NodeID: p.node.nodeID, Sign: sign}
				bc, err := json.Marshal(c)
				if err != nil {
					log.Panic(err)
				}
				//进行提交信息的广播
				fmt.Println("Commit for new dynamic key block broadcast")
				//p.broadcast(cCommit, bc)
				p.Broadcast(Constant.CCommit, bc)

				log.Info("commit broadcast started.")
				p.isCommitBroadcast[pre.Digest] = true
				//fmt.Println("commit广播完成")
				log.Info("commit broadcast completed.")
				fmt.Println("---------------------Commit------------------------")
			}
		}
		p.lock.Unlock()
	}
}

// 处理提交确认消息
func (p *PBFT) handleCommit(content []byte) {
	time.Sleep(3 * time.Second) //固定时延添加
	//TxRx = 0
	//使用json解析出Commit结构体
	c := new(Utils.Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Printf("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	fmt.Printf("has received the Commit from the node %s ... \n", c.NodeID)
	p.lock.Lock()
	//获取消息源节点的公钥，用于数字签名验证
	var MessageNodePubKey []byte
	if _, ok := p.pubkeys[c.NodeID]; !ok {
		MessageNodePubKey = Utils.GetPubKey(c.NodeID)
		p.pubkeys[c.NodeID] = Utils.GetPubKey(c.NodeID)
	} else {
		MessageNodePubKey = p.pubkeys[c.NodeID]
	}
	//MessageNodePubKey := getPubKey(c.NodeID)
	digestByte, _ := hex.DecodeString(c.Digest)

	if !Utils.RsaVerySignWithSha256(digestByte, c.Sign, MessageNodePubKey) {
		fmt.Println("节点签名验证失败！,拒绝将信息持久化到本地消息池")
		log.Info("RSA sign validating failed, refused to save the new block.")
	} else {
		switch c.RequestMessage.Message.Content {
		case "New ID Block":
			log.Info("Received new ID block in commit.")
			p.setCommitConfirmMap(c.Digest, c.NodeID, true)
			if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
				fmt.Println("当前prepare池无此摘要，拒绝将信息持久化到本地消息池")
				log.Info("No such digest in prepare pool, refused to save the new block.")
				log.Info(fmt.Sprintf("当前prepare池大小%v, 当前本地消息池大小%v", len(p.prePareConfirmCount), len(localMessagePool)))
			}
			count := 0
			for range p.commitConfirmCount[c.Digest] {
				count++
			}
			if p.node.nodeID == "1008" {
				log.Info(fmt.Sprintf("主节点收到节点%v的commit信息，共收到%v条commit信息。\n", c.NodeID, count))
			}
			//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！

			//if count >= nodeCount/3*2 && !p.isReply[c.Digest] && p.isCommitBroadcast[c.Digest] {
			if count >= (Constant.SPNum+Constant.UENum)/3*2 && !p.isReply[c.Digest] && p.isCommitBroadcast[c.Digest] {
				//fmt.Println("本节点已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息 ...")
				fmt.Println("Commit information from at least 2f nodes (including local nodes) has been received ...")

				//将消息信息，提交到本地消息池中！
				localMessagePool = append(localMessagePool, p.messagePool[c.Digest].Message)
				//info := p.node.nodeID + "节点已将msgid:" + strconv.Itoa(p.messagePool[c.Digest].ID) + "存入本地消息池中,消息内容为：" + p.messagePool[c.Digest].Content
				info := "Node " + p.node.nodeID + " has Stored the message which msgid = " + strconv.Itoa(p.messagePool[c.Digest].ID) + " in local and update state database"

				fmt.Println(info)

				// 将区块存储到本地区块链
				bc := new(Blockchain.BlockChain)
				bc = Blockchain.MainBlockchainObject(p.node.nodeID)
				block := p.messagePool[c.Digest].ABlock
				bc.UpdateMainBlock(&block, p.node.nodeID)
				fmt.Println("reply ...")
				log.Info("New ID block generated.")
				//先不reply

				p.isReply[c.Digest] = true
				if p.node.nodeID == "1008" {
					Constant.WaitCommit.Done()
					log.Info("Consensus for new ID block done: ", Constant.CurMainHeight)
				}
				//fmt.Println("reply完毕")
			}
			if Constant.CurMainHeight%10 == 0 && Constant.CurMainHeight != 10 {
				for i := Constant.CurMainHeight - 20; i < Constant.CurMainHeight-10; i++ {
					DigestOld, ok := p.DigestPoolMain[i]
					if ok {
						delete(p.messagePool, DigestOld)
						delete(p.prePareConfirmCount, DigestOld)
						delete(p.commitConfirmCount, DigestOld)
					} else {
						continue
					}
				}
			}
		case "New Dynamic Key Block":
			log.Info("Received new dynamic block in commit.")
			p.setCommitConfirmMap(c.Digest, c.NodeID, true)

			count := 0
			for range p.commitConfirmCount[c.Digest] {
				count++
			}
			if p.node.nodeID == "1008" {
				log.Info(fmt.Sprintf("主节点收到节点%v的commit信息，共收到%v条commit信息。\n", c.NodeID, count))
			}
			//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！

			//if count >= nodeCount/3*2 && !p.isReply[c.Digest] && p.isCommitBroadcast[c.Digest] {
			if count >= (Constant.SPNum+Constant.UENum)/3*2 && !p.isReply[c.Digest] && p.isCommitBroadcast[c.Digest] {
				//fmt.Println("本节点已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息 ...")
				fmt.Println("Commit information from at least 2f nodes (including local nodes) has been received ...")

				//将消息信息，提交到本地消息池中！
				localMessagePool = append(localMessagePool, p.messagePool[c.Digest].Message)
				//info := p.node.nodeID + "节点已将msgid:" + strconv.Itoa(p.messagePool[c.Digest].ID) + "存入本地消息池中,消息内容为：" + p.messagePool[c.Digest].Content
				info := "Node " + p.node.nodeID + " has Stored the message which msgid = " + strconv.Itoa(p.messagePool[c.Digest].ID) + " in local and update state database"

				fmt.Println(info)

				// 将区块存储到本地区块链
				bc := new(Blockchain.BlockChain)
				bc = Blockchain.SideBlockchainObject(p.node.nodeID)
				block := p.messagePool[c.Digest].ABlock
				bc.UpdateSideBlock(&block, p.node.nodeID)
				fmt.Println("reply ...")
				log.Info("New dynamic key block generated.")
				//先不reply

				p.isReply[c.Digest] = true
				if p.node.nodeID == "1008" {
					Constant.WaitCommit.Done()
					log.Info("Consensus for dynamic key block done.")
					log.Info("Consensus for new Dynamic Key block done: ", Constant.CurSideHeight)
				}
				//fmt.Println("reply完毕")
			}
			if Constant.CurSideHeight%10 == 0 && Constant.CurSideHeight != 10 {
				for i := Constant.CurSideHeight - 20; i < Constant.CurSideHeight-10; i++ {
					DigestOld, ok := p.DigestPoolSide[i]
					if ok {
						delete(p.messagePool, DigestOld)
						delete(p.prePareConfirmCount, DigestOld)
						delete(p.commitConfirmCount, DigestOld)
					} else {
						continue
					}
				}
			}

		}
	}
	p.lock.Unlock()
}

// 计算时延
func (p *PBFT) delayC() float64 {
	//结束计时
	end := time.Now().UnixNano()
	start := p.start
	during := float64(end-start) / 1000000000
	return during
}

// 主链序号累加
func (p *PBFT) MainSequenceIDAdd() {
	p.lock.Lock()
	p.MainSequenceID++
	p.lock.Unlock()
}

// 侧链序号累加
func (p *PBFT) SideSequenceIDAdd() {
	p.lock.Lock()
	p.SideSequenceID++
	p.lock.Unlock()
}

// 向除自己外的其他节点进行广播
func (p *PBFT) Broadcast(cmd Constant.Command, content []byte) {
	for i := range Constant.NodeTable {
		if i == p.node.nodeID {
			continue
		}
		message := Utils.JointMessage(cmd, content)
		go TCPDial(message, Constant.NodeTable[i])
	}
}

// 为多重映射开辟赋值
func (p *PBFT) setPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[string]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}

// 为多重映射开辟赋值
func (p *PBFT) setCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = make(map[string]bool)
	}
	p.commitConfirmCount[val][val2] = b
}

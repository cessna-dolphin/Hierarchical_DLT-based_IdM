package Network

import (
	"Hierarchical_IdM/ApplicationLayer/Sim"
	"Hierarchical_IdM/BlockchainLayer/Blockchain"
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"Hierarchical_IdM/BlockchainLayer/Utils"
	"fmt"
	"log"
	"os"
	"strconv"
)

func Server() {
	//生成日志目录
	if !Utils.IsExist(fmt.Sprintf("./%s", Constant.LogPath)) {
		err := os.Mkdir(Constant.LogPath, 0644)
		if err != nil {
			log.Panic()
		}

	}
	//生成数据目录
	if !Utils.IsExist(fmt.Sprintf("./%s", Constant.DataPath)) {
		err := os.Mkdir(Constant.DataPath, 0644)
		if err != nil {
			log.Panic()
		}
	}

	//入网节点生成公私钥
	nodeID := Utils.GenRsaKeys()
	log.Println("Generated node with ID: ", nodeID)
	Constant.ListenPort = nodeID

	//初始化节点池
	Constant.NodeTable = make(map[string]string, Constant.UENum+Constant.SPNum)
	ClientPortInt, _ := strconv.Atoi(Constant.ClientPort)
	for i := 0; i < Constant.SPNum+Constant.UENum; i++ {
		Constant.NodeTable[strconv.Itoa(i+ClientPortInt)] = fmt.Sprintf("127.0.0.1:%s", strconv.Itoa(i+ClientPortInt))
	}
	//指定主节点（Authorized） "1008"
	//TODO 目前Constant.NodeTable仅支持在生成区块链节点之前完成map的映射，这也是为何系统不支持运行中添加节点的原因。后续需要考虑全局变量NodeTable，每生成一个新节点都需要将旧节点的nodeTable更新
	//TODO 投机取巧 一次设置超多的节点丢进map里也可行，虽然占用资源
	if nodeID == Constant.ClientPort { //客户端，负责生成创世块
		Utils.InitLog(nodeID)
		world := new(Utils.World)
		world.InitNode(nodeID, Constant.UENum+Constant.SPNum)
		log.Println("World initiated with initial UE number and SP number: ", Constant.UENum, Constant.SPNum)
		Blockchain.CreateBlockChainWithGenesisBlock()
		log.Println("Both blockchain initializing with genesis block completed.")
		Utils.CopyGenesisBlock(nodeID)

		//todo test
		//输入连个整数，空格隔开。首个整数表示区块高度，第二个整数表示区块中交易序号
		for {
			var h, tx int
			fmt.Scanln(&h, &tx)
			mappings := Sim.ViewMapping(h, tx)
			fmt.Println("Picked main height:" + strconv.Itoa(h) + " and tx order:" + strconv.Itoa(tx) + " for test.")
			for k, v := range mappings {
				fmt.Println("mapping side height:" + strconv.Itoa(k) + " on side tx on number: " + strconv.Itoa(v))
			}
		}

	} else {
		//生成区块链节点
		p := NewIdM(nodeID, fmt.Sprintf("127.0.0.1:%s", nodeID))
		log.Println("Blockchain node generated.")
		Utils.InitLog(nodeID)
		//复制创世块
		Utils.CopyGenesisBlock(nodeID)
		if nodeID == "1008" {
			p.DynaKeyPos[1] = 0
			go p.sendTxTrans() //由于目前只有主节点（1008）能够打包区块，因此直接用txTransfer即可实现
			go p.txListen(0)
			go p.txTransfer()
		}
		p.TcpListen()
	}
}

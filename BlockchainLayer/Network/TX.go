package Network

import (
	"Hierarchical_IdM/BlockchainLayer/Blockchain"
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"Hierarchical_IdM/BlockchainLayer/Utils"
	"encoding/json"
	log "github.com/corgi-kx/logcustom"
	"strings"
	"time"
)

//本文档将模拟每个节点的交易到达以及交易打包

//// 模拟主链ID数据的生成
//func (p *PBFT) NewMainChainIDTX() {
//
//}

// 模拟主侧链上的交易到达
func (p *PBFT) txListen(ChangeIndex int64) {
	for {
		//每隔0.1s生成10个ID数据
		//lambda := Utils.GetPoisson(float64(100))
		for i := 0; i < 10; i++ {
			p.IDtxPool <- Utils.NewRandomIDTX()
		}
		if Constant.CurMainHeight%2 == 0 { //每生成2块ID块，生成10个动态公钥
			for i := 0; i < 10; i++ {
				p.DynaKeytxPool <- Blockchain.NewDynaKeyTX()
			}
		}
		TxRx = len(p.IDtxPool)
		//TransStart := NodeState{sTrans, time.Now().UnixNano(),TxRx, isPrimaryNode}
		//p.StatePool <- TransStart
		time.Sleep(100 * time.Millisecond) //控制交易生成时间

		////每5个ID块，生成1个动态密钥块
		//if Constant.CurMainHeight%5 == 0 {
		//	//lambda := Utils.GetPoisson(float64(100))
		//	for i := 0; i < int(lambda); i++ {
		//		p.DynaKeytxPool <- Utils.NewRandomDynaKeyTX()
		//	}
		//}
	}
}

// 将交易池中的交易打包，发送给主节点
func (p *PBFT) txTransfer() {
	// 每多少毫秒发送一次（限制发送次数）
	ticker := time.NewTicker(10 * time.Second)
	// 每次发送的交易量
	maxTx := 500

	//主侧链缓存池
	var txsMain, txsSide []Utils.Tx
	//主侧链交易是否做好准备发送
	mainReady, sideReady := make(chan int), make(chan int)
	for {
		<-ticker.C //每次定时器抵达，才开始以下步骤
		log.Info("Transaction timer reached. Start packaging transactions and send to primary node.")
		//开启协程，循环读取IDtxPool以及DKtxPool中的全部交易，直到某一种交易的缓存池达到发送阈值
		go func() {
			for IDtx, IDok := <-p.IDtxPool; IDok; {
				//log.Info("ID transaction pool valid.")
				txsMain = append(txsMain, *IDtx)
				//log.Info("Current main chain transaction pool with ID transactions: ", len(txsMain))
				if len(txsMain) >= maxTx {
					log.Info("Main chain transaction reached threshold: ", maxTx)

					mainReady <- 1 //主链交易已经准备好发送，向对应的通道发送信号
					log.Info("Main chain transaction pool is now ready for sending.")
				}
			}
		}()
		go func() {
			for DKtx, DKok := <-p.DynaKeytxPool; DKok; {
				if strings.Compare(DKtx.Data.Content, "Dynamic key existed") != 0 {
					txsSide = append(txsSide, *DKtx)
				}
				//log.Info("Dynamic key pool valid.")
				//log.Info("Current side chain transaction pool with dynamic key transactions: ", len(txsSide))
				if len(txsSide) >= maxTx {
					log.Info("Side chain transaction reached threshold: ", maxTx)
					sideReady <- 1 //侧链交易已经准备好发送，向对应的通道发送信号
					log.Info("Side chain transaction pool is now ready for sending.")
				}
			}
		}()
		//开启协程，监听主侧链上交易的准备情况
		go func() {
			select {
			case <-mainReady: //主链准备好，则发送，并抹去已发送的数据
				log.Info("Main chain ready.")
				p.sendTxs2Leader(txsMain)
				log.Info("ID transactions packed and sent to main chain.")
				if len(txsMain) != 0 {
					txsMain = txsMain[0:0] //清空切片
					log.Info("Main chain transaction pool cleared.")
				}

			case <-sideReady:
				log.Info("Side chain ready.")
				p.sendTxs2Leader(txsSide)
				log.Info("Dynamic key transactions packed and sent to side chain.")

				if len(txsSide) != 0 {
					txsSide = txsSide[0:0]
					log.Info("Side chain transaction pool cleared.")
				}

			default:
				log.Info("Neither chain is ready to send transactions.")
				//do nothing
			}
		}()
	}

}

// 向主节点发送交易信息
func (p *PBFT) sendTxs2Leader(txs []Utils.Tx) {
	txSet := Utils.TxSet{TxS: txs}
	br, err := json.Marshal(txSet)
	if err != nil {
		log.Panic(err)
	}
	if len(txs) != 0 { //确保交易存在，才发送
		content := Utils.JointMessage(Constant.CTxTrans, br)
		TCPDial(content, Constant.NodeTable["1008"]) //向主节点发送request
		p.handleTxTrans(br)
		log.Info("Transactions successfully sent.")
	}

}

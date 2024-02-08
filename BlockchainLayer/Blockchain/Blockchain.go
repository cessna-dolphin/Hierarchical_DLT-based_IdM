package Blockchain

import (
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"Hierarchical_IdM/BlockchainLayer/Utils"
	"fmt"
	"github.com/boltdb/bolt"
	log "github.com/corgi-kx/logcustom"
	"golang.org/x/exp/rand"
	"strconv"
	"time"
)

type BlockChain struct {
	DB  *BlockchainDB
	Tip []byte //保存最新区块哈希值
}

func newBlockchain() *BlockChain {
	return &BlockChain{}
}

//func CreateBlockChainWithGenesisBlock(nodeId string) {
//	block := CreateGenesisBlock([]Tx{*NewCoinBaseTransaction(nodeId)})
//	blockByte := block.Serialize()
//	blockchain := newBlockchain()
//	blockchain.DB.Put(block.Header.Hash, blockByte, blockTableName)
//	blockchain.DB.Put([]byte("1"), block.Header.Hash, blockTableName)
//	blockchain.Tip = block.Header.Hash
//}

// CreateBlockChainWithGenesisBlock 初始化区块链，包括主链和侧链
// address是钱包的地址
func CreateBlockChainWithGenesisBlock() {
	//保存最新区块的哈希值
	//var blockHash []byte
	//dbName := fmt.Sprintf(Constant.DbName, nodeId)

	//分别创建主链和侧链两个数据库
	dbMain, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, Constant.GenesisMainName), 0600, nil)
	defer dbMain.Close()
	if err != nil {
		log.Panicf("create db[%s] failed %v\n", Constant.DbMainName, err)
	}
	dbSide, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, Constant.GenesisSideName), 0600, nil)
	defer dbSide.Close()
	if err != nil {
		log.Panicf("create db[%s] failed %v\n", Constant.DbSideName, err)
	}

	//2.创建桶,把生成的创世区块存入主侧链各自的数据库中
	dbMain.Update(func(tx *bolt.Tx) error {
		bM := tx.Bucket([]byte(Constant.MainBlockTableName))
		if bM == nil {
			//没找到桶
			bM, err := tx.CreateBucket([]byte(Constant.MainBlockTableName))
			if err != nil {
				log.Panicf("create bucket[%s] failed %v\n", Constant.MainBlockTableName, err)
			}
			//生成主链的创世区块，块里是包括单条交易，ID编号为0000
			genesisMainBlock := Utils.CreateGenesisBlock(Utils.GenesisMainInit())
			log.Info("Created main chain genesis block")
			//存储
			//1.key,value分别以什么数据代表
			//2.如何把block结构存入数据库中---序列化
			err = bM.Put(genesisMainBlock.Header.Hash, genesisMainBlock.Serialize()) //key是hash，value是序列化的结果-----无论是key还是value都是[]byte
			if err != nil {
				log.Panicf("insert the genesis block failed %v\n", err)
			}
			log.Info("Main chain genesis block inserted.")
			//复制创世块文件
			//BLC.CreateGenesisDB(genesisBlock.Header.Hash, genesisBlock.Serialize())
			//blockHash = genesisBlock.Hash
			//存储最新区块的哈希
			err = bM.Put([]byte("1"), genesisMainBlock.Header.Hash)
			if err != nil {
				log.Panicf("save the hash of genesis block failed %v\n", err)
			}
			log.Info("Main chain genesis block hash saved.")
			//更新高度
			Constant.NewMainHeight = 1
			Constant.CurMainHeight = 1
			log.Info("Main chain current height is at: ", Constant.CurMainHeight)
		}
		//log.Info("创世块创建成功")
		return nil
	})
	dbSide.Update(func(tx *bolt.Tx) error {
		bS := tx.Bucket([]byte(Constant.SideBlockTableName))
		if bS == nil {
			//没找到桶
			bS, err := tx.CreateBucket([]byte(Constant.SideBlockTableName))
			if err != nil {
				log.Panicf("create bucket[%s] failed %v\n", Constant.SideBlockTableName, err)
			}
			//生成侧链的创世区块，块里是包括单条交易，动态密钥为"InitDynamicKey"
			genesisSideBlock := Utils.CreateGenesisBlock(Utils.GenesisSideInit())
			log.Info("Created side chain genesis block")
			//存储
			//1.key,value分别以什么数据代表
			//2.如何把block结构存入数据库中---序列化
			err = bS.Put(genesisSideBlock.Header.Hash, genesisSideBlock.Serialize()) //key是hash，value是序列化的结果-----无论是key还是value都是[]byte
			if err != nil {
				log.Panicf("insert the genesis block failed %v\n", err)
			}
			log.Info("Side chain genesis block inserted.")
			//复制创世块文件
			//BLC.CreateGenesisDB(genesisBlock.Header.Hash, genesisBlock.Serialize())
			//blockHash = genesisBlock.Hash
			//存储最新区块的哈希
			err = bS.Put([]byte("1"), genesisSideBlock.Header.Hash)
			if err != nil {
				log.Panicf("save the hash of genesis block failed %v\n", err)
			}
			log.Info("Side chain genesis block hash saved.")
			//更新高度
			Constant.NewSideHeight = 1
			Constant.CurSideHeight = 1
			log.Info("Side chain current height is at: ", Constant.CurSideHeight)
		}
		//log.Info("创世块创建成功")
		return nil
	})

	return
}

// 根据交易生成对应的新区块
func (blockChain *BlockChain) NewBlockFromTx(txs []Utils.Tx, nodeID string) {
	switch txs[0].Data.Category {
	//根据交易种类决定更新主链或侧链
	case "ID":
		hash := blockChain.DB.View([]byte("1"), Constant.MainBlockTableName, 0, "1008")
		blockBytes := blockChain.DB.View(hash, Constant.MainBlockTableName, 0, "1008")
		preBlock := Utils.DeserializeBlock(blockBytes)
		block := Utils.NewBlock(preBlock.Header.Height+1, preBlock.Header.Hash, txs)
		blockChain.UpdateMainBlock(block, nodeID)
	case "DynaKey":
		hash := blockChain.DB.View([]byte("1"), Constant.SideBlockTableName, 1, "1008")
		blockBytes := blockChain.DB.View(hash, Constant.SideBlockTableName, 1, "1008")
		preBlock := Utils.DeserializeBlock(blockBytes)
		block := Utils.NewBlock(preBlock.Header.Height+1, preBlock.Header.Hash, txs)
		blockChain.UpdateSideBlock(block, nodeID)
	default:
		panic("No such transaction category.")
	}
}

// UpdateMainBlock 更新新区块到主链数据库
func (blockChain *BlockChain) UpdateMainBlock(block *Utils.Block, nodeID string) {
	//更新区块
	blockChain.DB.Put(block.Header.Hash, block.Serialize(), Constant.MainBlockTableName, 0, nodeID)
	//更新tip
	blockChain.DB.Put([]byte("1"), block.Header.Hash, Constant.MainBlockTableName, 0, nodeID)
	blockChain.Tip = block.Header.Hash
	Constant.CurMainHeight = block.Header.Height
}

// UpdateSideBlock 更新新区块到侧链数据库
func (blockChain *BlockChain) UpdateSideBlock(block *Utils.Block, nodeID string) {
	//更新区块
	blockChain.DB.Put(block.Header.Hash, block.Serialize(), Constant.SideBlockTableName, 1, nodeID)
	//更新tip
	blockChain.DB.Put([]byte("1"), block.Header.Hash, Constant.SideBlockTableName, 1, nodeID)
	blockChain.Tip = block.Header.Hash
	Constant.CurSideHeight = block.Header.Height
}

// MainBlockchainObject 从主链上获取一个blockchain对象
func MainBlockchainObject(nodeId string) *BlockChain {
	//获取DB
	var DBFileName = "NodeMain_" + nodeId + ".db"
	db, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, DBFileName), 0600, &bolt.Options{Timeout: time.Millisecond * 500})
	if err != nil {
		log.Panic("open the db [%s] failed! %v\n", Constant.DbMainName, err)
	}
	defer db.Close()
	//获取Tip Tip即最新区块哈希值
	var tip []byte
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Constant.MainBlockTableName))
		if b != nil {
			tip = b.Get([]byte("1"))
		} else {
			log.Panic("blockchain is nil! %v\n", err)
		}
		return nil
	})
	if err != nil {
		log.Panic("get the blockchain object failed! %v\n", err)
	}
	realResult := make([]byte, len(tip))
	copy(realResult, tip)
	return &BlockChain{NewDB(), realResult}
}

// SideBlockchainObject 从侧链上获取一个blockchain对象
func SideBlockchainObject(nodeId string) *BlockChain {
	//获取DB
	var DBFileName = "NodeSide_" + nodeId + ".db"
	db, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, DBFileName), 0600, &bolt.Options{Timeout: time.Millisecond * 500})
	if err != nil {
		log.Panic("open the db [%s] failed! %v\n", Constant.DbSideName, err)
	}
	defer db.Close()
	//获取Tip
	var tip []byte
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Constant.SideBlockTableName))
		if b != nil {
			tip = b.Get([]byte("1"))
		} else {
			log.Panic("blockchain is nil! %v\n", err)
		}
		return nil
	})
	if err != nil {
		log.Panic("get the blockchain object failed! %v\n", err)
	}
	realResult := make([]byte, len(tip))
	copy(realResult, tip)
	return &BlockChain{NewDB(), realResult}
}

// GetHeightMain 获取当前主链高度
func (blockChain *BlockChain) GetHeightMain() int64 {
	return blockChain.Iterator().MainNext().Header.Height
}

// GetHeightSide 获取当前侧链高度
func (blockChain *BlockChain) GetHeightSide() int64 {
	return blockChain.Iterator().SideNext().Header.Height
}

// GetGenesisBlockMain 获取主链创世块
func (blockChain *BlockChain) GetGenesisBlockMain() *Utils.Block {
	bcit := blockChain.Iterator()
	var block *Utils.Block
	for {
		block = bcit.MainNext()
		if block.Header.Height == 1 {
			break
		}
	}
	return block
}

// GetGenesisBlockSide 获取侧链创世块
func (blockChain *BlockChain) GetGenesisBlockSide() *Utils.Block {
	bcit := blockChain.Iterator()
	var block *Utils.Block
	for {
		block = bcit.SideNext()
		if block.Header.Height == 1 {
			break
		}
	}
	return block
}

// 根据高度获取区块
func GetBlockByHeight(nodeID string, height int) *Utils.Block {
	MainBlockFromDB := MainBlockchainObject(nodeID) //从主链上获取区块链对象
	ThisMainBlockByte := MainBlockFromDB.DB.View(MainBlockFromDB.Tip, Constant.MainBlockTableName, 0, "1008")
	ThisMainBlock := Utils.DeserializeBlock(ThisMainBlockByte)
	for int(ThisMainBlock.Header.Height) != height {
		//若当前区块高度与指定高度不符，遍历前一区块
		ThisMainBlock = Utils.DeserializeBlock(MainBlockFromDB.DB.View(ThisMainBlock.Header.PreviousHash, Constant.MainBlockTableName, 0, "1008"))
	}
	return ThisMainBlock
}

// 根据已有的主链交易，随机选取一对ID并基于其生成动态密钥
func NewDynaKeyFromExistID(pos map[int]int) string {
	if Constant.CurMainHeight <= 1 { //仍在创世块前
		return "Dynamic key existed"
	}

	b1, b2 := rand.Intn(int(Constant.CurMainHeight-1))+2, rand.Intn(int(Constant.CurMainHeight-1))+2 //随机选取的两个主链区块高度，+2原因是不能获取到创世块
	log.Info("Get random main chain heights:", b1, b2)
	var mBlock1, mBlock2 *Utils.Block
	//在主链上获取两个随机区块
	mBlock1, mBlock2 = GetBlockByHeight("1008", b1), GetBlockByHeight("1008", b2)
	//获取两个随机交易编号
	num1, num2 := rand.Intn(len(mBlock1.Data.Data)), rand.Intn(len(mBlock2.Data.Data))
	log.Info("Get random ID transaction number:", num1, num2)
	m1Data, m2Data := mBlock1.Data.Data, mBlock2.Data.Data
	//TODO 将动态密钥的格式定义为：两个区块的高度+两个交易的编号+两个随机ID的前8位
	//随机选择的两项都已经有映射，将不再生成新的动态密钥
	if m1Data[num1].Mapping != nil && m2Data[num2].Mapping != nil {
		return "Dynamic key existed"
	}
	DynamicKey := "Height1_" + strconv.Itoa(b1) + "Height_2" + strconv.Itoa(b2) + "tx1_" + strconv.Itoa(num1) + "tx2" + strconv.Itoa(num2) + m1Data[num1].Data.Content[:8] + m2Data[num2].Data.Content[:8]
	log.Info("The dynamic key is: ", DynamicKey)

	//更改所选择的两个ID交易的映射关系
	//TODO 如何确定当前的Dynamic Key将会作为第几侧链区块的第几交易提交？

	//m1Data[num1].Mapping, m2Data[num2].Mapping = pos, pos                                                    //将规定的交易位置标志位存入两个选定的ID的映射位
	MainBlockFromDB := MainBlockchainObject("1008") //从主链上获取区块链对象
	mBlock1.Data.Data[num1].Mapping, mBlock2.Data.Data[num2].Mapping = pos, pos
	MainBlockFromDB.DB.Put(mBlock1.Header.Hash, mBlock1.Serialize(), Constant.MainBlockTableName, 0, "1008") //将更新过的交易重新持久化到原始区块中
	MainBlockFromDB.DB.Put(mBlock2.Header.Hash, mBlock2.Serialize(), Constant.MainBlockTableName, 0, "1008") //将更新过的交易重新持久化到原始区块中
	return DynamicKey
}

// 生成随机动态密钥
func NewDynaKeyTX(pos map[int]int) *Utils.Tx {
	data := Utils.Data{
		Category: "DynaKey",
		Content:  NewDynaKeyFromExistID(pos),
	}
	fixedPos := pos[int(Constant.CurSideHeight+1)]
	dynaMapping := make(map[int]int)
	dynaMapping[1] = fixedPos
	tx := Utils.Tx{
		Hash:      nil,
		TimeStamp: time.Now().Unix(),
		Data:      data,
		Mapping:   dynaMapping, //将事先规定的交易标志位作为严格限定动态密钥交易的排序参数
	}
	tx.HashTransaction()
	return &tx
}

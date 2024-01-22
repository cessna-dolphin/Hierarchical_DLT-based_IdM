package Blockchain

import (
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"Hierarchical_IdM/BlockchainLayer/Utils"
)

//区块链迭代器管理文件

// BlockChainIterator 实现迭代器基本结构
type BlockChainIterator struct {
	//DB  *bolt.DB //迭代目标
	DB          *BlockchainDB
	CurrentHash []byte //当前迭代目标的哈希
}

//next()

// Iterator 创建迭代器对象
func (blockChain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{blockChain.DB, blockChain.Tip}
}

// MainNext 实现迭代函数next,获取到每一个区块，主链用
func (bcit *BlockChainIterator) MainNext() *Utils.Block {
	var block *Utils.Block
	currentBlockBytes := bcit.DB.View(bcit.CurrentHash, Constant.MainBlockTableName, 0)
	block = Utils.DeserializeBlock(currentBlockBytes)
	bcit.CurrentHash = block.Header.PreviousHash
	return block
}

// SideNext 实现迭代函数next,获取到每一个区块，侧链用
func (bcit *BlockChainIterator) SideNext() *Utils.Block {
	var block *Utils.Block
	currentBlockBytes := bcit.DB.View(bcit.CurrentHash, Constant.SideBlockTableName, 1)
	block = Utils.DeserializeBlock(currentBlockBytes)
	bcit.CurrentHash = block.Header.PreviousHash
	return block
}

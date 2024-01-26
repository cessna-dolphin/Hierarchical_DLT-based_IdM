package Sim

import "Hierarchical_IdM/BlockchainLayer/Blockchain"

// 查看主侧映射关系
func ViewMapping(mainBlockHeight int, mainTxOrder int) map[int]int {
	targetBlock := Blockchain.GetBlockByHeight("1008", mainBlockHeight)
	targetTx := targetBlock.Data.Data[mainTxOrder]
	return targetTx.Mapping
}

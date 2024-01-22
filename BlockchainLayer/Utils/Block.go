package Utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math"
	"math/big"
	"time"
)

type Block struct {
	Header BlockHeader
	Data   BlockData
}

type BlockHeader struct {
	TimeStamp    int64
	Height       int64
	PreviousHash []byte
	Hash         []byte
}

type BlockData struct {
	DataHash []byte
	Data     []Tx
}

type BlockMetadata struct {
	Metadata []byte
}

// Serialize 区块结构序列化
func (block *Block) Serialize() []byte {

	var buffer bytes.Buffer
	//新建编码对象
	encoder := gob.NewEncoder(&buffer)
	//编码（序列化）
	if err := encoder.Encode(block); err != nil {
		log.Panicf("serialized the block to []byte failed %v\n", err)
	}
	return buffer.Bytes()
}

// DeserializeBlock 区块数据反序列化
func DeserializeBlock(blockBytes []byte) *Block {
	var block Block
	//新建decoder对象
	decoder := gob.NewDecoder(bytes.NewReader(blockBytes))
	if err := decoder.Decode(&block); err != nil {
		log.Panicf("deserialized []byte to block failed %v\n", err)
	}
	return &block
}

func NewBlock(height int64, prevBlockHash []byte, txs []Tx) *Block {
	header := BlockHeader{
		Height:       height,
		PreviousHash: prevBlockHash,
		Hash:         nil,
		TimeStamp:    time.Now().Unix(),
	}
	block := Block{
		Header: header,
		Data:   BlockData{nil, txs},
	}

	bigInt, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		log.Panic("随机数错误:", err)
	}

	nonce := bigInt.Int64()
	dataBytes := block.GenerateHash(nonce)
	hash := sha256.Sum256(dataBytes)
	block.Header.Hash = hash[:]
	block.Data.DataHash = block.HashTransaction()
	return &block
}

// HashTransaction 把指定区块中所有交易结构都序列化(类Merkle的哈希计算方法)
func (block *Block) HashTransaction() []byte {
	var txHashes [][]byte
	//将指定区块中所有交易哈希进行拼接
	for _, tx := range block.Data.Data {
		txHashes = append(txHashes, tx.Hash)
	}
	txHash := sha256.Sum256(bytes.Join(txHashes, []byte{}))
	return txHash[:]
}

func (block *Block) GenerateHash(nonce int64) []byte {
	timeStampBytes := IntToHex(block.Header.TimeStamp)
	heightBytes := IntToHex(block.Header.Height)
	data := bytes.Join([][]byte{
		heightBytes,
		timeStampBytes,
		block.Header.PreviousHash,
		block.Data.DataHash,
		IntToHex(nonce),
	}, []byte{})
	return data
}

// CreateGenesisBlock 生成创世区块
func CreateGenesisBlock(txs []Tx) *Block {
	block := NewBlock(1, nil, txs)
	return block
}

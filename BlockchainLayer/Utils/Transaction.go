package Utils

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"log"
	rand2 "math/rand"
	"sort"
	"strconv"
	"time"
)

// 交易的具体数据格式
type Data struct {
	Category string //交易类型 ID/DynaKey
	Content  string //具体交易格式 ID和动态公钥均用字符串表示
}
type Tx struct {
	Hash      []byte
	TimeStamp int64
	Data      Data        //交易具体形式
	Mapping   map[int]int //完成两个交易之间的映射
}

type TxSet struct {
	TxS []Tx
}

func NewTx() *Tx {
	tx := Tx{
		Hash: nil,
	}
	tx.HashTransaction()
	return &tx
}

// 为主链的创世块生成一个初始交易
func GenesisMainTx() *Tx {
	tx := Tx{
		Hash:      nil,
		TimeStamp: time.Now().Unix(),
		Data:      Data{Category: "ID", Content: "0000"},
	}
	tx.HashTransaction()
	return &tx
}

// 为侧链的创世块生成一个初始交易
func GenesisSideTx() *Tx {
	tx := Tx{
		Hash:      nil,
		TimeStamp: time.Now().Unix(),
		Data:      Data{Category: "DynaKey", Content: "InitDynamicKey"},
	}
	tx.HashTransaction()
	return &tx
}

// 生成随机ID
func NewRandomIDTX() *Tx {
	data := Data{
		Category: "ID",
		Content:  strconv.FormatInt(rand2.Int63(), 10),
	}
	tx := Tx{
		Hash:      nil,
		TimeStamp: time.Now().Unix(),
		Data:      data,
		//不对map进行初始化
	}
	tx.HashTransaction()
	return &tx
}

// 生成随机动态密钥
func NewRandomDynaKeyTX() *Tx {
	data := Data{
		Category: "DynaKey",
		Content:  RandomSeq(64), //TODO 随机64位字符串作为动态密钥
	}
	tx := Tx{
		Hash:      nil,
		TimeStamp: time.Now().Unix(),
		Data:      data,
	}
	tx.HashTransaction()
	return &tx
}

type TokenAction struct {
}

// HashTransaction 生成交易哈希（交易序列化）
func (tx *Tx) HashTransaction() {
	var result bytes.Buffer
	//交易序列化
	encoder := gob.NewEncoder(&result)
	if err := encoder.Encode(tx); err != nil {
		log.Panicf("tx Hash encoded failed %v\n", err)
	}

	//生成哈希值
	hash := sha256.Sum256(result.Bytes())
	tx.Hash = hash[:]
}

// 使用私钥进行数字签名
func EllipticCurveSign(privateKey *ecdsa.PrivateKey, hash []byte) []byte {
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash)
	if err != nil {
		log.Panic("EllipticCurveSign:", err)
	}
	signature := append(r.Bytes(), s.Bytes()...)
	return signature
}

// NewBaseTransaction 实现base交易
func NewBaseTransaction(address string) *Tx {
	tx := NewTx()
	return tx
}

// NewCoinBaseTransaction 实现base交易
func NewCoinBaseTransaction(address string) *Tx {
	tx := NewTx()
	return tx
}

func NewTransaction(address string) *Tx {
	tx := NewTx()
	return tx
}

// 主链创世块的初始化
func GenesisMainInit() []Tx {
	Txs := make([]Tx, 0)
	tx := GenesisMainTx()
	Txs = append(Txs, *tx)
	return Txs
}

// 侧链创世块的初始化
func GenesisSideInit() []Tx {
	Txs := make([]Tx, 0)
	tx := GenesisSideTx()
	Txs = append(Txs, *tx)
	return Txs
}

// 交易的排序，需要针对ID交易和动态密钥交易的特征进行
func SortTxs(txs []*Tx) {
	if len(txs) == 0 { //空交易切片，直接返回
		return
	}
	//ID交易的特点：时间戳有序，ID编号也可进行排序
	if txs[0].Data.Category == "ID" {
		sort.Slice(txs, func(i, j int) bool {
			if txs[i].TimeStamp == txs[j].TimeStamp {
				return txs[i].Data.Content < txs[j].Data.Content
			} else {
				return txs[i].TimeStamp < txs[j].TimeStamp
			}
		})
	}
	//动态密钥交易特点：时间戳有序，动态密钥本身也可进行排序，但需要在交易打包进区块前就知道该密钥将会作为第几个
	//由于动态密钥不需要继续向下映射，故考虑在生成动态密钥时，对Mapping赋值，此时按照Mapping的Value值进行排序即可
	//交易排序发生在curSideHeight增长之前，因此考虑对curSideHeight + 1的key值排序
	if txs[0].Data.Category == "DynaKey" {
		sort.Slice(txs, func(i, j int) bool {
			return txs[i].Mapping[1] < txs[j].Mapping[1]
		})
	}
}

// 将交易指针数组转换为普通数组
func TxsPointer2Array(txsP []*Tx) []Tx {
	txsA := make([]Tx, 0)
	for _, value := range txsP {
		txsA = append(txsA, *value)
	}
	return txsA
}

package Utils

import (
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"bytes"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	log2 "github.com/corgi-kx/logcustom"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
	"log"
	"math/big"
	"os"
	"strconv"
)

//
//数制/格式转换
//

// 命令转换为请求
func CommandToBytes(command string) []byte {
	var bytes [Constant.CommandLen]byte
	for i, c := range command {
		bytes[i] = byte(c)
	}
	return bytes[:]
}

// 请求转换为命令
func BytesToCommand(bytes []byte) string {
	var command []byte
	for _, v := range bytes {
		if v != 0x00 {
			command = append(command, v)
		}
	}
	return fmt.Sprintf("%s", command)
}

// IntToHex 实现int64转[]byte
func IntToHex(data int64) []byte {
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.BigEndian, data)
	if err != nil {
		log.Panicf("int transact to []byte failed! %v\n", err)
	}
	return buffer.Bytes()
}

// JSONToSlice 标准json格式转切片
func JSONToSlice(jsonString string) []string {
	var strSlice []string
	//json
	if err := json.Unmarshal([]byte(jsonString), &strSlice); err != nil {
		log.Panicf("json to []string failed! %v\n", err)
	}
	return strSlice
}

// StringToHash160 string转hash160
func StringToHash160(address string) []byte {
	pubKeyHash := Base58Decode([]byte(address))
	hash160 := pubKeyHash[:len(pubKeyHash)-Constant.AddressCheckSumLen]
	return hash160
}

// byte数组转字符串
func BytesToHexString(b []byte) string {
	var buf bytes.Buffer
	for _, v := range b {
		t := strconv.FormatInt(int64(v), 16)
		if len(t) > 1 {
			buf.WriteString(t)
		} else {
			buf.WriteString("0" + t)
		}
	}
	return buf.String()
}

//
//判断是否存在
//

// 判断文件或文件夹是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println(err)
		return false
	}
	return true
}

//// Check 检查数据库是否存在
//func Check(nodeId string) bool {
//	if DbExit(nodeId) {
//		fmt.Println("该节点已存在文件已存在...")
//		return true
//	}
//	return false
//}
//
//// DbExit 判断数据库文件是否存在
//func DbExit(nodeId string) bool {
//	//生成不同节点数据库文件
//	dbMainName := fmt.Sprintf(Constant.DbMainName, nodeId)
//	dbSideName := fmt.Sprintf(Constant.DbSideName, nodeId)
//	if _, err := os.Stat(dbMainName); os.IsNotExist(err) {
//		//数据库文件不存在
//		return false
//	}
//	return true
//}

// 日志文件初始化
func InitLog(nodeId string) {
	file, err := os.OpenFile(fmt.Sprintf("%slog%s.txt", "./log/", nodeId), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	//fmt.Println(file)
	if err != nil {
		log2.Error(err)
	}
	log2.SetOutputAll(file)
}

//
//编码/摘要等
//

// gob 编码
func GobEncode(data interface{}) []byte {
	var res bytes.Buffer
	enc := gob.NewEncoder(&res)
	err := enc.Encode(data)
	if nil != err {
		log.Panicf("encode the data failed! %v\n", err)
	}
	return res.Bytes()
}

// 对消息详情进行摘要
func GetDigest(request Request) string {
	b, err := json.Marshal(request)
	if err != nil {
		log.Panic(err)
	}
	hash := sha256.Sum256(b)
	//进行十六进制字符串编码
	return hex.EncodeToString(hash[:])
}

// 默认前十二位为命令名称
func JointMessage(cmd Constant.Command, content []byte) []byte {
	b := make([]byte, Constant.PrefixCMDLength)
	for i, v := range []byte(cmd) {
		b[i] = v
	}
	joint := make([]byte, 0)
	joint = append(b, content...)
	return joint
}

// 默认前十二位为命令名称
func SplitMessage(message []byte) (cmd string, content []byte) {
	cmdBytes := message[:Constant.PrefixCMDLength]
	newCMDBytes := make([]byte, 0)
	for _, v := range cmdBytes {
		if v != byte(0) {
			newCMDBytes = append(newCMDBytes, v)
		}
	}
	cmd = string(newCMDBytes)
	content = message[Constant.PrefixCMDLength:]
	return
}

// GetEnvNodeId 获取节点ID
func GetEnvNodeId() string {
	nodeId := os.Getenv("NODE_ID")
	if nodeId == "" {
		fmt.Println("NODE_ID is not set")
		os.Exit(1)
	}
	return nodeId
}

//
//区块相关
//

// CopyGenesisBlock 复制创世块
func CopyGenesisBlock(to string) {
	dbMainName := fmt.Sprintf(Constant.DbMainName, to)
	dbSideName := fmt.Sprintf(Constant.DbSideName, to)
	//db0, err := bolt.Open(Constant.GenesisName, 0600, nil)
	dbM, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, Constant.GenesisMainName), 0600, nil)
	defer dbM.Close()
	if err != nil {
		log.Panicf("create db[%s] failed %v\n", Constant.GenesisMainName, err)
	}
	var key, data []byte
	dbM.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Constant.MainBlockTableName))
		if b != nil {
			key = b.Get([]byte("1"))
			data = b.Get(key)
		} else {
			log.Panicf("the main genesis block is nil %v\n", err)
		}
		return nil
	})

	dbM1, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, dbMainName), 0600, nil)
	defer dbM1.Close()
	if err != nil {
		log.Panicf("create db[%s] failed %v\n", dbMainName, err)
	}
	dbM1.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Constant.MainBlockTableName))
		if b == nil {
			b, err := tx.CreateBucket([]byte(Constant.MainBlockTableName))
			if err != nil {
				log.Panicf("create bucket[%s] failed %v\n", Constant.MainBlockTableName, err)
			}
			err = b.Put(key, data) //key是hash，value是序列化的结果-----无论是key还是value都是[]byte
			if err != nil {
				log.Panicf("insert the genesis block failed %v\n", err)
			}
			err = b.Put([]byte("1"), key)
			if err != nil {
				log.Panicf("save the hash of genesis block failed %v\n", err)
			}
		}
		return nil
	})
	Constant.CurMainHeight = 1
	log.Println("Main chain genesis block copied to: ", to)

	dbS, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, Constant.GenesisSideName), 0600, nil)
	defer dbS.Close()
	if err != nil {
		log.Panicf("create db[%s] failed %v\n", Constant.GenesisSideName, err)
	}
	dbS.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Constant.SideBlockTableName))
		if b != nil {
			key = b.Get([]byte("1"))
			data = b.Get(key)
		} else {
			log.Panicf("the side genesis block is nil %v\n", err)
		}
		return nil
	})

	dbS1, err := bolt.Open(fmt.Sprintf("%s%s", Constant.DataPath, dbSideName), 0600, nil)
	defer dbS1.Close()
	if err != nil {
		log.Panicf("create db[%s] failed %v\n", dbSideName, err)
	}
	dbS1.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Constant.SideBlockTableName))
		if b == nil {
			b, err := tx.CreateBucket([]byte(Constant.SideBlockTableName))
			if err != nil {
				log.Panicf("create bucket[%s] failed %v\n", Constant.SideBlockTableName, err)
			}
			err = b.Put(key, data) //key是hash，value是序列化的结果-----无论是key还是value都是[]byte
			if err != nil {
				log.Panicf("insert the genesis block failed %v\n", err)
			}
			err = b.Put([]byte("1"), key)
			if err != nil {
				log.Panicf("save the hash of genesis block failed %v\n", err)
			}
		}
		return nil
	})
	Constant.CurSideHeight = 1
	log.Println("Side chain genesis block copied to: ", to)
	return
}

// 获取公钥Hash
func GetPublicKeyHashFromAddress(address string) []byte {
	addressBytes := []byte(address)
	fullHash := Base58Decode(addressBytes)
	publicKeyHash := fullHash[1 : len(fullHash)-Constant.CheckSum]
	return publicKeyHash
}

//
//数学运算
//

// Abs 绝对值
func Abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// 获取泊松分布序列
func GetPoisson(Lambda float64) float64 {
	src := rand.New(rand.NewSource(uint64(GetRandom())))
	p := distuv.Poisson{Lambda: Lambda, Src: src}
	result := p.Rand()
	return result
}

// 获取随机的十位数，作为source
func GetRandom() int {
	x := big.NewInt(10000000000)
	for {
		result, err := cryptorand.Int(cryptorand.Reader, x)
		if err != nil {
			log.Panic(err)
		}
		if result.Int64() > 1000000000 {
			return int(result.Int64())
		}
	}
}

// 获取高斯分布序列
func GetNormal(Mu float64, Sigma float64) float64 {
	src := rand.New(rand.NewSource(uint64(GetRandom())))
	n := distuv.Normal{Mu, Sigma, src}
	result := n.Rand()
	return result
}

const elements = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// 生成一个随机字符串
func RandomSeq(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = elements[rand.Intn(len(elements))]
	}
	return string(b)
}

package Utils

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	log "github.com/corgi-kx/logcustom"
	uuid "github.com/satori/go.uuid"
	"github.com/thedevsaddam/gojsonq/v2"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const FileName = "./Data/WorldStatus.json"
const FileName2 = "./Data/World.json"

type World struct {
	txSuccess  int
	txFail     int
	txSuccess2 int
	txFail2    int
	data       []Record
	dataD      []Record
}

type TmpData struct {
	key []byte
	r   []Record
}

type Record struct {
	Key      string
	Version  uint
	MetaData MetaData
}

type MetaData struct {
	Devices   []Device
	Resources map[string]int
	Account   Account
}

type Status struct {
	Data    []Record
	ModTime time.Time
}

type Device struct {
	Id     string
	Energy int
	Status string
}

type Account struct {
	Balance int
	Address []byte
}

type Resource struct {
	Status  string
	Amount  int
	ModTime time.Time
}

// 初始化
func (world *World) WorldTable(data Status, name string) {
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()
	res, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		log.Panic(err)
	}
	file.Write(res)
}

// 展示全部status
func (world *World) ShowTheWorld() {
	v := make([]Record, 0)
	gojsonq.New().File(FileName).From("MetaData").Out(&v)
	fmt.Printf("%#v\n", v)
}

// 根据key查找
func (world *World) FindTheWorld(key string) bool {
	v := make([]Record, 0)
	gojsonq.New().File(FileName).From("MetaData").Out(&v)
	fmt.Printf("%#v\n", v)
	for _, value := range v {
		if strings.Compare(value.Key, key) == 0 {
			fmt.Println(value)
		}
	}
	return false
}

//
//func (world *World) SetTheWorld(data tmpData) {
//	world.key = append(world.key, data.key)
//	world.DB.Put(data.key, data.r[0].Serialize(), worldTableName)
//}
//
//func (world *World) UpdateTheWorld(key []byte, data string) {
//	dataByte := world.DB.View(key, worldTableName)
//	s := DeserializeStatus(dataByte)
//	s.Version++
//	world.DB.Put(key, s.Serialize(), worldTableName)
//}

// Serialize 序列化
func (r *Record) Serialize() []byte {
	var buffer bytes.Buffer
	//新建编码对象
	encoder := gob.NewEncoder(&buffer)
	//编码（序列化）
	if err := encoder.Encode(r); err != nil {
		log.Panicf("serialized the record to []byte failed %v\n", err)
	}
	return buffer.Bytes()
}

// DeserializeStatus 反序列化
func DeserializeStatus(Bytes []byte) *Record {
	var r Record
	//新建decoder对象
	decoder := gob.NewDecoder(bytes.NewReader(Bytes))
	if err := decoder.Decode(&r); err != nil {
		log.Panicf("deserialized []byte to block failed %v\n", err)
	}
	return &r
}

// GenDevices 生成设备
func GenDevices() []Device {
	n := rand.Intn(10) + 90
	var devices []Device
	for i := 0; i < n; i++ {
		devices = append(devices, Device{
			Id:     GenUId(),
			Energy: 1,
			Status: "running",
		})
	}
	return devices
}

// GenAccount 初始化账户
func GenAccount(nodeId string) Account {
	return Account{
		Balance: 100,
		Address: GetPubKey(nodeId),
	}
}

// GenResource 初始化资源
func GenResource(devices []Device) (res map[string]int) {
	sum := 0
	for _, v := range devices {
		sum += v.Energy
	}
	var list = []string{"idle", "sell", "tradingS", "tradingB"}
	var amountList = []int{50, sum - 50, 0, 0}
	//var res map[string]int
	res = make(map[string]int, 4)
	for k, v := range list {
		res[v] = amountList[k]
	}
	//for k, v := range list {
	//	res = append(res, Resource{
	//		Status:  v,
	//		Amount:  amountList[k],
	//		ModTime: time.Time{},
	//	})
	//}
	return res
}

// 生成uid
func GenUId() string {
	uid := uuid.NewV4()
	return uid.String()
}

// InitNode 节点初始化
func (world *World) InitNode(nodeId string, num int) (s Status) {
	id, err := strconv.Atoi(nodeId)
	if err != nil {
		log.Info("error")
	}
	var records []Record
	for i := 0; i < num; i++ {
		devices := GenDevices()
		resource := GenResource(devices)
		account := GenAccount(nodeId)
		metaData := MetaData{
			Devices:   devices,
			Resources: resource,
			Account:   account,
		}
		r := Record{
			Key:      strconv.Itoa(id + i),
			Version:  0,
			MetaData: metaData,
		}
		records = append(records, r)
	}
	s = Status{
		Data:    records,
		ModTime: time.Now(),
	}
	world.WorldTable(s, FileName)
	world.WorldTable(s, FileName2)
	world.data = records
	world.dataD = records
	return
}

func (world *World) GetNodesStatus() {

}

func (world *World) UpdateNodesStatus() {

}

type TxTest struct {
	Rin  Txx
	Rout Txx
}

type Txx struct {
	Id     string
	amount int
}

// GenNewTxFromF1 从更新的数据中生成交易
func (world *World) GenNewTxFromF1(num int) (tx TxTest) {
	var rin, rout Txx
	var cnt = 0
l1:
	//随机一个buyer
	buyer := rand.Intn(num)
	//seller := rand.Intn(5)
	s := rand.NormFloat64()*10 + 15
	//r := world.data[seller].MetaData.Resources
	//fmt.Println(r)
	seller := rand.Intn(num)
	//找一个资源充足的seller
	if seller == buyer || s < 0 {
		goto l1
	}
	res := world.data[seller].MetaData.Resources
	resS := res["sell"]
	if s <= float64(resS) && s > 0 || cnt > 50 {
		rin.Id = world.data[seller].Key
		rin.amount = int(s)
		rout.Id = world.data[buyer].Key
		rout.amount = int(s)
	}
	cnt++
	if rout.amount == 0 {
		goto l1
	}
	tx.Rin = rin
	tx.Rout = rout
	fmt.Println(tx)
	return
}

// GenNewTxFromF2 从未更新的数据中生成交易
func (world *World) GenNewTxFromF2() {

}

// UpdateRes 更新资源状态
func (world *World) UpdateRes() {
	for k, v := range world.data {
		m := v.MetaData.Resources["idle"]
		if m > 50 {
			world.data[k].MetaData.Resources["sell"] += m - 50
			world.data[k].MetaData.Resources["idle"] = 50
		}
	}
}

func (world *World) UpdateByP(txs []TxTest) []TxTest {
	var ttt []TxTest
	for _, tx := range txs {
		var flag = 0
		var bur = 0
		for k, v := range world.data {
			//更新卖家
			if v.Key == tx.Rin.Id {
				if tx.Rin.amount <= v.MetaData.Resources["sell"] {
					world.data[k].MetaData.Resources["sell"] -= tx.Rin.amount
					world.data[k].MetaData.Resources["tradingS"] += tx.Rin.amount
				} else {
					flag = 1
					break
				}
			}
			//更新买家
			if v.Key == tx.Rout.Id {
				bur = k
			}
		}
		if flag == 0 {
			world.data[bur].MetaData.Resources["tradingB"] += tx.Rout.amount
			ttt = append(ttt, tx)
		} else {
			world.txFail++
		}

	}
	return ttt
}
func (world *World) UpdateStateP(txs []TxTest) {
	for _, tx := range txs {
		seller, buyer := -1, -1
		//更新卖家
		for k, v := range world.data {
			if v.Key == tx.Rin.Id {
				if v.MetaData.Resources["tradingS"] < tx.Rin.amount {
					break
				}
				seller = k
				break
			}
		}

		//更新买家
		for k2, v := range world.data {
			if v.Key == tx.Rout.Id {
				if v.MetaData.Resources["tradingB"] < tx.Rout.amount {
					break
				}
				buyer = k2
				break
			}
		}
		if seller < 0 || buyer < 0 {
			world.txFail++
			continue
		}
		world.data[seller].MetaData.Resources["tradingS"] -= tx.Rin.amount
		world.data[buyer].MetaData.Resources["tradingB"] -= tx.Rout.amount
		world.data[buyer].MetaData.Resources["idle"] += tx.Rout.amount
		world.txSuccess++
	}
}

// UpdateState 更新数据库
func (world *World) UpdateState(txs []TxTest) {
	for _, tx := range txs {
		var flag = 1
		//更新卖家
		for k, v := range world.data {
			if v.Key == tx.Rin.Id {
				if v.MetaData.Resources["sell"] < tx.Rin.amount {
					flag = 0
					break
				}
				world.data[k].MetaData.Resources["sell"] -= tx.Rin.amount
			}
		}
		if flag == 0 {
			world.txFail++
			continue
		}
		world.txSuccess++
		//更新买家
		for k2, v := range world.data {
			if v.Key == tx.Rout.Id {
				world.data[k2].MetaData.Resources["idle"] += tx.Rout.amount
			}
		}
	}
}

// VerifyTx 验证交易有效性
func (world *World) VerifyTx() bool {

	return false
}

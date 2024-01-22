package Constant

import (
	"math"
	"sync"
)

// 主链数据库名称
const DbMainName = "NodeMain_%s.db"

// 侧链数据库名称
const DbSideName = "NodeSide_%s.db"

// 主链区块链表名称
const MainBlockTableName = "MainBlocks"

// 侧链区块链表名称
const SideBlockTableName = "SideBlocks"

// 数据路径
const LogPath = "log"

// 随机数不能超过的最大值
const MaxInt = math.MaxInt64

// 校验和长度
const AddressCheckSumLen = 4

// 数据路径
const DataPath = "./Data/"

// 账户表名称
const WalletTableName = "Wallets"

// GenesisName 复制的创世块
const GenesisName = "GenesisBlock.db"

// MainChainGenesisName 主链创世块名称
const GenesisMainName = "MainGenesis.db"

// SideChainGenesisName 侧链创世块名称
const GenesisSideName = "SideGenesis.db"

// CommandLen  命令长度
const CommandLen = 12

// PROTOCOL 协议
const PROTOCOL = "tcp"

// 两次sha256(公钥hash)后截取的字节数量
const CheckSum = 4

// 命令参数
const (
	CTxTrans        Command = "txTrans"
	CRequest        Command = "request"
	CPrePrepare     Command = "preprepare"
	CPrepare        Command = "prepare"
	CCommit         Command = "commit"
	PrefixCMDLength         = 12
)

// 网络参数
var (
	ListenHost = "127.0.0.1"
	ListenPort = "1007"
	ClientPort = "1007"
	LeaderPort = "1008"
)

// DLT-Based IdM相关参数
var (
	UENum         int   //UE节点数量
	SPNum         int   //SP节点数量
	CurSideHeight int64 //当前侧链高度
	NewSideHeight int64 //最新侧链区块高度
	CurMainHeight int64 //当前主链高度
	NewMainHeight int64 //最新主链区块高度
)

// 节点池，主要用来存储监听地址
var NodeTable map[string]string

// 监听同步机制，等待一次commit完成再进入下一轮
var WaitCommit sync.WaitGroup

// Request序号
var RequestID = 0

type Command string
type Reply string

package Blockchain

import (
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	log "github.com/corgi-kx/logcustom"
	"os"
	"time"
)

type BlockchainDB struct {
	ListenPort string
}

// 数据路径
var dataPath = fmt.Sprintf("./%s", Constant.DataPath)

func NewDB() *BlockchainDB {
	bd := &BlockchainDB{Constant.ListenPort}
	return bd
}

// Put 根据主链或侧链 存入数据
func (DB *BlockchainDB) Put(k, v []byte, bt string, chainType int) {
	//lock.Lock()
	////log.Info("上锁")
	//defer lock.Unlock()
	var DBFileName string
	//0为主链，1为侧链
	switch chainType {
	case 0:
		DBFileName = "NodeMain_" + Constant.ListenPort + ".db"
	case 1:
		DBFileName = "NodeSide_" + Constant.ListenPort + ".db"

	}
	db, err := bolt.Open(fmt.Sprintf("%s%s", dataPath, DBFileName), 0600, &bolt.Options{Timeout: time.Millisecond * 500})
	defer func(db *bolt.DB) {
		err := db.Close()
		if err != nil {
			log.Info(err)
		}
	}(db)
	if err != nil {
		log.Warn(err)
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bt))
		if bucket == nil {
			var err error
			bucket, err = tx.CreateBucket([]byte(bt))
			if err != nil {
				log.Panic(err)
			}
		}
		err := bucket.Put(k, v)
		if err != nil {
			log.Panic(err)
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	//log.Info("解锁")
}

// View 查看数据
func (DB *BlockchainDB) View(k []byte, bt string, chainType int) []byte {
	//lock.Lock()
	////log.Info("上锁")
	//defer lock.Unlock()
	var DBFileName string
	//0为主链，1为侧链
	switch chainType {
	case 0:
		DBFileName = "NodeMain_" + Constant.ListenPort + ".db"
	case 1:
		DBFileName = "NodeSide_" + Constant.ListenPort + ".db"

	}
	db, err := bolt.Open(fmt.Sprintf("%s%s", dataPath, DBFileName), 0600, &bolt.Options{Timeout: time.Millisecond * 500})

	if err != nil {
		log.Warn(err)
	}
	defer func(db *bolt.DB) {
		err := db.Close()
		if err != nil {
			log.Warn(err)
		}
	}(db)
	var result []byte
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bt))
		if bucket == nil {
			msg := "database view warning:没有找到仓库：" + bt
			return errors.New(msg)
		}
		result = bucket.Get(k)
		return nil
	})
	if err != nil {
		//log.Warn(err)
		return nil
	}
	realResult := make([]byte, len(result))
	copy(realResult, result)
	//log.Info("解锁")
	return realResult
}

// Delete 删除数据
func (DB *BlockchainDB) Delete(k []byte, bt string, chainType int) bool {
	//lock.Lock()
	////log.Info("上锁")
	//defer lock.Unlock()
	var DBFileName string
	//0为主链，1为侧链
	switch chainType {
	case 0:
		DBFileName = "NodeMain_" + Constant.ListenPort + ".db"
	case 1:
		DBFileName = "NodeSide_" + Constant.ListenPort + ".db"

	}
	db, err := bolt.Open(fmt.Sprintf("%s%s", dataPath, DBFileName), 0600, nil)
	defer db.Close()
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bt))
		if bucket == nil {
			msg := "database delete warning:没有找到仓库：" + bt
			return errors.New(msg)
		}
		err := bucket.Delete(k)
		if err != nil {
			log.Panic(err)
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	//log.Info("解锁")
	return true
}

// DeleteBucket 删除仓库
func (DB *BlockchainDB) DeleteBucket(bt string, chainType int) bool {
	var DBFileName string
	//0为主链，1为侧链
	switch chainType {
	case 0:
		DBFileName = "NodeMain_" + Constant.ListenPort + ".db"
	case 1:
		DBFileName = "NodeSide_" + Constant.ListenPort + ".db"

	}
	db, err := bolt.Open(fmt.Sprintf("%s%s", dataPath, DBFileName), 0600, nil)
	defer db.Close()
	if err != nil {
		log.Panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(bt))
		if err != nil {
			log.Panic(err)
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return true
}

// UpdateHash 更新最新hash
func (DB *BlockchainDB) UpdateHash(key []byte, bt string, chainType int) {
	var DBFileName string
	//0为主链，1为侧链
	switch chainType {
	case 0:
		DBFileName = "NodeMain_" + Constant.ListenPort + ".db"
	case 1:
		DBFileName = "NodeSide_" + Constant.ListenPort + ".db"

	}
	db, err := bolt.Open(fmt.Sprintf("%s%s", dataPath, DBFileName), 0600, nil)
	defer db.Close()
	if err != nil {
		log.Panic(err)
	}

	var hash []byte
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bt))
		if bucket == nil {
			msg := "database view warning:没有找到仓库：" + string(bt)
			return errors.New(msg)
		}
		hash = bucket.Get(key)
		return nil
	})
	if err != nil {
		//log.Warn(err)
		return
	}
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bt))
		if bucket == nil {
			var err error
			bucket, err = tx.CreateBucket([]byte(bt))
			if err != nil {
				log.Panic(err)
			}
		}
		err := bucket.Put([]byte{'1'}, hash)
		if err != nil {
			log.Panic(err)
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

}

// IsBlotExist 判断数据库是否存在
func IsBlotExist(nodeID string) bool {
	var DBFileName = "Node_" + nodeID + ".db"
	_, err := os.Stat(fmt.Sprintf("%s%s", dataPath, DBFileName))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// IsBucketExist 判断仓库是否存在
func IsBucketExist(DB *BlockchainDB, bt string) bool {
	var isBucketExist bool
	var DBFileName = "Node_" + Constant.ListenPort + ".db"
	db, err := bolt.Open(fmt.Sprintf("%s%s", dataPath, DBFileName), 0600, nil)
	if err != nil {
		log.Panic(err)
	}
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bt))
		if bucket == nil {
			isBucketExist = false
		} else {
			isBucketExist = true
		}
		return nil
	})
	if err != nil {
		log.Panic("database IsBucketExist err:", err)
	}

	err = db.Close()
	if err != nil {
		log.Panic("db close err :", err)
	}
	return isBucketExist
}

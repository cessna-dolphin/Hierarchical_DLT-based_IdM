package Utils

import (
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

// GenRsaKeys 如果当前目录下不存在目录Keys，则创建目录，生成单个节点rsa公私秘钥
func GenRsaKeys() string {
	if !IsExist("./Keys") {
		log.Println("Keys directory not exists. Creating directory.")
		fmt.Println("检测到还未生成公私钥目录，正在生成公私钥 ...")
		err := os.Mkdir("Keys", 0644)
		if err != nil {
			log.Panic()
		}
	}
	//TODO 每次都从ClientPortInt开始遍历，直到当前节点编号i。复杂度高
	ClientPortInt, _ := strconv.Atoi(Constant.ClientPort)
	for i := ClientPortInt; i <= Constant.UENum+Constant.SPNum+ClientPortInt; i++ { //1007为客户端，1008为主节点，其余都为SP或UE节点
		if !IsExist("./Keys/" + strconv.Itoa(i)) {
			err := os.Mkdir("./Keys/"+strconv.Itoa(i), 0644)
			if err != nil {
				log.Panic()
			}
		} else {
			continue
		}
		priv, pub := GetKeyPair()
		privFileName := "Keys/" + strconv.Itoa(i) + "/N" + strconv.Itoa(i) + "_RSA_PIV"
		file, err := os.OpenFile(privFileName, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Panic(err)
		}
		defer file.Close()
		file.Write(priv)
		log.Println("Private key generated and stored.")

		pubFileName := "Keys/" + strconv.Itoa(i) + "/N" + strconv.Itoa(i) + "_RSA_PUB"
		file2, err := os.OpenFile(pubFileName, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Panic(err)
		}
		defer file2.Close()
		file2.Write(pub)
		log.Println("Public key generated and stored.")
		//fmt.Println("已生成RSA公私钥，节点", i)
		fmt.Println(" ---------------------------------------------------------------------------------")
		fmt.Println("Initialize The Node ...")
		fmt.Println("Generated RSA public and private key, node:", i)
		log.Println("RSA keys generated for node: ", i)

		return strconv.Itoa(i)
	}

	return "nil"
}

// 生成rsa公私钥
func GetKeyPair() (prvkey, pubkey []byte) {
	// 生成私钥文件
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	prvkey = pem.EncodeToMemory(block) //pem是保存私钥的一种证书格式
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	pubkey = pem.EncodeToMemory(block)
	return
}

// 数字签名
func RsaSignWithSha256(data []byte, keyBytes []byte) []byte {
	h := sha256.New()
	h.Write(data)
	hashed := h.Sum(nil)
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("private key error"))
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("ParsePKCS8PrivateKey err", err)
		panic(err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		fmt.Printf("Error from signing: %s\n", err)
		panic(err)
	}

	return signature
}

// 签名验证
func RsaVerySignWithSha256(data, signData, keyBytes []byte) bool {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("public key error"))
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	hashed := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(pubKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signData)
	if err != nil {
		panic(err)
	}
	return true
}

// 传入节点编号， 获取对应的公钥
func GetPubKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/N" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// 传入节点编号， 获取对应的私钥
func GetPrivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/N" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}

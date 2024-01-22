package main

import (
	"Hierarchical_IdM/BlockchainLayer/Constant"
	"Hierarchical_IdM/BlockchainLayer/Network"
	"fmt"
	"github.com/spf13/viper"
	"net/http"
)

func init() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	Constant.UENum = viper.GetInt("Hierarchical_IdM.UENum")
	Constant.SPNum = viper.GetInt("Hierarchical_IdM.SPNum")
}

func main() {
	Network.Server()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, err := fmt.Fprint(w, "Hello, World!")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

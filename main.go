package main

import (
	log "bedrock-claude-proxy/log"
	"bedrock-claude-proxy/pkg"
	"flag"
	"runtime"

	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
	conf_path := flag.String("c", "conf.json", "config json file")
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	var conf *pkg.Config
	// var err error
	if len(*conf_path) > 0 {
		conf, err = pkg.NewConfigFromLocal(*conf_path)
		if err != nil {
			log.Logger.Error(err)
			conf = &pkg.Config{}
		}
	} else {
		conf = &pkg.Config{}
	}

	conf.MarginWithENV()

	log.Logger.Debug("show config detail:")
	log.Logger.Debug(conf.ToJSON())

	service := pkg.NewHttpService(conf)
	service.Start()
}

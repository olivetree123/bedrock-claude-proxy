package main

import (
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
			pkg.Log.Error(err)
			conf = &pkg.Config{}
		}
	} else {
		conf = &pkg.Config{}
	}

	conf.MarginWithENV()

	pkg.Log.Debug("show config detail:")
	pkg.Log.Debug(conf.ToJSON())

	service := pkg.NewHttpService(conf)
	service.Start()
}

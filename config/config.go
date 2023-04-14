package config

import (
	"errors"
	"github.com/Unknwon/goconfig"
	"log"
	"os"
)

const configFile = "/conf/conf.ini"

var File *goconfig.ConfigFile
//加载此文件的时候 会先走初始化方法
func init()  {
	//拿到当前的程序的目录
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configPath := currentDir + configFile

	//参数  mssgserver.exe  D:/xxx
	len := len(os.Args)
	if len > 1 {
		dir := os.Args[1]
		if dir != "" {
			configPath = dir + configFile
		}
	}
	log.Println("配置文件的路径",configPath)
	if !fileExist(configPath) {
		panic(errors.New("配置文件不存在"))
	}
	//文件系统的读取 
	File,err = goconfig.LoadConfigFile(configPath)
	if err != nil {
		log.Fatal("读取配置文件出错:",err)
	}
}

func fileExist(fileName string) bool {
	_,err := os.Stat(fileName)
	return err == nil || os.IsExist(err)
}
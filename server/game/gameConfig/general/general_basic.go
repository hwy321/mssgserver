package general

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type generalBasic struct {
	Title	string    `json:"title"`
	Levels	[]gLevel `json:"levels"`
}
type gLevel struct {
	Level		int8`json:"level"`
	Exp			int `json:"exp"`
	Soldiers	int `json:"soldiers"`
}

var GeneralBasic = &generalBasic{}
var generalBasicFile = "/conf/game/general/general_basic.json"

func (g *generalBasic) Load()  {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configFile := currentDir + generalBasicFile
	//参数  mssgserver.exe  D:/xxx
	length := len(os.Args)
	if length > 1 {
		dir := os.Args[1]
		if dir != "" {
			configFile = dir + generalBasicFile
		}
	}
	data,err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data,g)
	if err != nil {
		log.Println("json格式不正确，解析出错")
		panic(err)
	}
}

func (g *generalBasic) GetLevel(level int8) *gLevel {
	for _, v:= range g.Levels{
		if v.Level == level {
			return &v
		}
	}
	return nil
}

func (g *generalBasic) ExpToLevel(exp int) (int8, int) {
	var level int8 = 0
		limitExp := g.Levels[len(g.Levels)-1].Exp
		for _, v := range g.Levels {
			if exp >= v.Exp && v.Level > level {
				level = v.Level
			}
		}

		if limitExp < exp {
			return level, limitExp
		}else{
			return level, exp
	}
}

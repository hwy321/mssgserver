package gameConfig

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

//地图资源配置
var MapBCConf mapBuildCustomConf

type bcLevel struct {
	Level    int8              `json:"level"`
	Time     int              `json:"time"`	//升级需要的时间
	Durable  int              `json:"durable"`
	Defender int              `json:"defender"`
	Need     NeedRes `json:"need"`
	Result   result           `json:"result"`
}


type customConf struct {
	Type   int8        	`json:"type"`
	Name   string      	`json:"name"`
	Levels []bcLevel 		`json:"levels"`
}

type result struct {
	ArmyCnt int 		`json:"army_cnt"`
}

type BCLevelCfg struct {
	bcLevel
	Type   int8        	`json:"type"`
	Name   string      	`json:"name"`
}

type mapBuildCustomConf struct {
	Title   string   				`json:"title"`
	Cfg		[]customConf 			`json:"cfg"`
	cfgMap  map[int8]customConf
}


const mapBuildCustomConfFile = "/conf/game/map_build_custom.json"


func (m *mapBuildCustomConf) Load()  {
	m.cfgMap = make(map[int8]customConf)
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configPath := currentDir + mapBuildCustomConfFile

	//参数  mssgserver.exe  D:/xxx
	len := len(os.Args)
	if len > 1 {
		dir := os.Args[1]
		if dir != "" {
			configPath = dir + mapBuildCustomConfFile
		}
	}
	data,err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data,m)
	if err != nil {
		log.Println("json格式不正确，解析出错")
		panic(err)
	}
	for _,v := range m.Cfg{
		m.cfgMap[v.Type] = v
	}
}


func (m *mapBuildCustomConf) BuildConfig(cfgType int8, level int8) (*BCLevelCfg, bool) {
	if c, ok := m.cfgMap[cfgType]; ok {
		if len(c.Levels) < int(level){
			return nil, false
		}

		lc := c.Levels[level-1]
		cfg := BCLevelCfg{Type: cfgType, Name: c.Name}
		cfg.Level = level
		cfg.Need = lc.Need
		cfg.Result = lc.Result
		cfg.Durable = lc.Durable
		cfg.Time = lc.Time
		cfg.Result = lc.Result

		return &cfg, true
	}
	return nil, false
}

//可容纳队伍数量
func (m* mapBuildCustomConf) GetHoldArmyCnt(cfgType int8, level int8) int {
	cfg, ok := m.BuildConfig(cfgType, level)
	if ok == false {
		return 0
	}
	return cfg.Result.ArmyCnt
}

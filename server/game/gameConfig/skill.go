package gameConfig

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
)

var Skill skill

type skill struct {
	skills []Conf
	skillConfMap map[int]Conf
	outline outline
}


type trigger struct {
	Type int    `json:"type"`
	Des  string `json:"des"`
}

type triggerType struct {
	Des  string     `json:"des"`
	List [] trigger `json:"list"`
}

type effect struct {
	Type   int    `json:"type"`
	Des    string `json:"des"`
	IsRate bool   `json:"isRate"`
}

type effectType struct {
	Des  string    `json:"des"`
	List [] effect `json:"list"`
}

type target struct {
	Type   int    `json:"type"`
	Des    string `json:"des"`
}

type targetType struct {
	Des  string    `json:"des"`
	List [] target `json:"list"`
}


type outline struct {
	TriggerType triggerType `json:"trigger_type"`	//触发类型
	EffectType  effectType  `json:"effect_type"`	//效果类型
	TargetType  targetType  `json:"target_type"`	//目标类型
}


type level struct {
	Probability int   `json:"probability"`  //发动概率
	EffectValue []int `json:"effect_value"` //效果值
	EffectRound []int `json:"effect_round"` //效果持续回合数
}

type Conf struct {
	CfgId		  int	  `json:"cfgId"`
	Name          string  `json:"name"`
	Trigger       int     `json:"trigger"` 			//发起类型
	Target        int     `json:"target"`  			//目标类型
	Des           string  `json:"des"`
	Limit         int     `json:"limit"`          	//可以被武将装备上限
	Arms          []int   `json:"arms"`           	//可以装备的兵种
	IncludeEffect []int   `json:"include_effect"` 	//技能包括的效果
	Levels        []level `json:"levels"`
}
const skillFile = "/conf/game/skill/skill_outline.json"
const skillPath = "/conf/game/skill/"

func (s *skill) Load()  {
	s.skills = make([]Conf,0)
	s.skillConfMap = make(map[int]Conf)

	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configFile := currentDir + skillFile
	configPath := currentDir + skillPath
	//参数  mssgserver.exe  D:/xxx
	length := len(os.Args)
	if length > 1 {
		dir := os.Args[1]
		if dir != "" {
			configFile = dir + skillFile
			configPath = dir + skillPath
		}
	}
	data,err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data,&s.outline)
	if err != nil {
		log.Println("json格式不正确，解析出错")
		panic(err)
	}
	files ,err := ioutil.ReadDir(configPath)
	if err != nil {
		log.Println("读取技能目录出错")
		panic(err)
	}
	for _, v := range files{
		if v.IsDir() {
			name := v.Name()
			dirPath := path.Join(configPath,name)
			skillFiles ,err := ioutil.ReadDir(dirPath)
			if err != nil {
				panic(err)
			}
			for _, sf := range skillFiles{
				if sf.IsDir() {
					continue
				}
				sFileName := sf.Name()
				skillDirFile := path.Join(dirPath,sFileName)
				sData,err := ioutil.ReadFile(skillDirFile)
				if err != nil {
					panic(err)
				}
				conf := Conf{}
				err = json.Unmarshal(sData,&conf)
				if err != nil {
					panic(err)
				}
				s.skills = append(s.skills,conf)
				s.skillConfMap[conf.CfgId] = conf
			}
		}
	}
	log.Println(s)
}
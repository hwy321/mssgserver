package general

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type gArmsCondition struct {
	Level		    int     `json:"level"`
	StarLevel		int     `json:"star_lv"`
}


type gArmsCost struct {
	Gold		    int     `json:"gold"`
}


type gArms struct {
	Id         int         		`json:"id"`
	Name       string      		`json:"name"`
	Condition  gArmsCondition 	`json:"condition"`
	ChangeCost gArmsCost   		`json:"change_cost"`
	HarmRatio  []int     		`json:"harm_ratio"`
}


type Arms struct {
	Title	string `json:"title"`
	Arms	[]gArms `json:"arms"`
	AMap    map[int]gArms
}

var GeneralArms = &Arms{
	AMap: make(map[int]gArms),
}
var generalArmsFile = "/conf/game/general/general_arms.json"

func (g *Arms) Load()  {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configFile := currentDir + generalArmsFile
	//参数  mssgserver.exe  D:/xxx
	length := len(os.Args)
	if length > 1 {
		dir := os.Args[1]
		if dir != "" {
			configFile = dir + generalArmsFile
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
	for _, v:= range g.Arms{
		g.AMap[v.Id] = v
	}
}


func (a *Arms) GetArm(id int) (gArms, error){
	return a.AMap[id], nil
}

func (a* Arms) GetHarmRatio(attId, defId int) float64{
	attArm, ok1 := a.AMap[attId]
	_, ok2 := a.AMap[defId]
	if ok1 && ok2 {
		return float64(attArm.HarmRatio[defId-1])/100.0
	}else{
		return 1.0
	}
}

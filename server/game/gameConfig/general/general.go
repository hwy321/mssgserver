package general

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
)

type general struct {
	Title string 	`json:"title"`
	GArr  []generalDetail    	`json:"list"`
	GMap  map[int]generalDetail
	totalProbability int
}
type generalDetail struct {
	Name     		string	`json:"name"`
	CfgId    		int		`json:"cfgId"`
	Force    		int		`json:"force"`  //武力
	Strategy 		int		`json:"strategy"` //策略
	Defense  		int		`json:"defense"` //防御
	Speed    		int		`json:"speed"` //速度
	Destroy      	int   	`json:"destroy"` //破坏力
	ForceGrow    	int   	`json:"force_grow"`
	StrategyGrow 	int   	`json:"strategy_grow"`
	DefenseGrow  	int   	`json:"defense_grow"`
	SpeedGrow   	int   	`json:"speed_grow"`
	DestroyGrow  	int   	`json:"destroy_grow"`
	Cost         	int8  	`json:"cost"`
	Probability  	int   	`json:"probability"`
	Star         	int8   	`json:"star"`
	Arms         	[]int 	`json:"arms"`
	Camp         	int8  	`json:"camp"`
}

var General = &general{}
var generalFile = "/conf/game/general/general.json"

func (g *general) Load()  {
	g.GArr = make([]generalDetail,0)
	g.GMap = make(map[int]generalDetail)
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configFile := currentDir + generalFile
	//参数  mssgserver.exe  D:/xxx
	length := len(os.Args)
	if length > 1 {
		dir := os.Args[1]
		if dir != "" {
			configFile = dir + generalFile
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

	for _,v := range g.GArr{
		g.GMap[v.CfgId] = v
		g.totalProbability += v.Probability
	}
}
//随机武将
func (g *general) Rand() int {
	// 7+12=19   0-19 8
	rate := rand.Intn(g.totalProbability)
	var cur = 0
	for _,v := range g.GArr{
		if rate >= cur && rate < cur + v.Probability {
			return v.CfgId
		}
		cur += v.Probability
	}
	return 0
}

func (g *general) Cost(cfgId int) int8 {
	return g.GMap[cfgId].Cost
}
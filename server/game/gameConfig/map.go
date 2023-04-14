package gameConfig

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"mssgserver/server/game/global"
	"os"
)

type mapData struct {
	Width	int 			`json:"w"`
	Height	int				`json:"h"`
	List	[][]int			`json:"list"`
}

type NationalMap struct {
	MId			int		`xorm:"mid"`
	X			int		`xorm:"x"`
	Y			int		`xorm:"y"`
	Type		int8	`xorm:"type"`
	Level		int8	`xorm:"level"`
}
const (
	MapBuildSysFortress = 50	//系统要塞
	MapBuildFortress = 56		//玩家要塞
	MapBuildSysCity = 51		//系统城市
)

var MapRes = &mapRes{
	Confs: make(map[int]NationalMap),
	SysBuild: make(map[int]NationalMap),
}

type mapRes struct {
	Confs map[int]NationalMap
	SysBuild map[int]NationalMap
}

const mapFile = "/conf/game/map.json"
func (m *mapRes) Load()  {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	configPath := currentDir + mapFile

	//参数  mssgserver.exe  D:/xxx
	length := len(os.Args)
	if length > 1 {
		dir := os.Args[1]
		if dir != "" {
			configPath = dir + mapFile
		}
	}
	data,err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	mapData := &mapData{}
	err = json.Unmarshal(data,mapData)
	if err != nil {
		log.Println("json格式不正确，解析出错")
		panic(err)
	}
	global.MapWith = mapData.Width
	global.MapHeight = mapData.Height
	log.Println("list len",len(mapData.List))
	for index,v := range mapData.List{
		t := int8(v[0])
		l := int8(v[1])
		nm := NationalMap{
			X: index%global.MapWith,
			Y: index/global.MapHeight,
			Type: t,
			Level: l,
			MId: index,
		}
		m.Confs[index] = nm
		if t == MapBuildSysCity || t == MapBuildSysFortress {
			m.SysBuild[index] = nm
		}
	}
}

func (m *mapRes) ToPositionMap(x,y int) (NationalMap,bool)  {
	posId := global.ToPosition(x,y)
	nm,ok := m.Confs[posId]
	return nm,ok
}

func (m *mapRes) PositionBuild(x int, y int) (NationalMap, bool) {
	posId := global.ToPosition(x,y)
	nm,ok := m.Confs[posId]
	return nm,ok
}

func (m *mapRes) IsCanBuild(x int, y int) bool {
	posId := global.ToPosition(x,y)
	nm,ok := m.Confs[posId]
	if ok {
		if nm.Type == 0 {
			return false
		}
		return true
	}
	return false
}
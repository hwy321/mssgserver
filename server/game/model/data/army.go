package data

import (
	"encoding/json"
	"fmt"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/game/global"
	"mssgserver/server/game/model"
	"mssgserver/utils"
	"time"
	"xorm.io/xorm"
)

const (
	ArmyCmdIdle   		= 0	//空闲
	ArmyCmdAttack 		= 1	//攻击
	ArmyCmdDefend 		= 2	//驻守
	ArmyCmdReclamation 	= 3	//屯垦
	ArmyCmdBack   		= 4 //撤退
	ArmyCmdConscript  	= 5 //征兵
	ArmyCmdTransfer  	= 6 //调动
)

const (
	ArmyStop  		= 0
	ArmyRunning  	= 1
)
var ArmyDao = &armyDao{
	armyChan: make(chan *Army,100),
}
type armyDao struct {
	armyChan chan *Army
}

func (a *armyDao) running() {
	for  {
		select {
		case army := <- a.armyChan:
			if army.Id > 0 {
				db.Engine.Table(army).ID(army.Id).Cols(
					"soldiers", "generals", "conscript_times",
					"conscript_cnts", "cmd", "from_x", "from_y", "to_x",
					"to_y", "start", "end").Update(army)
			}
		}
	}

}

func init()  {
	go ArmyDao.running()
}
//军队
type Army struct {
	Id             		int    		`xorm:"id pk autoincr"`
	RId            		int    		`xorm:"rid"`
	CityId         		int    		`xorm:"cityId"`
	Order          		int8   		`xorm:"a_order"`
	Generals       		string 		`xorm:"generals"`
	Soldiers       		string 		`xorm:"soldiers"`
	ConscriptTimes 		string 		`xorm:"conscript_times"`	//征兵结束时间，json数组
	ConscriptCnts  		string 		`xorm:"conscript_cnts"`		//征兵数量，json数组
	Cmd                	int8       	`xorm:"cmd"`
	FromX              	int        	`xorm:"from_x"`
	FromY              	int        	`xorm:"from_y"`
	ToX                	int        	`xorm:"to_x"`
	ToY                	int        	`xorm:"to_y"`
	Start              	time.Time  	`json:"-"xorm:"start"`
	End                	time.Time  	`json:"-"xorm:"end"`
	State              	int8       	`xorm:"-"` 				//状态:0:running,1:stop
	GeneralArray       	[]int      	`json:"-" xorm:"-"`
	SoldierArray       	[]int      	`json:"-" xorm:"-"`
	ConscriptTimeArray 	[]int64    	`json:"-" xorm:"-"`
	ConscriptCntArray  	[]int      	`json:"-" xorm:"-"`
	Gens               	[]*General 	`json:"-" xorm:"-"`
	CellX              	int        	`json:"-" xorm:"-"`
	CellY              	int        	`json:"-" xorm:"-"`
}

func (a *Army) TableName() string {
	return "army"
}
//执行update操作之前进行的操作
func (a *Army) BeforeUpdate() {
	a.beforeModify()
}
func (a *Army) beforeModify() {
	data, _ := json.Marshal(a.GeneralArray)
	a.Generals = string(data)

	data, _ = json.Marshal(a.SoldierArray)
	a.Soldiers = string(data)

	data, _ = json.Marshal(a.ConscriptTimeArray)
	a.ConscriptTimes = string(data)

	data, _ = json.Marshal(a.ConscriptCntArray)
	a.ConscriptCnts = string(data)
}
//执行insert操作之前执行的
func (a *Army) BeforeInsert() {
	a.beforeModify()
}

func (a *Army) AfterSet(name string, cell xorm.Cell){
	//[0,0,0]
	if name == "generals"{
		a.GeneralArray = []int{0,0,0}
		if cell != nil{
			gs, ok := (*cell).([]uint8)
			if ok {
				json.Unmarshal(gs, &a.GeneralArray)
				fmt.Println(a.GeneralArray)
			}
		}
	}else if name == "soldiers"{
		a.SoldierArray = []int{0,0,0}
		if cell != nil{
			ss, ok := (*cell).([]uint8)
			if ok {
				json.Unmarshal(ss, &a.SoldierArray)
				fmt.Println(a.SoldierArray)
			}
		}
	}else if name == "conscript_times"{
		a.ConscriptTimeArray = []int64{0,0,0}
		if cell != nil{
			ss, ok := (*cell).([]uint8)
			if ok {
				json.Unmarshal(ss, &a.ConscriptTimeArray)
				fmt.Println(a.ConscriptTimeArray)
			}
		}
	}else if name == "conscript_cnts"{
		a.ConscriptCntArray = []int{0,0,0}
		if cell != nil{
			ss, ok := (*cell).([]uint8)
			if ok {
				json.Unmarshal(ss, &a.ConscriptCntArray)
				fmt.Println(a.ConscriptCntArray)
			}
		}
	}
}

func (a *Army) IsCellView() bool{
	return true
}
func (a *Army) IsCanView(rid, x, y int) bool{
	return true
}
func (a *Army) BelongToRId() []int{
	return []int{a.RId}
}

func (a *Army) PushMsgName() string{
	return "army.push"
}


func (a *Army) Position() (int, int){
	diffTime := a.End.Unix()-a.Start.Unix()
	passTime := time.Now().Unix()-a.Start.Unix()
	rate := float32(passTime)/float32(diffTime)
	x := 0
	y := 0
	if a.Cmd == ArmyCmdBack {
		diffX := a.FromX - a.ToX
		diffY := a.FromY - a.ToY
		x = int(rate*float32(diffX)) + a.ToX
		y = int(rate*float32(diffY)) + a.ToY
	}else{
		diffX := a.ToX - a.FromX
		diffY := a.ToY - a.FromY
		x = int(rate*float32(diffX)) + a.FromX
		y = int(rate*float32(diffY)) + a.FromY
	}

	x = utils.MinInt(utils.MaxInt(x, 0), global.MapWith)
	y = utils.MinInt(utils.MaxInt(y, 0), global.MapHeight)
	return x, y
}

func (a *Army) TPosition() (int, int){
	return a.ToX, a.ToY
}

func (a *Army) Push(){
	net.Mgr.Push(a)
}


func (a *Army) ToModel() interface{}{
	p := model.Army{}
	p.CityId = a.CityId
	p.Id = a.Id
	p.UnionId = GetUnion(a.RId)
	p.Order = a.Order
	p.Generals = a.GeneralArray
	p.Soldiers = a.SoldierArray
	p.ConTimes = a.ConscriptTimeArray
	p.ConCnts = a.ConscriptCntArray
	p.Cmd = a.Cmd
	p.State = a.State
	p.FromX = a.FromX
	p.FromY = a.FromY
	p.ToX = a.ToX
	p.ToY = a.ToY
	p.Start = a.Start.Unix()
	p.End = a.End.Unix()
	return p
}
//pos 0-2
func (a *Army) PositionCanModify(pos int) bool {
	if pos>=3 || pos <0{
		return false
	}
	if a.Cmd == ArmyCmdIdle {
		return true
	}else if a.Cmd == ArmyCmdConscript {
		endTime := a.ConscriptTimeArray[pos]
		return endTime == 0
	}else{
		return false
	}
}

func (a *Army) SyncExecute() {
	ArmyDao.armyChan <- a
	//通知客户端更新
	a.Push()
	a.CellX,a.CellY = a.Position()
}

func (a *Army) CheckConscript() {
	if a.Cmd == ArmyCmdConscript {
		finish := true
		for i,v := range a.ConscriptTimeArray{
			var cur = time.Now().Unix()
			if cur >= v {
				//征兵完成
				a.SoldierArray[i] = a.SoldierArray[i] + a.ConscriptCntArray[i]
				a.ConscriptTimeArray[i] = 0
				a.ConscriptCntArray[i] = 0
			}else{
				finish = false
			}
		}
		if finish {
			a.Cmd = ArmyCmdIdle
		}
	}

}

func (a *Army) IsCanOutWar() bool {
	//空闲状态
	return a.Gens != nil && a.Cmd == ArmyCmdIdle
}

func (a *Army) IsIdle() bool {
	return a.Cmd == ArmyCmdIdle
}

func (a *Army) ToSoldier() {
	if a.SoldierArray != nil {
		data, _ := json.Marshal(a.SoldierArray)
		a.Soldiers = string(data)
	}
}

func (a *Army) ToGeneral() {
	if a.GeneralArray != nil {
		data, _ := json.Marshal(a.GeneralArray)
		a.Generals = string(data)
	}
}

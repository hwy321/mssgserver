package data

import (
	"encoding/json"
	"fmt"
	"log"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/game/gameConfig/general"
	"mssgserver/server/game/model"
	"time"
	"xorm.io/xorm"
)

const (
	GeneralNormal      	= 0 //正常
	GeneralComposeStar 	= 1 //星级合成
	GeneralConvert 		= 2 //转换
)
const SkillLimit = 3

var GeneralDao = &generalDao{
	genChan: make(chan *General,100),
}
type generalDao struct {
	genChan chan *General
}

func (g *generalDao) running() {
	for  {
		select {
		case gen := <- g.genChan:
			if gen.Id > 0 {
				_,err := db.Engine.Table(gen).ID(gen.Id).Cols(
					"level", "exp", "order", "cityId",
					"physical_power", "star_lv", "has_pr_point",
					"use_pr_point", "force_added", "strategy_added",
					"defense_added", "speed_added", "destroy_added",
					"parentId", "compose_type", "skills", "state").Update(gen)
				if err != nil {
					log.Println("generalDao running error",err)
				}
			}
		}
	}

}

func init()  {
	go GeneralDao.running()
}
type General struct {
	Id            int       		`xorm:"id pk autoincr"`
	RId           int       		`xorm:"rid"`
	CfgId         int       		`xorm:"cfgId"`
	PhysicalPower int       		`xorm:"physical_power"`
	Level         int8      		`xorm:"level"`
	Exp           int       		`xorm:"exp"`
	Order         int8      		`xorm:"order"`
	CityId        int       		`xorm:"cityId"`
	CreatedAt     time.Time 		`xorm:"created_at"`
	CurArms       int       		`xorm:"arms"`
	HasPrPoint    int       		`xorm:"has_pr_point"`
	UsePrPoint    int       		`xorm:"use_pr_point"`
	AttackDis     int  				`xorm:"attack_distance"`
	ForceAdded    int  				`xorm:"force_added"`
	StrategyAdded int  				`xorm:"strategy_added"`
	DefenseAdded  int  				`xorm:"defense_added"`
	SpeedAdded    int  				`xorm:"speed_added"`
	DestroyAdded  int  				`xorm:"destroy_added"`
	StarLv        int8  			`xorm:"star_lv"`
	Star          int8  			`xorm:"star"`
	ParentId      int  				`xorm:"parentId"`
	Skills		  string			`xorm:"skills"`
	SkillsArray   []*model.GSkill	`xorm:"-"`
	State         int8 				`xorm:"state"`
}

func (g *General) TableName() string {
	return "general"
}


func (g *General) AfterSet(name string, cell xorm.Cell){
	if name == "skills"{
		g.SkillsArray = make([]*model.GSkill, 3)
		if cell != nil{
			gs, ok := (*cell).([]uint8)
			if ok {
				json.Unmarshal(gs, &g.Skills)
				fmt.Println(g.SkillsArray)
			}
		}
	}
}

func (g *General) beforeModify()  {
	data, _ := json.Marshal(g.SkillsArray)
	g.Skills = string(data)
}

func (g *General) BeforeInsert() {
	g.beforeModify()
}

func (g *General) BeforeUpdate() {
	g.beforeModify()
}

func (g *General) ToModel() interface{}{
	p := model.General{}
	p.CityId = g.CityId
	p.Order = g.Order
	p.PhysicalPower = g.PhysicalPower
	p.Id = g.Id
	p.CfgId = g.CfgId
	p.Level = g.Level
	p.Exp = g.Exp
	p.CurArms = g.CurArms
	p.HasPrPoint = g.HasPrPoint
	p.UsePrPoint = g.UsePrPoint
	p.AttackDis = g.AttackDis
	p.ForceAdded = g.ForceAdded
	p.StrategyAdded = g.StrategyAdded
	p.DefenseAdded = g.DefenseAdded
	p.SpeedAdded = g.SpeedAdded
	p.DestroyAdded = g.DestroyAdded
	p.StarLv = g.StarLv
	p.Star = g.Star
	p.State = g.State
	p.ParentId = g.ParentId
	p.Skills = g.SkillsArray
	return p
}

func (g *General) SyncExecute() {
	GeneralDao.genChan <- g
	g.Push()
}

func (g *General) GetDestroy() int {
	cfg,ok := general.General.GMap[g.CfgId]
	if ok 	{
		return cfg.Destroy + cfg.DestroyGrow * int(g.Level) + g.DestroyAdded
	}
	return 0
}

func (g *General) GetForce() int {
	cfg,ok := general.General.GMap[g.CfgId]
	if ok {
		return cfg.Force + cfg.ForceGrow * int(g.Level) + g.ForceAdded
	}
	return 0
}

func (g *General) GetStrategy() int {
	cfg,ok := general.General.GMap[g.CfgId]
	if ok {
		return cfg.Strategy + cfg.StrategyGrow * int(g.Level) + g.StrategyAdded
	}
	return 0
}
func (g *General) GetSpeed() int {
	cfg,ok := general.General.GMap[g.CfgId]
	if ok {
		return cfg.Speed + cfg.SpeedGrow * int(g.Level) + g.SpeedAdded
	}
	return 0
}
func (g *General) GetDefense() int {
	cfg,ok := general.General.GMap[g.CfgId]
	if ok {
		return cfg.Defense + cfg.DefenseGrow * int(g.Level) + g.DefenseAdded
	}
	return 0
}



/* 推送同步 begin */
func (g *General) IsCellView() bool{
	return false
}

func (g *General) IsCanView(rid, x, y int) bool{
	return false
}

func (g *General) BelongToRId() []int{
	return []int{g.RId}
}

func (g *General) PushMsgName() string{
	return "general.push"
}

func (g *General) Position() (int, int){
	return -1, -1
}

func (g *General) TPosition() (int, int){
	return -1, -1
}

func (g *General) Push(){
	net.Mgr.Push(g)
}
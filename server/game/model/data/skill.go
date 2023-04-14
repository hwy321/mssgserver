package data

import (
	"encoding/json"
	"fmt"
	"mssgserver/server/game/model"
	"xorm.io/xorm"
)

//军队
type Skill struct {
	Id             	int    `xorm:"id pk autoincr"`
	RId          	int    `xorm:"rid"`
	CfgId          	int    `xorm:"cfgId"`
	BelongGenerals 	string `xorm:"belong_generals"`
	Generals 		[]int  `xorm:"-"`
}

func (a *Skill) AfterSet(name string, cell xorm.Cell){
	//[0,0,0]
	if name == "belong_generals"{
		a.Generals = make([]int,0)
		if cell != nil{
			gs, ok := (*cell).([]uint8)
			if ok {
				json.Unmarshal(gs, &a.Generals)
				fmt.Println(a.Generals)
			}
		}
	}
}
func NewSkill(rid int, cfgId int) *Skill{
	return &Skill{
		CfgId: cfgId,
		RId: rid,
		Generals: []int{},
		BelongGenerals: "[]",
	}
}

func (s *Skill) TableName() string {
	return "skill"
}

func (s *Skill) ToModel() interface{}{
	p := model.Skill{}
	p.Id = s.Id
	p.CfgId = s.CfgId
	p.Generals = s.Generals
	return p
}

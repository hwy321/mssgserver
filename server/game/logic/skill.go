package logic

import (
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/server/common"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
)

var SkillService = &skillService{}
type skillService struct {

}

func (s *skillService) GetSkills(rid int) ([]model.Skill,error)  {
	mrs := make([]data.Skill,0)
	mr := &data.Skill{}
	err := db.Engine.Table(mr).
		Where("rid=? ",rid).Find(&mrs)
	if err != nil {
		log.Println("技能查询出错",err)
		return nil, common.New(constant.DBError,"技能查询出错")
	}
	modelMrs := make([]model.Skill,0)
	for _,v := range mrs{
		modelMrs = append(modelMrs,v.ToModel().(model.Skill))
	}
	return modelMrs,nil
}

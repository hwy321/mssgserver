package logic

import (
	"encoding/json"
	"log"
	"math/rand"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/gameConfig/general"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"mssgserver/utils"
	"time"
)



//最大回合数
const maxRound = 10

type warRound struct {
	Battle	[][]int	`json:"b"`
}

type WarResult struct {
	Round 	[]*warRound
	Result	int			//0失败，1平，2胜利
}

type ArmyWar struct {
	attack *data.Army
	defense *data.Army
	attackPos []*armyPosition
	defensePos []*armyPosition
}

func (w *ArmyWar) init() {
	//城内设施加成
	attackAdds := []int{0,0,0,0}
	if w.attack.CityId > 0{
		attackAdds = CityFacilityService.GetAdditions(w.attack.CityId,
			gameConfig.TypeForce,
			gameConfig.TypeDefense,
			gameConfig.TypeSpeed,
			gameConfig.TypeStrategy)
	}
	//
	defenseAdds := []int{0,0,0,0}
	if w.defense.CityId > 0{
		defenseAdds = CityFacilityService.GetAdditions(w.defense.CityId,
			gameConfig.TypeForce,
			gameConfig.TypeDefense,
			gameConfig.TypeSpeed,
			gameConfig.TypeStrategy)
	}
	//TODO 阵营加成
	w.attackPos = make([]*armyPosition, 0)
	w.defensePos = make([]*armyPosition, 0)

	for i, g := range w.attack.Gens {
		if g == nil {
			w.attackPos = append(w.attackPos, nil)
		}else{
			pos := &armyPosition{
				general:  g,
				soldiers: w.attack.SoldierArray[i],
				force:    g.GetForce()  + attackAdds[0] ,
				defense:  g.GetDefense() + attackAdds[1] ,
				speed:    g.GetSpeed() + attackAdds[2] ,
				strategy: g.GetStrategy() + attackAdds[3],
				destroy:  g.GetDestroy() ,
				arms:     g.CurArms,
				position: i,
			}
			w.attackPos = append(w.attackPos, pos)
		}
	}

	for i, g := range w.defense.Gens {
		if g == nil {
			w.defensePos = append(w.defensePos, nil)
		}else{
			pos := &armyPosition{
				general:  g,
				soldiers: w.defense.SoldierArray[i],
				force:    g.GetForce() + defenseAdds[0] ,
				defense:  g.GetDefense() + defenseAdds[1] ,
				speed:    g.GetSpeed() + defenseAdds[2] ,
				strategy: g.GetStrategy() + defenseAdds[3] ,
				destroy:  g.GetDestroy(),
				arms:     g.CurArms,
				position: i,
			}
			w.defensePos = append(w.defensePos, pos)
		}
	}

}

func (w *ArmyWar) battle() []*warRound {
	//随机出手  根据攻击和防御 兵种的克制  扣减士兵
	//结束的条件 主将 士兵为0 或者到达最大回合数
	rounds := make([]*warRound, 0)
	cur := 0
	for true{
		r, isEnd := w.round()
		rounds = append(rounds, r)
		cur += 1
		if cur >= maxRound || isEnd{
			break
		}
	}

	for i := 0; i < 3; i++ {
		if w.attackPos[i] != nil {
			w.attack.SoldierArray[i] = w.attackPos[i].soldiers
		}
		if w.defensePos[i] != nil {
			w.defense.SoldierArray[i] = w.defensePos[i].soldiers
		}
	}

	return rounds
}

func (w *ArmyWar) round() (*warRound, bool) {
	war := &warRound{}
	n := rand.Intn(10)
	attack := w.attackPos
	defense := w.defensePos

	isEnd := false
	//随机先手
	if n % 2 == 0{
		attack = w.defensePos
		defense = w.attackPos
	}

	for _, att := range attack {

		////////攻击方begin//////////
		if att == nil || att.soldiers == 0{
			continue
		}
		//随机防守方接收伤害的位置
		def, _ := w.randArmyPosition(defense)
		if def == nil{
			isEnd = true
			goto end
		}

		attHarmRatio := general.GeneralArms.GetHarmRatio(att.arms, def.arms)
		attHarm := float64(utils.AbsInt(att.force-def.defense)*att.soldiers)*attHarmRatio*0.0005
		if att.force < def.defense {
			//伤害减免
			attHarm = attHarm * 0.1
		}
		attKill := int(attHarm)
		//伤害值 最大只能是对方的兵力
		attKill = utils.MinInt(attKill, def.soldiers)
		//防守方 扣减士兵
		def.soldiers -= attKill
		//攻击方加经验
		att.general.Exp += attKill*5


		//大营干死了，直接结束
		if def.position == 0 && def.soldiers == 0 {
			isEnd = true
			b := battle{AId: att.general.Id, ALoss: 0, DId: def.general.Id, DLoss: attKill}
			war.Battle = append(war.Battle, b.to())
			goto end
		}
		////////攻击方end//////////

		////////防守方begin//////////
		if def.soldiers == 0 || att.soldiers == 0{
			continue
		}

		defHarmRatio := general.GeneralArms.GetHarmRatio(def.arms, att.arms)
		defHarm := float64(utils.AbsInt(def.force-att.defense)*def.soldiers)*defHarmRatio*0.0005
		if def.force < att.defense {
			//伤害减免
			defHarm = defHarm * 0.1
		}
		defKill := int(defHarm)

		defKill = utils.MinInt(defKill, att.soldiers)
		att.soldiers -= defKill
		def.general.Exp += defKill*5

		b := battle{AId: att.general.Id, ALoss: defKill, DId: def.general.Id, DLoss: attKill}
		war.Battle = append(war.Battle, b.to())

		//大营干死了，直接结束
		if att.position == 0 && att.soldiers == 0 {
			isEnd = true
			goto end
		}
		////////防守方end//////////

	}

end:
	return war, isEnd
}

func (w *ArmyWar) randArmyPosition(pos []*armyPosition) (*armyPosition, int) {
	isEmpty := true
	for _, v := range pos {
		if v != nil && v.soldiers != 0 {
			isEmpty = false
			break
		}
	}

	if isEmpty {
		return nil, -1
	}


	for true {
		r := rand.Intn(100)
		index := r % len(pos)
		if pos[index] != nil && pos[index].soldiers != 0{
			return pos[index], index
		}
	}

	return nil, -1
}

//战斗位置的属性
type armyPosition struct {
	general  *data.General
	soldiers int //兵力
	force    int //武力
	strategy int //策略
	defense  int //防御
	speed    int //速度
	destroy  int //破坏
	arms     int //兵种
	position int //位置
}

type battle struct {
	AId   int `json:"a_id"`   //本回合发起攻击的武将id
	DId   int `json:"d_id"`   //本回合防御方的武将id
	ALoss int `json:"a_loss"` //本回合攻击方损失的兵力
	DLoss int `json:"d_loss"` //本回合防守方损失的兵力
}

func (b battle) to() []int {
	r := make([]int, 0)
	r = append(r, b.AId)
	r = append(r, b.DId)
	r = append(r, b.ALoss)
	r = append(r, b.DLoss)
	return r
}


var WarService = &warService{}
type warService struct {

}

func (w *warService) GetWarReports(rid int) ([]model.WarReport,error)  {
	mrs := make([]data.WarReport,0)
	mr := &data.WarReport{}
	err := db.Engine.Table(mr).
		Where("a_rid=? or d_rid=?",rid,rid).
		Limit(30,0).
		Desc("ctime").
		Find(&mrs)
	if err != nil {
		log.Println("战报查询出错",err)
		return nil, common.New(constant.DBError,"战报查询出错")
	}
	modelMrs := make([]model.WarReport,0)
	for _,v := range mrs{
		modelMrs = append(modelMrs,v.ToModel().(model.WarReport))
	}
	return modelMrs,nil
}

func IsWarFree(x,y int) bool{
	//判断是否为建筑
	rb,ok := RoleBuildService.PositionBuild(x,y)
	if ok {
		return rb.IsWarFree()
	}else{
		rc,ok := RoleCityService.PositionCity(x,y)
		if ok {
			//多加一个判断 城池 一个联盟的就不能攻击
			//一个联盟 本联盟 已经沦陷的
			rr := RoleAttrService.Get(rc.RId)
			if rr != nil {
				if rr.ParentId > 0 {
					return rc.IsWarFree()
				}
			}

		}
	}
	return false
}

func IsCanDefend(x,y,rid int) bool{
	unionId := data.GetUnion(rid)
	rb,ok := RoleBuildService.PositionBuild(x,y)
	if ok {
		toUnionId := data.GetUnion(rb.RId)
		parentId := data.GetParentId(rb.RId)
		if rb.RId == rid {
			return true
		}
		if unionId == toUnionId || unionId == parentId {
			return true
		}
	}
	rc,ok := RoleCityService.PositionCity(x,y)
	if ok {
		toUnionId := data.GetUnion(rc.RId)
		parentId := data.GetParentId(rc.RId)
		if rc.RId == rid {
			return true
		}
		if unionId == toUnionId || unionId == parentId {
			return true
		}
	}
	return false
}

func NewEmptyWar(attack *data.Army) *data.WarReport{
	//战报处理
	pArmy := attack.ToModel().(model.Army)
	begArmy, _ := json.Marshal(pArmy)

	//武将战斗前
	begGeneral := make([][]int, 0)
	for _, g := range attack.Gens {
		if g != nil {
			pg := g.ToModel().(model.General)
			begGeneral = append(begGeneral, pg.ToArray())
		}
	}
	begGeneralData, _ := json.Marshal(begGeneral)

	wr := &data.WarReport{X: attack.ToX, Y: attack.ToY, AttackRid: attack.RId,
		AttackIsRead: false, DefenseIsRead: true, DefenseRid: 0,
		BegAttackArmy: string(begArmy), BegDefenseArmy: "",
		EndAttackArmy: string(begArmy), EndDefenseArmy: "",
		BegAttackGeneral: string(begGeneralData),
		EndAttackGeneral: string(begGeneralData),
		BegDefenseGeneral: "",
		EndDefenseGeneral: "",
		Rounds: "",
		Result: 0,
		CTime: time.Now(),
	}
	return wr
}
package logic

import (
	"log"
	"mssgserver/db"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/model/data"
	"time"
)

var RoleResService = &roleResService{
	rolesRes: make(map[int]*data.RoleRes),
}
type roleResService struct {
	rolesRes map[int]*data.RoleRes
}

func (r *roleResService) Load()  {
	rr := make([]*data.RoleRes, 0)
	err := db.Engine.Find(&rr)
	if err != nil {
		log.Println(" load role_res table error")
	}

	for _, v := range rr {
		r.rolesRes[v.RId] = v
	}

	go r.produce()
}
func (r *roleResService) GetRoleRes(rid int) *data.RoleRes {
	roleRes := &data.RoleRes{}
	ok ,err := db.Engine.Table(roleRes).Where("rid=?",rid).Get(roleRes)
	if err != nil {
		log.Println("查询角色资源出错",err)
		return nil
	}
	if ok {
		return roleRes
	}
	return nil
}

func (r *roleResService) GetYield(rid int) data.Yield {
	//基础产量 + 城池设施的产量 + 建筑产量
	rbYield := RoleBuildService.GetYield(rid)
	cfYield := CityFacilityService.GetYield(rid)
	var yield data.Yield
	yield.Gold = rbYield.Gold + cfYield.Gold + gameConfig.Base.Role.GoldYield
	yield.Stone = rbYield.Stone + cfYield.Stone + gameConfig.Base.Role.StoneYield
	yield.Iron = rbYield.Iron + cfYield.Iron + gameConfig.Base.Role.IronYield
	yield.Grain = rbYield.Grain + cfYield.Grain + gameConfig.Base.Role.GrainYield
	yield.Wood = rbYield.Wood + cfYield.Wood +  gameConfig.Base.Role.WoodYield
	return yield
}

func (r *roleResService) IsEnoughGold(rid int, cost int) bool {
	rr := r.GetRoleRes(rid)
	return rr.Gold >= cost
}

func (r *roleResService) CostGold(rid int, cost int) {
	rr := r.GetRoleRes(rid)
	if rr.Gold >= cost {
		rr.Gold -= cost
		rr.SyncExecute()
	}
}

func (r *roleResService) TryUseNeed(rid int, need gameConfig.NeedRes) bool {
	rr := r.GetRoleRes(rid)
	if need.Decree <= rr.Decree && need.Grain <= rr.Grain &&
		need.Stone <= rr.Stone && need.Wood <= rr.Wood &&
		need.Iron <= rr.Iron && need.Gold <= rr.Gold {
		rr.Decree -= need.Decree
		rr.Iron -= need.Iron
		rr.Wood -= need.Wood
		rr.Stone -= need.Stone
		rr.Grain -= need.Grain
		rr.Gold -= need.Gold

		rr.SyncExecute()
		return true
	}else{

		return false
	}
}

func (r *roleResService) produce() {
	for  {
		//一直去获取产量 隔一段时间获取一次
		recoveryTime := gameConfig.Base.Role.RecoveryTime
		time.Sleep(time.Duration(recoveryTime) * time.Hour)
		var index int
		for _,v := range r.rolesRes{
			capacity := GetDepotCapacity(v.RId)
			yield := r.GetYield(v.RId)
			if v.Wood < capacity {
				//根据实际的游戏策划需求来
				v.Wood += yield.Wood/6
			}
			if v.Stone < capacity {
				//根据实际的游戏策划需求来
				v.Stone += yield.Stone/6
			}
			if v.Iron < capacity {
				//根据实际的游戏策划需求来
				v.Iron += yield.Iron/6
			}
			if v.Grain < capacity {
				//根据实际的游戏策划需求来
				v.Grain += yield.Grain/6
			}
			if index%6 == 0 {
				if v.Decree < gameConfig.Base.Role.Decree {
					v.Decree += 1
				}
			}
			v.SyncExecute()
		}
	}
}

func (r *roleResService) DecreeIsEnough(rid int, cost int) bool {
	rr, ok := r.rolesRes[rid]
	if ok {
		if rr.Decree >= cost {
			return true
		}else{
			return false
		}
	}else{
		return false
	}
}

func (r *roleResService) TryUseDecree(rid int, decree int)  bool{
	rr, ok := r.rolesRes[rid]

	if ok {
		if rr.Decree >= decree {
			rr.Decree -= decree
			rr.SyncExecute()
			return true
		}else{
			return false
		}
	}else{
		return false
	}
}

func GetDepotCapacity(rid int) int {
	return CityFacilityService.GetCapacity(rid) + gameConfig.Base.Role.DepotCapacity
}


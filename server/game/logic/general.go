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
	"sync"
	"time"
)

var GeneralService = &generalService{
	genByRole: make(map[int][]*data.General),
	genByGId: make(map[int]*data.General),
}
type generalService struct {
	mutex     sync.RWMutex
	genByRole map[int][]*data.General
	genByGId  map[int]*data.General
}

func (g *generalService) Load()  {
	err := db.Engine.Table(data.General{}).Where("state=?",
		data.GeneralNormal).Find(g.genByGId)

	if err != nil {
		log.Println(err)
		return
	}

	for _, v := range g.genByGId {
		if _, ok := g.genByRole[v.RId]; ok==false {
			g.genByRole[v.RId] = make([]*data.General, 0)
		}
		g.genByRole[v.RId] = append(g.genByRole[v.RId], v)
	}
	go g.updatePhysicalPower()
}

func (g *generalService) GetGenerals(rid int) ([]model.General,error)  {
	mrs := make([]*data.General,0)
	mr := &data.General{}
	err := db.Engine.Table(mr).Where("rid=?",rid).Find(&mrs)
	if err != nil {
		log.Println("武将查询出错",err)
		return nil, common.New(constant.DBError,"武将查询出错")
	}
	if len(mrs) <= 0 {
		//随机3个武将
		var count = 0
		for  {
			if count >= 3 {
				break
			}
			cfgId := general.General.Rand()
			gen,err := g.NewGeneral(cfgId,rid,1)
			if err != nil {
				log.Println(err)
				continue
			}
			mrs = append(mrs,gen)
			count++
		}
	}
	modelMrs := make([]model.General,0)
	for _,v := range mrs{
		modelMrs = append(modelMrs,v.ToModel().(model.General))
	}
	return modelMrs,nil
}

func (g *generalService) Draw(rid int,nums int) []model.General{
	mrs := make([]*data.General,0)
	for i := 0 ;i < nums;i++ {
		cfgId := general.General.Rand()
		gen,_ := g.NewGeneral(cfgId,rid,1)
		mrs = append(mrs,gen)
	}
	modelMrs := make([]model.General,0)
	for _,v := range mrs{
		modelMrs = append(modelMrs,v.ToModel().(model.General))
	}
	return modelMrs
}
const (
	GeneralNormal      	= 0 //正常
	GeneralComposeStar 	= 1 //星级合成
	GeneralConvert 		= 2 //转换
)

func (g *generalService) NewGeneral(cfgId int, rid int, level int8) (*data.General, error) {
	cfg := general.General.GMap[cfgId]
	//初始 武将 无技能 但是有三个技能槽
	sa := make([]*model.GSkill,3)
	ss,_ := json.Marshal(sa)
	gen := &data.General{
		PhysicalPower: gameConfig.Base.General.PhysicalPowerLimit,
		RId: rid,
		CfgId: cfg.CfgId,
		Order: 0,
		CityId: 0,
		Level: level,
		CreatedAt: time.Now(),
		CurArms: cfg.Arms[0],
		HasPrPoint: 0,
		UsePrPoint: 0,
		AttackDis: 0,
		ForceAdded: 0,
		StrategyAdded: 0,
		DefenseAdded: 0,
		SpeedAdded: 0,
		DestroyAdded: 0,
		Star: cfg.Star,
		StarLv: 0,
		ParentId: 0,
		SkillsArray: sa,
		Skills: string(ss),
		State: GeneralNormal,
	}

	_,err := db.Engine.Table(gen).Insert(gen)
	if err != nil {
		log.Println("GetGenerals插入",err)
		return nil,err
	}
	return gen,nil
}

func (g *generalService) Get(id int) (*data.General,bool) {
	mr := &data.General{}
	ok,err := db.Engine.Table(mr).Where("id=?",id).Get(mr)
	if err != nil {
		log.Println("武将查询出错",err)
		return nil,false
	}
	if ok {
		return mr,true
	}
	return nil,false
}

func (g *generalService) updatePhysicalPower() {
	limit := gameConfig.Base.General.PhysicalPowerLimit
	power := gameConfig.Base.General.RecoveryPhysicalPower

	for  {
		time.Sleep(time.Hour * 1)
		for _,v := range g.genByGId {
			if v.PhysicalPower < limit {
				v.PhysicalPower += power
				if v.PhysicalPower > limit {
					v.PhysicalPower = limit
				}
				v.SyncExecute()
			}
		}
	}
}

func (g *generalService) TryUsePhysicalPower(army *data.Army, power int) {
	for _,v := range army.Gens{
		if v == nil {
			continue
		}
		if v.PhysicalPower > power {
			v.PhysicalPower -= power
		}
		v.SyncExecute()
	}

}

func (g *generalService) GetDestroy(army *data.Army) int {
	destroy := 0
	for _,v := range army.Gens{
		if v == nil {
			continue
		}
		destroy += v.GetDestroy()
	}
	return destroy
}


func (g *generalService) GetByRid(rid int) ([]*data.General,bool)  {
	mrs := make([]*data.General,0)
	mr := &data.General{}
	err := db.Engine.Table(mr).Where("rid=?",rid).Find(&mrs)
	if err != nil {
		log.Println("武将查询出错",err)
		return nil, false
	}
	return mrs,true
}

//获取npc武将
func (gen *generalService) GetNPCGenerals(cnt int, star int8, level int8) ([]data.General, bool) {
	//获取系统的武将
	gs, ok := gen.GetByRid(0)
	if ok == false {
		return make([]data.General, 0), false
	}else{
		target := make([]data.General, 0)
		for _, g := range gs {
			if g.Level == level && g.Star == star{
				target = append(target, *g)
			}
		}

		if len(target) < cnt{
			return make([]data.General, 0), false
		}else{
			m := make(map[int]int)
			for true {
				r := rand.Intn(len(target))
				m[r] = r
				if len(m) == cnt{
					break
				}
			}

			rgs := make([]data.General, 0)
			for _, v := range m {
				t := target[v]
				rgs = append(rgs, t)
			}
			return rgs, true
		}
	}
}

package logic

import (
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/global"
	"mssgserver/server/game/model/data"
	"sync"
)

type sysArmyService struct {
	mutex 		sync.Mutex
	sysArmys    map[int][]*data.Army //key:posId 系统建筑军队
}

func (s *sysArmyService) getArmyCfg(x, y int) (star int8, lv int8, soilders int) {
	defender := 1
	star = 3
	lv = 5
	soilders = 100

	if mapBuild, ok := gameConfig.MapRes.PositionBuild(x, y); ok{
		cfg := gameConfig.MapBuildConf.BuildConfig(mapBuild.Type, mapBuild.Level)
		if cfg != nil {
			defender = cfg.Defender
			if npc, ok := gameConfig.Base.GetNPC(cfg.Level); ok {
				soilders = npc.Soilders
			}
		}
	}

	if defender == 1{
		star = 3
		lv = 5
	}else if defender == 2{
		star = 4
		lv = 10
	}else {
		star = 5
		lv = 20
	}

	return star, lv, soilders
}

func (s *sysArmyService) GetArmy(x int, y int) []*data.Army {
	posId := global.ToPosition(x, y)
	s.mutex.Lock()
	a, ok := s.sysArmys[posId]
	s.mutex.Unlock()
	if ok {
		return a
	}else{
		armys := make([]*data.Army, 0)

		star, lv, soilders := s.getArmyCfg(x, y)
		out, ok := GeneralService.GetNPCGenerals(3, star, lv)
		if ok {
			gsId := make([]int, 0)
			gs := make([]*data.General, 3)

			for i := 0; i < len(out) ; i++ {
				gs[i] = &out[i]
			}

			scnt := []int{soilders, soilders, soilders}
			army := &data.Army{RId: 0, Order: 0, CityId: 0,
				GeneralArray: gsId, Gens: gs, SoldierArray: scnt}
			army.ToGeneral()
			army.ToSoldier()

			armys = append(armys, army)
			posId := global.ToPosition(x, y)
			s.sysArmys[posId] = armys
		}

		return armys
	}
}

func (s *sysArmyService) DelArmy(x int, y int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	posId := global.ToPosition(x, y)
	delete(s.sysArmys, posId)
}

func NewSysArmy() *sysArmyService {
	return &sysArmyService{
		sysArmys: make(map[int][]*data.Army),
	}
}

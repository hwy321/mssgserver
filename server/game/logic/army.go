package logic

import (
	"encoding/json"
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/gameConfig/general"
	"mssgserver/server/game/global"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"mssgserver/utils"
	"sync"
	"time"
)

var ArmyService = &armyService{
	updateArmyChan: make(chan *data.Army,100),
	arriveArmyChan: make(chan *data.Army,100),
	giveUpIdChan: make(chan int,100),
	endTimeArmys: make(map[int64][]*data.Army),
	passByPosArmys: make(map[int]map[int]*data.Army),
	stopInPosArmys: make(map[int]map[int]*data.Army),
	sys: NewSysArmy(),
}
type armyService struct {
	passBy  		sync.RWMutex
	updateArmyChan chan *data.Army
	arriveArmyChan  chan *data.Army

	giveUpIdChan chan int
	//驻守的军队 key posId rid 军队
	stopInPosArmys map[int]map[int]*data.Army
	//缓存到达时间 军队
	endTimeArmys map[int64][]*data.Army
	passByPosArmys 	map[int]map[int]*data.Army //玩家路过位置的军队 key:posId,armyId
	sys *sysArmyService
}

func (a *armyService) Update(army *data.Army)  {
	a.updateArmyChan <- army
}
func (a *armyService) Init()  {
	//初始化
	go a.check()
	go a.running()
}

func (a *armyService) check() {
	for  {
		time.Sleep(time.Millisecond * 100)
		armysMap := a.endTimeArmys
		cur := time.Now().Unix()
		for endTime, armys := range armysMap{
			if cur >= endTime {
				a.Arrive(armys)
				delete(armysMap,endTime)
			}
		}
	}

}
func (a *armyService) running() {
	for  {
		select {
		case army := <- a.arriveArmyChan:
			a.exeArrive(army)
		case army := <- a.updateArmyChan:
			a.exeUpdate(army)
		case posId := <- a.giveUpIdChan:
			a.GiveUpPosId(posId)
		}
	}
}

func (r *armyService) GetArmys(rid int) ([]model.Army,error)  {
	mrs := make([]data.Army,0)
	mr := &data.Army{}
	err := db.Engine.Table(mr).Where("rid=?",rid).Find(&mrs)
	if err != nil {
		log.Println("军队查询出错",err)
		return nil, common.New(constant.DBError,"军队查询出错")
	}
	modelMrs := make([]model.Army,0)
	for _,v := range mrs{
		modelMrs = append(modelMrs,v.ToModel().(model.Army))
	}
	return modelMrs,nil
}

func (r *armyService) GetArmysByCity(rid ,cId int) ([]model.Army,error)  {
	mrs := make([]data.Army,0)
	mr := &data.Army{}
	err := db.Engine.Table(mr).Where("rid=? and cityId=?",rid,cId).Find(&mrs)
	if err != nil {
		log.Println("军队查询出错",err)
		return nil, common.New(constant.DBError,"军队查询出错")
	}
	modelMrs := make([]model.Army,0)
	for _,v := range mrs{
		modelMrs = append(modelMrs,v.ToModel().(model.Army))
	}
	return modelMrs,nil
}

func (a *armyService) ScanBlock(roleId int,req *model.ScanBlockReq) ([]model.Army, error) {
	x := req.X
	y := req.Y
	length := req.Length
	out := make([]model.Army, 0)
	if x < 0 || x >= global.MapWith || y < 0 || y >= global.MapHeight {
		return out,nil
	}

	maxX := utils.MinInt(global.MapWith, x+length-1)
	maxY := utils.MinInt(global.MapHeight, y+length-1)

	a.passBy.RLock()
	defer a.passBy.RUnlock()
	for i := x; i <= maxX; i++ {
		for j := y; j <= maxY; j++ {
			posId := global.ToPosition(i, j)
			armys, ok := a.passByPosArmys[posId]
			if ok {
				//是否在视野范围内
				is := armyIsInView(roleId, i, j)
				if is == false{
					continue
				}
				for _, army := range armys {
					out = append(out, army.ToModel().(model.Army))
				}
			}
		}
	}
	return out,nil
}

func (a *armyService) updateGenerals(armys... *data.Army) {
	for _, army := range armys {
		army.Gens = make([]*data.General, 0)
		for _, gid := range army.GeneralArray {
			if gid == 0{
				army.Gens = append(army.Gens, nil)
			}else{
				g, _ := GeneralService.Get(gid)
				army.Gens = append(army.Gens, g)
			}
		}
	}
}


func (a *armyService) GetArmyByCityAndOrder(rid ,cId int,order int8) (*data.Army,bool)  {
	army := &data.Army{}
	ok,err := db.Engine.Table(army).Where("rid=? and cityId=? and a_order=?",rid,cId,order).Get(army)
	if err != nil {
		log.Println("军队查询出错",err)
		return nil, false
	}
	if ok {
		army.CheckConscript()
		a.updateGenerals(army)
		return army,true
	}
	return nil, false
}

func (a *armyService) GetCreate(cid int, rid int, order int8) (*data.Army, bool) {
	//根据城池id 角色id，order 进行查询
	//有 返回 没有创建并返回
	army,ok := a.GetArmyByCityAndOrder(rid,cid,order)
	if ok {
		return army,true
	}
	//需要创建
	army = &data.Army{
		RId: rid,
		Order: order,
		CityId: cid,
		Generals: `[0,0,0]`,
		Soldiers: `[0,0,0]`,
		GeneralArray: []int{0,0,0},
		SoldierArray: []int{0,0,0},
		ConscriptCnts: `[0,0,0]`,
		ConscriptTimes: `[0,0,0]`,
		ConscriptCntArray: []int{0,0,0},
		ConscriptTimeArray: []int64{0,0,0},
	}
	a.updateGenerals(army)
	_,err := db.Engine.Table(army).Insert(army)
	if err != nil {
		log.Println("armyService GetCreate err",err)
		return nil,false
	}
	return army,true
}


func (a *armyService) GetDbArmys(rid int) ([]*data.Army,error)  {
	mrs := make([]*data.Army,0)
	mr := &data.Army{}
	err := db.Engine.Table(mr).Where("rid=?",rid).Find(&mrs)
	if err != nil {
		log.Println("军队查询出错",err)
		return nil, common.New(constant.DBError,"军队查询出错")
	}
	for _,v := range mrs {
		v.CheckConscript()
		a.updateGenerals(v)
	}
	return mrs,nil
}

func (r *armyService) IsRepeat(rid int, cfgId int) bool {
	armys,err := r.GetDbArmys(rid)
	if err != nil {
		return true
	}
	for _ ,v := range armys{
		for _,gId := range v.GeneralArray{
			if gId == cfgId {
				return true
			}
		}
	}
	return false
}

func (a *armyService) Get(id int) *data.Army {
	army := &data.Army{}
	ok,err := db.Engine.Table(army).Where("id=?",id).Get(army)
	if err != nil {
		log.Println("军队查询出错",err)
		return nil
	}
	if ok {
		//还需要做一步操作  检测一下是否征兵完成
		army.CheckConscript()
		a.updateGenerals(army)
		return army
	}
	return nil
}

func (a *armyService) GetArmy(cid int, order int8) *data.Army {
	army := &data.Army{}
	ok,err := db.Engine.Table(army).Where("cityId=? and a_order=?",cid,order).Get(army)
	if err != nil {
		log.Println("armyService GetArmy err",err)
		return nil
	}
	if ok {
		//还需要做一步操作  检测一下是否征兵完成
		army.CheckConscript()
		a.updateGenerals(army)
		return army
	}
	return nil
}

func (a *armyService) GetArmyByCid(cid int) []*data.Army {
	armys := make([]*data.Army,0)
	army := &data.Army{}
	err := db.Engine.Table(army).Where("cityId=?",cid).Find(&armys)
	if err != nil {
		log.Println("armyService GetArmy err",err)
		return nil
	}
	for _,ar :=range armys{
		//还需要做一步操作  检测一下是否征兵完成
		ar.CheckConscript()
		a.updateGenerals(ar)
	}
	return armys
}

func (r *armyService) PushAction(army *data.Army) {
	if army.Cmd == data.ArmyCmdAttack ||
		army.Cmd == data.ArmyCmdDefend ||
		army.Cmd == data.ArmyCmdTransfer {
		end := army.End
		_,ok := r.endTimeArmys[end.Unix()]
		if !ok {
			r.endTimeArmys[end.Unix()] = make([]*data.Army,0)
		}
		r.endTimeArmys[end.Unix()] = append(r.endTimeArmys[end.Unix()],army)

	}else if army.Cmd == data.ArmyCmdBack {
		end := army.End
		_,ok := r.endTimeArmys[end.Unix()]
		if !ok {
			r.endTimeArmys[end.Unix()] = make([]*data.Army,0)
		}
		r.endTimeArmys[end.Unix()] = append(r.endTimeArmys[end.Unix()],army)
		//army.Start = time.Now()
	}

}

func (a *armyService) Arrive(armys []*data.Army) {
	for _,army := range armys {
		a.arriveArmyChan <- army
	}
}

func (a *armyService) exeArrive(army *data.Army) {
	//开启一个战争
	if army.Cmd == data.ArmyCmdAttack {
		if !IsWarFree(army.ToX, army.ToY) &&
			!IsCanDefend(army.ToX,army.ToY,army.RId){
			a.newBattle(army)
		}else{
			wr := NewEmptyWar(army)
			wr.SyncExecute()
		}
		army.State = data.ArmyStop
		a.Update(army)
	}else if army.Cmd == data.ArmyCmdBack {
		//回城成功
		army.ToX = army.FromX
		army.ToY = army.ToY
		army.State = data.ArmyStop
		army.Cmd = data.ArmyCmdIdle
		a.Update(army)
	}else if army.Cmd == data.ArmyCmdDefend {
		//呆在哪里不动
		ok := IsCanDefend(army.ToX, army.ToY, army.RId)
		if ok {
			//目前是自己的领地才能驻守
			army.State = data.ArmyStop
			a.addStopArmy(army)
			a.Update(army)
		}else{
			war := NewEmptyWar(army)
			war.SyncExecute()
			a.ArmyBack(army)
		}
	}else if army.Cmd == data.ArmyCmdTransfer {
		//调动到位置了
		if army.State == data.ArmyRunning{
			ok := RoleBuildService.BuildIsRId(army.ToX, army.ToY, army.RId)
			if ok == false{
				a.ArmyBack(army)
			}else{
				b, _ := RoleBuildService.PositionBuild(army.ToX, army.ToY)
				if b.IsHasTransferAuth(){
					army.State = data.ArmyStop
					army.Cmd = data.ArmyCmdIdle
					x := army.ToX
					y := army.ToY
					army.FromX = x
					army.FromY = y
					army.ToX = x
					army.ToY = y
					a.addStopArmy(army)
					a.Update(army)
				}else{
					a.ArmyBack(army)
				}
			}
		}
	}

}

func (a *armyService) newBattle(attackArmy *data.Army) {
	//分为两部分
	//1.城池攻打处理  2. 建筑攻打处理
	city, ok := RoleCityService.PositionCity(attackArmy.ToX,attackArmy.ToY)
	if ok {
		//处理城池攻打
		//查询城池是否有空闲的军队
		posId := global.ToPosition(attackArmy.ToX,attackArmy.ToY)
		enemys := a.GetStopArmys(posId)
		//城池内部 空闲的部队
		armys := a.GetArmyByCid(city.CityId)
		for _,v :=range armys{
			enemys = append(enemys,v)
		}
		if enemys == nil || len(enemys) == 0 {
			//无军队 直接攻打城池 扣减耐久值
			destroy := GeneralService.GetDestroy(attackArmy)
			city.DurableChange(-destroy)
			city.SyncExecute()
			//生成战报 空的战斗
			wr := NewEmptyWar(attackArmy)
			wr.Result = 2
			wr.DefenseRid = city.RId
			wr.DefenseIsRead = false
			//判断城池耐久 是否为0
			checkCityOccupy(wr, attackArmy, city)
			wr.SyncExecute()
		}else{
			//有军队的情况
			lastWar, warReports := trigger(attackArmy, enemys, true)
			if lastWar.Result > 1 {
				wr := warReports[len(warReports)-1]
				checkCityOccupy(wr, attackArmy, city)
			}
			for _, wr := range warReports {
				wr.SyncExecute()
			}
		}

	}else{
		a.executeBuild(attackArmy)
	}


}

func trigger(army *data.Army, enemys []*data.Army, isRoleEnemy bool) (*WarResult, []*data.WarReport) {

	posId := global.ToPosition(army.ToX, army.ToY)
	warReports := make([]*data.WarReport, 0)
	var lastWar *WarResult = nil
	for _, enemy := range enemys {
		//战报处理
		pArmy := army.ToModel().(model.Army)
		pEnemy := enemy.ToModel().(model.Army)

		begArmy1, _ := json.Marshal(pArmy)
		begArmy2, _ := json.Marshal(pEnemy)

		//武将战斗前
		begGeneral1 := make([][]int, 0)
		for _, g := range army.Gens {
			if g != nil {
				pg := g.ToModel().(model.General)
				begGeneral1 = append(begGeneral1, pg.ToArray())
			}
		}
		begGeneralData1, _ := json.Marshal(begGeneral1)

		begGeneral2 := make([][]int, 0)
		for _, g := range enemy.Gens {
			if g != nil {
				pg := g.ToModel().(model.General)
				begGeneral2 = append(begGeneral2, pg.ToArray())
			}
		}
		begGeneralData2, _ := json.Marshal(begGeneral2)
		//发生战斗
		lastWar = newWar(army,enemy)

		//武将战斗后
		endGeneral1 := make([][]int, 0)
		for _, g := range army.Gens {
			if g != nil {
				pg := g.ToModel().(model.General)
				endGeneral1 = append(endGeneral1, pg.ToArray())
				level, exp := general.GeneralBasic.ExpToLevel(g.Exp)
				g.Level = level
				g.Exp = exp
				g.SyncExecute()
			}
		}
		endGeneralData1, _ := json.Marshal(endGeneral1)

		endGeneral2 := make([][]int, 0)
		for _, g := range enemy.Gens {
			if g != nil {
				pg := g.ToModel().(model.General)
				endGeneral2 = append(endGeneral2, pg.ToArray())
				level, exp := general.GeneralBasic.ExpToLevel(g.Exp)
				g.Level = level
				g.Exp = exp
				g.SyncExecute()
			}
		}
		endGeneralData2, _ := json.Marshal(endGeneral2)

		pArmy = army.ToModel().(model.Army)
		pEnemy = enemy.ToModel().(model.Army)
		endArmy1, _ := json.Marshal(pArmy)
		endArmy2, _ := json.Marshal(pEnemy)

		rounds, _ := json.Marshal(lastWar.Round)

		wr := &data.WarReport{X: army.ToX, Y: army.ToY, AttackRid: army.RId,
			AttackIsRead: false, DefenseIsRead: false, DefenseRid: enemy.RId,
			BegAttackArmy: string(begArmy1), BegDefenseArmy: string(begArmy2),
			EndAttackArmy: string(endArmy1), EndDefenseArmy: string(endArmy2),
			BegAttackGeneral: string(begGeneralData1),
			BegDefenseGeneral: string(begGeneralData2),
			EndAttackGeneral: string(endGeneralData1),
			EndDefenseGeneral: string(endGeneralData2),
			Rounds: string(rounds),
			Result: lastWar.Result,
			CTime: time.Now(),
		}

		warReports = append(warReports, wr)

		enemy.ToSoldier()
		enemy.ToGeneral()
		//是否有玩家的军队
		if isRoleEnemy {
			if lastWar.Result > 1 {
				if isRoleEnemy {
					ArmyService.deleteStopArmy(posId)
				}
				//失败部队 走 返回城池 战斗完 返回原地
				ArmyService.ArmyBack(enemy)
			}
			enemy.SyncExecute()
		}else{
			wr.DefenseIsRead = true
		}
	}
	army.SyncExecute()
	return lastWar, warReports
}

func newWar(attack *data.Army, defense *data.Army) *WarResult {
	//
	w := ArmyWar{attack: attack, defense: defense}
	w.init()
	wars := w.battle()

	result := &WarResult{Round: wars}
	if w.attackPos[0].soldiers == 0{
		result.Result = 0
	}else if w.defensePos[0] != nil && w.defensePos[0].soldiers != 0{
		result.Result = 1
	}else{
		result.Result = 2
	}

	return result
}

func checkCityOccupy(wr *data.WarReport, army *data.Army, city *data.MapRoleCity) {
	if city.CurDurable <= 0 {
		//占领 有联盟才能俘虏玩家
		roleAttribute := RoleAttrService.Get(army.RId)
		if roleAttribute != nil{
			unionId := roleAttribute.UnionId
			if unionId > 0 {
				//攻占了
				wr.Occupy = 1
				rr := RoleAttrService.Get(city.RId)
				if rr != nil {
					rr.ParentId = unionId
					rr.SyncExecute()
				}
				city.OccupyTime = time.Now()
			}else{
				wr.Occupy = 0
			}
		}else{
			wr.Occupy = 0
		}
	}else{
		wr.Occupy = 0
	}
	city.SyncExecute()
}

func (a *armyService) executeBuild(army *data.Army) {
	//如果不是城池，就是攻击建筑
	//建筑生成对应的NPC，我们是和NPC打仗了
	roleBuild, _ := RoleBuildService.PositionBuild(army.ToX, army.ToY)

	posId := global.ToPosition(army.ToX, army.ToY)
	posArmys := ArmyService.GetStopArmys(posId)
	//判断是否为玩家的领地
	isRoleEnemy := len(posArmys) != 0
	var enemys []*data.Army
	if !isRoleEnemy {
		//生成NPC的部队
		enemys = ArmyService.sys.GetArmy(army.ToX, army.ToY)
	}else{
		for _,v := range posArmys{
			enemys = append(posArmys,v)
		}
	}
	lastWar, warReports := trigger(army, enemys, isRoleEnemy)
	if lastWar.Result > 1 {
		if roleBuild != nil {
			//计算破坏力
			destory := GeneralService.GetDestroy(army)
			wr := warReports[len(warReports)-1]
			wr.DestroyDurable = utils.MinInt(destory, roleBuild.CurDurable)
			roleBuild.CurDurable = utils.MaxInt(0, roleBuild.CurDurable - destory)
			if roleBuild.CurDurable == 0{
				//攻打成功
				//是不是超过了玩家应该能占领的土地上限
				bLimit := gameConfig.Base.Role.BuildLimit
				if bLimit > RoleBuildService.BuildCnt(army.RId) {
					//
					wr.Occupy = 1
					RoleBuildService.RemoveFromRole(roleBuild)
					RoleBuildService.AddBuild(army.RId, army.ToX, army.ToY)
					OccupyRoleBuild(army.RId, army.ToX, army.ToY)
				}else{
					wr.Occupy = 0
				}
			}else{
				wr.Occupy = 0
			}
		}else{
			//系统的领地
			//占领完 之后 生成新的一条记录
			wr := warReports[len(warReports)-1]
			blimit := gameConfig.Base.Role.BuildLimit
			if blimit > RoleBuildService.BuildCnt(army.RId){
				OccupySystemBuild(army.RId, army.ToX, army.ToY)
				wr.DestroyDurable = 10000
				wr.Occupy = 1
			}else{
				wr.Occupy = 0
			}
			ArmyService.sys.DelArmy(army.ToX, army.ToY)
		}
	}

	for _, wr := range warReports {
		wr.SyncExecute()
	}
}

func OccupySystemBuild(rid int, x int, y int) {
	if _, ok := RoleBuildService.PositionBuild(x, y); ok {
		return
	}

	if gameConfig.MapRes.IsCanBuild(x, y){
		rb, ok := RoleBuildService.AddBuild(rid, x, y)
		if ok {
			rb.OccupyTime = time.Now()
			rb.SyncExecute()
		}
	}
}

func OccupyRoleBuild(rid int, x int, y int) {
	if b, ok := RoleBuildService.PositionBuild(x, y); ok {
		b.CurDurable = b.MaxDurable
		b.OccupyTime = time.Now()
		b.RId = rid
		b.SyncExecute()
	}
}

func (a *armyService) GetStopArmys(posId int) []*data.Army {
	armysMap ,ok := a.stopInPosArmys[posId]
	if ok {
		armys := make([]*data.Army,0)
		for _,v := range armysMap{
			armys = append(armys,v)
		}
		return armys
	}
	return nil
}

func (a *armyService) deleteStopArmy(posId int) {
	delete(a.stopInPosArmys,posId)
}

func (a *armyService) ArmyBack(army *data.Army) {

	army.State = data.ArmyRunning
	army.Cmd  = data.ArmyCmdBack

	//清除掉之前的
	t := army.End.Unix()
	if actions, ok := a.endTimeArmys[t]; ok {
		for i, v := range actions {
			if v.Id == army.Id{
				actions = append(actions[:i], actions[i+1:]...)
				a.endTimeArmys[t] = actions
				break
			}
		}
	}

	army.Start = time.Now()
	army.End = army.Start.Add(time.Second * 20)

	a.PushAction(army)
}

func (a *armyService) exeUpdate(army *data.Army) {
	army.SyncExecute()
	if army.Cmd == data.ArmyCmdBack {
		posId := global.ToPosition(army.ToX, army.ToY)
		armys, ok := a.stopInPosArmys[posId]
		if ok {
			delete(armys, army.Id)
			a.stopInPosArmys[posId] = armys
		}
	}
}

func (a *armyService) GiveUp(posId int) {
	a.giveUpIdChan <- posId
}

func (a *armyService) GiveUpPosId(posId int) {
	armys ,ok := a.stopInPosArmys[posId]
	if ok {
		for _,army := range armys{
			a.ArmyBack(army)
		}
		delete(a.stopInPosArmys,posId)
	}
}

func (a *armyService) addStopArmy(army *data.Army) {
	posId := global.ToPosition(army.ToX, army.ToY)

	if _, ok := a.stopInPosArmys[posId]; ok == false {
		a.stopInPosArmys[posId] = make(map[int]*data.Army)
	}
	a.stopInPosArmys[posId][army.Id] = army
}
//归属于该位置的军队数量
func (a *armyService) BelongPosArmyCnt(rid int, x, y int) int {
	cnt := 0
	armys, err := a.GetArmys(rid)
	if err == nil {
		for _, army := range armys {
			if army.FromX == x && army.FromY == y{
				cnt += 1
			}else if army.Cmd == data.ArmyCmdTransfer && army.ToX == x && army.ToY == y {
				cnt += 1
			}
		}
	}

	return cnt
}

func armyIsInView(rid, x, y int) bool {
	//简单点 先设为true
	return true
}

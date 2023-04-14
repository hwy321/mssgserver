package logic

import (
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/global"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"mssgserver/utils"
	"sync"
	"time"
)

var RoleBuildService = &roleBuildService{
	posRB: make(map[int]*data.MapRoleBuild),
	roleRB: make(map[int][]*data.MapRoleBuild),
	giveUpRB: make(map[int64]map[int]*data.MapRoleBuild),
}
type roleBuildService struct {
	mutex sync.RWMutex
	giveUpMutex sync.RWMutex
	//位置 key posId
	posRB map[int]*data.MapRoleBuild
	//key 角色id
	roleRB map[int][]*data.MapRoleBuild
	//放弃的
	giveUpRB map[int64]map[int]*data.MapRoleBuild
}
func (r *roleBuildService) Load()  {
	//加载系统的建筑以及玩家的建筑
	//首先需要判断数据库 是否保存了系统的建筑 没有 进行一个保存
	total,err := db.Engine.
		Where("type=? or type=?",gameConfig.MapBuildSysCity,gameConfig.MapBuildSysFortress).
		Count(new(data.MapRoleBuild))
	if err != nil {
		panic(err)
	}
	if int64(len(gameConfig.MapRes.SysBuild)) != total {
		//证明数据库存储的系统建筑 有问题
		db.Engine.
			Where("type=? or type=?",gameConfig.MapBuildSysCity,gameConfig.MapBuildSysFortress).
			Delete(new(data.MapRoleBuild))
		for _, v := range gameConfig.MapRes.SysBuild{
			build := &data.MapRoleBuild{
				RId:   0,
				Type:  v.Type,
				Level: v.Level,
				X:     v.X,
				Y:     v.Y,
			}
			build.Init()
			db.Engine.InsertOne(build)
		}
	}
	//查询所有的角色建筑
	dbRB := make(map[int]*data.MapRoleBuild)
	db.Engine.Find(dbRB)

	for _,v := range dbRB{
		v.Init()
		posId := global.ToPosition(v.X,v.Y)
		r.posRB[posId] = v
		_,ok := r.roleRB[v.RId]
		if !ok {
			r.roleRB[v.RId] = make([]*data.MapRoleBuild,0)
		}
		r.roleRB[v.RId] = append(r.roleRB[v.RId],v)
	}

	for _,v := range dbRB{
		v.Init()
		if v.GiveUpTime > 0 {
			_ ,ok := r.giveUpRB[v.GiveUpTime]
			if !ok {
				r.giveUpRB[v.GiveUpTime] = make(map[int]*data.MapRoleBuild)
			}
			r.giveUpRB[v.GiveUpTime][v.Id] = v
		}
	}

	go r.checkGiveUp()
}

func (r *roleBuildService) GetBuilds(rid int) ([]model.MapRoleBuild,error)  {
	mrs := make([]data.MapRoleBuild,0)
	mr := &data.MapRoleBuild{}
	err := db.Engine.Table(mr).Where("rid=?",rid).Find(&mrs)
	if err != nil {
		log.Println("建筑查询出错",err)
		return nil, common.New(constant.DBError,"建筑查询出错")
	}
	modelMrs := make([]model.MapRoleBuild,0)
	for _,v := range mrs{
		modelMrs = append(modelMrs,v.ToModel().(model.MapRoleBuild))
	}
	return modelMrs,nil
}

func (r *roleBuildService) ScanBlock(req *model.ScanBlockReq) ([]model.MapRoleBuild, error) {
	x := req.X
	y := req.Y
	length := req.Length
	var mrbs = make([]model.MapRoleBuild,0)
	if x < 0 || x >= global.MapWith || y < 0 || y >= global.MapHeight {
		return mrbs,nil
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	maxX := utils.MinInt(global.MapWith, x+length-1)
	maxY := utils.MinInt(global.MapHeight, y+length-1)

	//范围  x-length  x + length  y-length y+length
	for i := x-length; i<=maxX;i++ {
		for j := y-length;j<=maxY;j++ {
			posId := global.ToPosition(i,j)
			mrb,ok := r.posRB[posId]
			if ok {
				mrbs = append(mrbs,mrb.ToModel().(model.MapRoleBuild))
			}
		}
	}

	return mrbs,nil
}

func (r *roleBuildService) GetYield(rid int) data.Yield {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	rbs,ok := r.roleRB[rid]
	var yield data.Yield
	if ok {
		for _,v := range rbs{
			yield.Stone += v.Stone
			yield.Wood += v.Wood
			yield.Iron += v.Iron
			yield.Grain += v.Grain
		}
	}
	return yield
}

func (r *roleBuildService) PositionBuild(x int, y int) (*data.MapRoleBuild,bool){
	posId := global.ToPosition(x,y)
	rb,ok := r.posRB[posId]
	return rb,ok
}

func (r *roleBuildService) BuildCnt(rid int) int {
	rbs , ok := r.roleRB[rid]
	if ok {
		return len(rbs)
	}
	return 0
}

func (r *roleBuildService) RemoveFromRole(build *data.MapRoleBuild) {
	rbs,ok := r.roleRB[build.RId]
	if ok {
		for i, v := range rbs {
			if v.Id == build.Id{
				r.roleRB[build.RId] = append(rbs[:i], rbs[i+1:]...)
				break
			}
		}
	}
	r.giveUpMutex.Lock()
	delete(r.giveUpRB,build.GiveUpTime)
	r.giveUpMutex.Unlock()

	build.Reset()
	build.SyncExecute()
}

func (r *roleBuildService) MapResTypeLevel(x int, y int) (bool, int8, int8) {
	posId := global.ToPosition(x,y)
	nm,ok := gameConfig.MapRes.Confs[posId]
	if ok {
		return true,nm.Type,nm.Level
	}
	return false,0,0
}

func (r *roleBuildService) AddBuild(rid int, x int, y int) (*data.MapRoleBuild,bool) {
	posId := global.ToPosition(x, y)
	rb, ok := r.posRB[posId]
	if ok {
		if r.roleRB[rid] == nil {
			r.roleRB[rid] = make([]*data.MapRoleBuild,0)
		}
		r.roleRB[rid] = append(r.roleRB[rid],rb)
		return rb,true
	}else {
		//数据库插入
		if b, ok :=gameConfig.MapRes.PositionBuild(x,y); ok{
			if cfg:= gameConfig.MapBuildConf.BuildConfig(b.Type, b.Level); cfg != nil {
				rb := &data.MapRoleBuild{
					RId: rid, X: x, Y: y,
					Type: b.Type, Level: b.Level, OPLevel: b.Level,
					Name: cfg.Name, CurDurable: cfg.Durable,
					MaxDurable: cfg.Durable,
				}
				rb.Init()

				if _, err := db.Engine.Table(model.MapRoleBuild{}).Insert(rb); err == nil{
					r.posRB[posId] = rb
					if _, ok := r.roleRB[rid]; ok == false{
						r.roleRB[rid] = make([]*data.MapRoleBuild, 0)
					}
					r.roleRB[rid] = append(r.roleRB[rid], rb)
					return rb, true
				}
			}
		}
	}
	return nil,false
}

func (r *roleBuildService) RoleFortressCnt(rid int) int {
	builds, err := r.GetBuilds(rid)
	if err != nil {
		return 0
	}
	var cnt = 0
	for _,v := range builds{
		if v.Type == gameConfig.MapBuildFortress {
			cnt += 1
		}
	}
	return cnt
}

func (r *roleBuildService) GiveUp(build *data.MapRoleBuild) int{

	if build.IsWarFree() {
		return constant.BuildWarFree
	}

	if build.GiveUpTime > 0{
		return constant.BuildGiveUpAlready
	}

	build.GiveUpTime = time.Now().Unix() + gameConfig.Base.Build.GiveUpTime

	build.SyncExecute()

	//需要放入内存当中
	r.giveUpMutex.Lock()
	defer r.giveUpMutex.Unlock()
	_,ok := r.giveUpRB[build.GiveUpTime]
	if !ok {
		r.giveUpRB[build.GiveUpTime] = make(map[int]*data.MapRoleBuild)
	}
	r.giveUpRB[build.GiveUpTime][build.Id] = build

	return constant.OK
}

func (r *roleBuildService) checkGiveUp() {
	for  {
		time.Sleep(time.Second * 2)
		cur := time.Now().Unix()
		var ret []int
		var builds []*data.MapRoleBuild
		for i := cur - 10;i<=cur;i++ {
			rbs,ok := r.giveUpRB[i]
			if ok {
				for _,v := range rbs{
					builds = append(builds,v)
					//当前的坐标 放弃土地后 当前土地上的部队需要返回
					ret = append(ret,global.ToPosition(v.X,v.Y))
				}
			}
		}
		for _,build := range builds{
			r.RemoveFromRole(build)
		}
		for _,posId := range ret{
			ArmyService.GiveUp(posId)
		}
	}
}

func (r *roleBuildService) BuildIsRId(x int, y int, rid int) bool {
	build, ok := r.PositionBuild(x, y)
	if ok {
		return build.RId == rid
	}
	return false
}

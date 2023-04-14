package logic

import (
	"log"
	"math/rand"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/global"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"mssgserver/utils"
	"sync"
	"time"
	"xorm.io/xorm"
)

var RoleCityService = &roleCityService{
	dbRB: make(map[int]*data.MapRoleCity),
	posRC: make(map[int]*data.MapRoleCity),
	roleRC: make(map[int][]*data.MapRoleCity),
}
type roleCityService struct {
	mutex sync.RWMutex
	dbRB map[int]*data.MapRoleCity
	//位置 key posId
	posRC map[int]*data.MapRoleCity
	//key 角色id
	roleRC map[int][]*data.MapRoleCity
}

func (r *roleCityService) Load(){
	//查询所有的角色建筑
	db.Engine.Find(r.dbRB)

	for _,v := range r.dbRB{
		posId := global.ToPosition(v.X,v.Y)
		r.posRC[posId] = v
		_,ok := r.roleRC[v.RId]
		if !ok {
			r.roleRC[v.RId] = make([]*data.MapRoleCity,0)
		}
		r.roleRC[v.RId] = append(r.roleRC[v.RId],v)
	}
}
func (r *roleCityService) InitCity(rid int, name string, req *net.WsMsgReq) error {
	roleCity := &data.MapRoleCity{}
	ok,err := db.Engine.Table(roleCity).Where("rid=?",rid).Get(roleCity)
	if err != nil {
		log.Println("查询角色城池出错",err)
		return common.New(constant.DBError,"数据库出错")
	}
	if ok {
		return 	nil
	}else{
		//初始化
		for{
			roleCity.X = rand.Intn(global.MapWith)
			roleCity.Y = rand.Intn(global.MapHeight)
			//这个城池 能不能在这个坐标点创建 需要判断 系统城池五格之内 不能有玩家的城池
			if r.IsCanBuild(roleCity.X,roleCity.Y) {
				roleCity.RId = rid
				roleCity.Name = name
				roleCity.CurDurable = gameConfig.Base.City.Durable
				roleCity.CreatedAt = time.Now()
				roleCity.IsMain = 1
				if session := req.Context.Get("dbSession");session != nil {
					_,err = session.(*xorm.Session).Table(roleCity).Insert(roleCity)
				}else{
					_,err = db.Engine.Table(roleCity).Insert(roleCity)
				}
				if err != nil {
					log.Println("插入角色城池出错",err)
					return common.New(constant.DBError,"数据库出错")
				}
				posId := global.ToPosition(roleCity.X,roleCity.Y)
				r.posRC[posId] = roleCity
				_,ok := r.roleRC[rid]
				if !ok {
					r.roleRC[rid] = make([]*data.MapRoleCity,0)
				}else{
					r.roleRC[rid] = append(r.roleRC[rid],roleCity)
				}
				r.dbRB[roleCity.CityId] = roleCity
				//初始化城池的设施
				if err := CityFacilityService.TryCreate(roleCity.CityId,rid,req);err != nil{
					log.Println("城池设施出错",err)
					return common.New(err.(*common.MyError).Code(),err.Error())
				}
				break
			}
		}

	}
	return nil
}

func (r *roleCityService) IsCanBuild(x int, y int) bool {
	confs := gameConfig.MapRes.Confs
	pIndex := global.ToPosition(x,y)
	_,ok := confs[pIndex]
	if !ok {
		return false
	}

	//城池 1范围内 不能超过边界
	if x + 1 >= global.MapWith || y+1 >= global.MapHeight  || y-1 < 0 || x-1<0{
		return false
	}
	sysBuild := gameConfig.MapRes.SysBuild
	//系统城池的5格内 不能创建玩家城池
	for _,v := range sysBuild{
		if v.Type == gameConfig.MapBuildSysCity {
			if x >= v.X-5 &&
				x <= v.X+5 &&
				y >= v.Y-5 &&
				y <= v.Y+5{
				return false
			}
		}
	}

	//玩家城池的5格内 也不能创建城池
	for i := x-5;i<=x+5;i++ {
		for j := y-5;j<=y+5;j++ {
			posId := global.ToPosition(i,j)
			_,ok := r.posRC[posId]
			if ok {
				return false
			}
		}
	}
	return true
}

func (r *roleCityService) GetRoleCitys(rid int) ([]model.MapRoleCity, error) {
	citys := make([]data.MapRoleCity,0)
	city := &data.MapRoleCity{}
	err := db.Engine.Table(city).Where("rid=?",rid).Find(&citys)
	modelCitys := make([]model.MapRoleCity,0)
	if err != nil {
		log.Println("查询角色城池出错",err)
		return modelCitys,err
	}
	for _,v := range citys {
		modelCitys = append(modelCitys,v.ToModel().(model.MapRoleCity))
	}
	return modelCitys,nil

}

func (r *roleCityService) ScanBlock(req *model.ScanBlockReq) ([]model.MapRoleCity, error) {
	x := req.X
	y := req.Y
	length := req.Length
	var mrcs = make([]model.MapRoleCity,0)
	if x < 0 || x >= global.MapWith || y < 0 || y >= global.MapHeight {
		return mrcs,nil
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	maxX := utils.MinInt(global.MapWith, x+length-1)
	maxY := utils.MinInt(global.MapHeight, y+length-1)

	//范围  x-length  x + length  y-length y+length
	for i := x-length; i<=maxX;i++ {
		for j := y-length;j<=maxY;j++ {
			posId := global.ToPosition(i,j)
			mrb,ok := r.posRC[posId]
			if ok {
				mrcs = append(mrcs,mrb.ToModel().(model.MapRoleCity))
			}
		}
	}

	return mrcs,nil
}

func (r *roleCityService) Get(cid int) (*data.MapRoleCity,bool){
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	rc,ok := r.dbRB[cid]
	if ok {
		return rc,true
	}
	return nil,false
}

func (r *roleCityService) GetMainCity(rid int) *data.MapRoleCity {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	rcs,ok := r.roleRC[rid]
	if ok {
		for _,v := range rcs{
			if v.IsMain == 1 {
				return v
			}
		}
	}
	return nil
}

func (r *roleCityService) GetCityCost(cid int) int8 {
	return CityFacilityService.GetCost(cid) + gameConfig.Base.City.Cost
}

func (r *roleCityService) PositionCity(x int, y int) (*data.MapRoleCity, bool) {
	posId := global.ToPosition(x,y)
	rc,ok := r.posRC[posId]
	return rc,ok
}

func (rc *roleCityService) GetByRId(rid int) ([]*data.MapRoleCity, bool) {
	rc.mutex.RLock()
	r, ok := rc.roleRC[rid]
	rc.mutex.RUnlock()
	return r, ok
}

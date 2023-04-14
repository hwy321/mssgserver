package logic

import (
	"encoding/json"
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/model/data"
	"time"
	"xorm.io/xorm"
)

var CityFacilityService = &cityFacilityService{}
type cityFacilityService struct {

}

func (c *cityFacilityService) TryCreate(cid, rid int, req *net.WsMsgReq) error {
	cf := &data.CityFacility{}
	ok,err := db.Engine.Table(cf).Where("cityId=?",cid).Get(cf)
	if err != nil {
		log.Println("查询城市设施出错",err)
		return common.New(constant.DBError,"数据库错误")
	}
	if ok {
		return nil
	}
	cf.RId = rid
	cf.CityId = cid
	list := gameConfig.FacilityConf.List
	facs := make([]data.Facility,len(list))
	for index,v := range list{
		fac := data.Facility{
			Name: v.Name,
			Type: v.Type,
			PrivateLevel: 0,
			UpTime: 0,
		}
		facs[index] = fac
	}
	dataJson,_ := json.Marshal(facs)
	cf.Facilities = string(dataJson)
	if session := req.Context.Get("dbSession");session != nil {
		_,err = session.(*xorm.Session).Table(cf).Insert(cf)
	}else{
		_,err = db.Engine.Table(cf).Insert(cf)
	}
	if err != nil {
		log.Println("插入城市设施出错",err)
		return common.New(constant.DBError,"数据库错误")
	}
	return nil
}


func (c *cityFacilityService) GetByRId(rid int) ([]*data.CityFacility, error){
	cf := make([]*data.CityFacility,0)
	err := db.Engine.Table(new(data.CityFacility)).Where("rid=?",rid).Find(&cf)
	if err != nil {
		log.Println(err)
		return cf,common.New(constant.DBError,"数据库错误")
	}
	return cf, nil
}

func (c *cityFacilityService) GetYield(rid int) data.Yield {
	//查询 把表中的设施 获取到
	//设施的不同类型 去设施配置中查询匹配，匹配到增加产量的设施 木头 金钱 产量的计算
	//设施的等级不同 产量也不同
	cfs ,err := c.GetByRId(rid)
	var y data.Yield
	if err == nil {
		for _,v :=range cfs {
			facilities := v.Facility()
			for _,fa := range facilities{
				//计算等级 资源的产出是不同的
				if fa.GetLevel() > 0 {
					values := gameConfig.FacilityConf.GetValues(fa.Type,fa.GetLevel())
					adds := gameConfig.FacilityConf.GetAdditions(fa.Type)
					for i,aType := range adds{
						if aType == gameConfig.TypeWood {
							y.Wood += values[i]
						}else if aType == gameConfig.TypeGrain {
							y.Grain += values[i]
						}else if aType == gameConfig.TypeIron {
							y.Iron += values[i]
						}else if aType == gameConfig.TypeStone {
							y.Stone += values[i]
						}else if aType == gameConfig.TypeTax {
							y.Gold += values[i]
						}
					}
				}

			}
		}
	}
	return y
}

func (c *cityFacilityService) GetFacility(rid int, cid int) []data.Facility {
	f := &data.CityFacility{}
	ok,err := db.Engine.Table(new(data.CityFacility)).Where("rid=? and cityId=?",rid,cid).Get(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	if ok {
		return f.Facility()
	}
	return nil
}

func (c *cityFacilityService) GetFacility1(rid int, cid int) []*data.Facility {
	f := &data.CityFacility{}
	ok,err := db.Engine.Table(new(data.CityFacility)).Where("rid=? and cityId=?",rid,cid).Get(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	if ok {
		return f.Facility1()
	}
	return nil
}

func (c *cityFacilityService) UpFacility(rid int, cid int, fType int8) (*data.Facility,error) {
	facs := c.GetFacility1(rid,cid)
	result := &data.Facility{}
	for _, fac := range facs{
		if fac.Type == fType {
			//找到对应的升级设施
			//判断是否能升级 首先此设施是否在升级中，资源是否够
			if !fac.CanUp() {
				return nil,common.New(constant.UpError,"不能升级")
			}
			//level max
			maxLevel := fac.GetMaxLevel(fType)
			if fac.GetLevel() >= int8(maxLevel) {
				return nil,common.New(constant.UpError,"不能升级")
			}
			//先判断资源使用多少 ，判断用户资源是否足够
			need := gameConfig.FacilityConf.Need(fType, fac.GetLevel()+1)
			ok := RoleResService.TryUseNeed(rid, need)
			if !ok {
				return nil,common.New(constant.ResNotEnough,"资源不足不能升级")
			}
			fac.UpTime = time.Now().Unix()
			result = fac
		}
	}
	jsonByte,_ := json.Marshal(facs)
	cfac := c.Get(rid,cid)
	cfac.Facilities = string(jsonByte)
	cfac.SyncExecute()
	return result,nil
}

func (c *cityFacilityService) Get(rid int, cid int) *data.CityFacility {
	f := &data.CityFacility{}
	ok,err := db.Engine.Table(new(data.CityFacility)).Where("rid=? and cityId=?",rid,cid).Get(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	if ok {
		return f
	}
	return nil
}


func (c *cityFacilityService) GetByCid(cid int) *data.CityFacility {
	f := &data.CityFacility{}
	ok,err := db.Engine.Table(new(data.CityFacility)).Where("cityId=?",cid).Get(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	if ok {
		return f
	}
	return nil
}



func (c *cityFacilityService) GetFacilityLevel(cid int, fType int8) int8 {
	cf := c.GetByCid(cid)
	if cf == nil {
		return 0
	}
	facs := cf.Facility1()
	for _,v := range facs{
		if v.Type == fType {
			return v.GetLevel()
		}
	}
	return 0
}

func (c *cityFacilityService) GetCost(cid int) int8 {
	//TypeCost
	cf := c.GetByCid(cid)
	facilities := cf.Facility()
	var cost int
	for _,fa := range facilities{
		//计算等级 资源的产出是不同的
		if fa.GetLevel() > 0 {
			values := gameConfig.FacilityConf.GetValues(fa.Type,fa.GetLevel())
			adds := gameConfig.FacilityConf.GetAdditions(fa.Type)
			for i,aType := range adds{
				if aType == gameConfig.TypeCost {
					cost += values[i]
				}
			}
		}
	}
	return int8(cost)
}

func (c *cityFacilityService) GetCapacity(rid int) int {
	cfs ,err := c.GetByRId(rid)
	var cap int
	if err == nil {
		for _,v :=range cfs {
			facilities := v.Facility()
			for _,fa := range facilities{
				//计算等级 资源的产出是不同的
				if fa.GetLevel() > 0 {
					values := gameConfig.FacilityConf.GetValues(fa.Type,fa.GetLevel())
					adds := gameConfig.FacilityConf.GetAdditions(fa.Type)
					for i,aType := range adds{
						if aType == gameConfig.TypeWarehouseLimit {
							cap += values[i]
						}
					}
				}

			}
		}
	}
	return cap
}

func (c *cityFacilityService) GetSoldier(cid int) int {
	cf := c.GetByCid(cid)
	facilities := cf.Facility()
	var total int
	for _,fa := range facilities{
		//计算等级 资源的产出是不同的
		if fa.GetLevel() > 0 {
			values := gameConfig.FacilityConf.GetValues(fa.Type,fa.GetLevel())
			adds := gameConfig.FacilityConf.GetAdditions(fa.Type)
			for i,aType := range adds{
				if aType == gameConfig.TypeSoldierLimit {
					log.Println("TypeSoldierLimit",values[i])
					total += values[i]
				}
			}
		}
	}
	return total
}

func (c *cityFacilityService) GetAdditions(cid int,additionType...int8) []int {
	cf := c.GetByCid(cid)
	ret := make([]int, len(additionType))
	if cf == nil{
		return ret
	}else{
		for i,at := range additionType{
			limit := 0
			for _, f := range cf.Facility() {
				if f.GetLevel() > 0{
					values := gameConfig.FacilityConf.GetValues(f.Type, f.GetLevel())
					additions := gameConfig.FacilityConf.GetAdditions(f.Type)
					for i, aType := range additions {
						if aType == at {
							limit += values[i]
						}
					}
				}
			}
			ret[i] = limit
		}
	}
	return ret
}

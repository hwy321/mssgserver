package data

import (
	"encoding/json"
	"log"
	"mssgserver/db"
	"mssgserver/server/game/gameConfig"
	"time"
)

type Facility struct {
	Name         string `json:"name"`
	PrivateLevel int8   `json:"level"` 		//等级，外部读的时候不能直接读，要用GetLevel
	Type         int8   `json:"type"`
	UpTime       int64  `json:"up_time"`	//升级的时间戳，0表示该等级已经升级完成了
}

func (f *Facility) GetLevel() int8 {
	if f.UpTime > 0{
		cur := time.Now().Unix()
		cost := gameConfig.FacilityConf.CostTime(f.Type, f.PrivateLevel+1)
		if cur >= f.UpTime + int64(cost){
			f.PrivateLevel +=1
			f.UpTime = 0
		}
	}
	return f.PrivateLevel
}

func (f *Facility) CanUp() bool {
	f.GetLevel()
	return f.UpTime == 0
}

func (f *Facility) GetMaxLevel(fType int8) int {
	return gameConfig.FacilityConf.MaxLevel(fType)
}

var CityFacDao = &cityFacDao{
	cfChan: make(chan *CityFacility),
}
type cityFacDao struct {
	cfChan chan *CityFacility
}

func (c *cityFacDao) running() {
	for  {
		select {
		case cf := <- c.cfChan:
			_,err := db.Engine.Table(new(CityFacility)).ID(cf.Id).Cols("facilities").Update(cf)
			if err != nil {
				log.Println("cityFacDao running error",err)
			}
		}
	}
}

func init()  {
	go CityFacDao.running()
}
type CityFacility struct {
	Id         int    `xorm:"id pk autoincr"`
	RId        int    `xorm:"rid"`
	CityId     int    `xorm:"cityId"`
	Facilities string `xorm:"facilities"`
}

func (c *CityFacility) TableName() string {
	return "city_facility"
}

func (c *CityFacility) Facility() []Facility {
	facilities := make([]Facility, 0)
	json.Unmarshal([]byte(c.Facilities), &facilities)
	return facilities
}

func (c *CityFacility) Facility1() []*Facility {
	facilities := make([]*Facility, 0)
	json.Unmarshal([]byte(c.Facilities), &facilities)
	return facilities
}

func (c *CityFacility) SyncExecute() {
	CityFacDao.cfChan <- c
}

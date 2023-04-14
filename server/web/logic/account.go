package logic

import (
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/server/common"
	"mssgserver/server/models"
	"mssgserver/server/web/model"
	"mssgserver/utils"
	"time"
)

var DefaultAccountLogic = &AccountLogic{}
type AccountLogic struct {

}

func (l AccountLogic) Register(rq *model.RegisterReq) error {
	username := rq.Username
	user := &models.User{}
	ok,err := db.Engine.Table(user).Where("username=?",username).Get(user)
	if err != nil {
		log.Println("注册查询失败",err)
		return common.New(constant.DBError,"数据库异常")
	}
	if ok {
		//有数据 提示用户已存在
		return common.New(constant.UserExist,"用户已存在")
	}else {
		user.Mtime = time.Now()
		user.Ctime = time.Now()
		user.Username = rq.Username
		user.Passcode = utils.RandSeq(6)
		user.Passwd = utils.Password(rq.Password,user.Passcode)
		user.Hardware = rq.Hardware
		_,err := db.Engine.Table(user).Insert(user)
		if err != nil {
			log.Println("注册插入失败",err)
			return common.New(constant.DBError,"数据库异常")
		}
		return nil
	}
}

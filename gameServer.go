package main

import (
	"mssgserver/config"
	"mssgserver/net"
	"mssgserver/server/game"
)

/**
  1. 登录完成了，创建角色（玩家）
  2. 需要根据用户 查询此用户拥有的角色，没有 创建角色
  3. 木材 铁 令牌 金钱  主城  武将等等的  这些数据 要不要初始化，已经玩过游戏，这些值是不是需要查询出来
  4. 地图 有关系，城池 资源土地 要塞 需要定义
  5. 资源，军队，城池，武将等等
 */
func main()  {
	host := config.File.MustValue("game_server","host","127.0.0.1")
	port := config.File.MustValue("game_server","port","8001")
	s := net.NewServer(host + ":" + port)
	s.NeedSecret(false)
	game.Init()
	s.Router(game.Router)
	s.Start()
}

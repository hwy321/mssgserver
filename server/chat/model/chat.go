package model

type LoginReq struct {
	RId      int    `json:"rid"`
	NickName string `json:"nickName"`
	Token    string `json:"token"`
}

type LoginRsp struct {
	RId      int    `json:"rid"`
	NickName string `json:"nickName"`
}


type JoinReq struct {
	Type	int8	`json:"type"`	//0世界聊天、1联盟聊天
	Id		int		`json:"id"`
}

type JoinRsp struct {
	Type	int8	`json:"type"`	//0世界聊天、1联盟聊天
	Id		int		`json:"id"`
}

type HistoryReq struct {
	Type	int8	`json:"type"`	//0世界聊天、1联盟聊天
}

type HistoryRsp struct {
	Type	int8      `json:"type"`	//0世界聊天、1联盟聊天
	Msgs	[]ChatMsg `json:"msgs"`
}
type ChatMsg struct {
	RId      int    `json:"rid"`
	NickName string `json:"nickName"`
	Type	int8	`json:"type"`	//0世界聊天、1联盟聊天
	Msg		string 	`json:"msg"`
	Time	int64	`json:"time"`
}

type ChatReq struct {
	Type	int8	`json:"type"`	//0世界聊天、1联盟聊天
	Msg		string 	`json:"msg"`
}

type ExitReq struct {
	Type	int8	`json:"type"`	//0世界聊天、1联盟聊天
	Id		int		`json:"id"`
}

type ExitRsp struct {
	Type	int8	`json:"type"`	//0世界聊天、1联盟聊天
	Id		int		`json:"id"`
}

type LogoutReq struct {
	RId      int	`json:"RId"`
}

type LogoutRsp struct {
	RId      int	`json:"RId"`
}
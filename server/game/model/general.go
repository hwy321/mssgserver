package model

//抽卡
type DrawGeneralReq struct {
	DrawTimes int  `json:"drawTimes"` //抽卡次数
}

type DrawGeneralRsp struct {
	Generals []General `json:"generals"`
}

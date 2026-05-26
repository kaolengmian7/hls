package errors

// 业务错误码集中定义；前缀表示模块。
//
//	1xxxx -> 房间/会话
//	2xxxx -> 角色/准备
//	3xxxx -> 经济（金币、装备、抽奖）
//	9xxxx -> 通用/参数/系统
const (
	CodeRoomNotFound    = 10001
	CodeRoomFull        = 10002
	CodeRoomPlaying     = 10003
	CodeRoomClosed      = 10004
	CodePlayerDuplicate = 10005
	CodeInvalidSession  = 10006

	CodeRoleTaken      = 20001
	CodeRoleFull       = 20002
	CodeNoRoleAssigned = 20003
	CodeNotReady       = 20004
	CodeStartCondition = 20005

	CodeInsufficientCoin = 30001
	CodeItemNotFound     = 30004
	CodeWrongRole        = 30005

	CodeInvalidParam = 90001
	CodeForbidden    = 90002
	CodeInternal     = 99999
)

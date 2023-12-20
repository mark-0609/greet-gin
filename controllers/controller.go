package controllers

type Controller struct{}

// Response 返回信息结构体
type Response struct {
	Code int         `example:"0" json:"code"`
	Msg  string      `example:"success" json:"msg"`
	Data interface{} `json:"data"`
}

// SuccessMsg 成功msg Response
func (base *Controller) SuccessMsg(msg string) Response {
	return Response{
		Code: 0,
		Msg:  msg,
	}
}

// FailMsg 失败msg Response
func (base *Controller) FailMsg(msg string) Response {
	return Response{
		Code: -1,
		Msg:  msg,
	}
}

// UnauthorityMsg 没有权限msg Response
func (base *Controller) UnauthorityMsg(msg string) Response {
	return Response{
		Code: -2,
		Msg:  msg,
	}
}

// DataMsg 数据msg Response
func (base *Controller) DataMsg(data interface{}) Response {
	return Response{
		Code: 0,
		Data: data,
	}
}

// ResponseMsg 完全体Response
func (base *Controller) ResponseMsg(code int, msg string, data interface{}) Response {
	return Response{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}

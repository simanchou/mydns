// Code generated by goctl. DO NOT EDIT.
package types

type BaseResp struct {
	Ok bool `json:"ok"`
}

type AddReq struct {
	Name string `form:"name"`
}

type DelReq struct {
	Name string `path:"name"`
}

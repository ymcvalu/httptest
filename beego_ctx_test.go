package httptest

import (
	"bytes"
	"encoding/json"
	"github.com/astaxie/beego/context"
	"io"
	"log"
	"net/http"
	"testing"
)

type ReqParams struct {
	Name   string `json:"name"`
	Passwd string `json:"password"`
}

func beegoTestJsonHandler(ctx *context.Context) {
	buf := bytes.NewBuffer(nil)

	io.Copy(buf, ctx.Request.Body)

	req := ReqParams{}
	json.Unmarshal(buf.Bytes(), &req)

	log.Printf("receive params: %v", req)

	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	ctx.ResponseWriter.Write([]byte("success"))
}

func TestBeegoJsonParams(t *testing.T) {
	resp, ctx, err := NewContextBuilder().
		SetMethod("POST").
		SetJson(ReqParams{
			Name:   "Jim",
			Passwd: "123456",
		}).
		BeegoContext()
	if err != nil {
		t.Error(err)
		return
	}

	beegoTestJsonHandler(ctx)

	t.Log(resp.StatusCode(), string(resp.Body()))
}

type PageInfo struct {
	PageSize int `form:"page_size"`
	PageNum  int `form:"page_num"`
}

type UserReq struct {
	PageInfo
	Name string `form:"name"`
	Age  int    `form:"age"`
}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func beegoTestFormHandler(ctx *context.Context) {
	name := ctx.Request.FormValue("name")
	age := ctx.Request.FormValue("age")
	pn := ctx.Request.FormValue("page_num")
	ps := ctx.Request.FormValue("page_size")

	log.Printf("receiver form params: {name: %v, age: %s, pn: %s, ps: %s}", name, age, pn, ps)

	log.Printf("path params: id = %v", ctx.Input.Param(":id"))

	ctx.Output.Header("foo", "bar")
	ctx.WriteString(`{"code":200, "msg":"succ."}`)
}

func TestFormParams(t *testing.T) {
	resp, ctx, err := NewContextBuilder().SetMethod("POST").
		AddPathParam("country", "1").
		AddObjToForms(UserReq{
			PageInfo: PageInfo{
				PageNum:  10,
				PageSize: 15,
			},
			Name: "Jim",
			Age:  20,
		}).
		AddPathParam("id", 1).
		BeegoContext()

	if err != nil {
		t.Error(err)
		return
	}

	beegoTestFormHandler(ctx)

	t.Log(resp.StatusCode())
	t.Log(resp.Header().Get("foo"))
	ret := new(Result)
	resp.Decode(ret)
	t.Logf("%v", ret)
}

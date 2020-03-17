package httptest

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"testing"
)

func ginTestJsonHandler(ctx *gin.Context) {
	req := new(ReqParams)
	ctx.Bind(req)
	log.Println(*req)
	ctx.String(http.StatusOK, "success")
}

func TestGinJsonParams(t *testing.T) {
	resp, ctx, err := NewContextBuilder().
		SetMethod("POST").
		SetJson(ReqParams{
			Name:   "Jim",
			Passwd: "123456",
		}).
		GinContext()
	if err != nil {
		t.Error(err)
		return
	}

	ginTestJsonHandler(ctx)

	t.Log(resp.StatusCode(), string(resp.Body()))
}

func handleFormGin(ctx *gin.Context) {
	name := ctx.Request.FormValue("name")
	age := ctx.Request.FormValue("age")
	pn := ctx.Request.FormValue("page_num")
	ps := ctx.Request.FormValue("page_size")

	log.Printf("receiver form params: {name: %v, age: %s, pn: %s, ps: %s}", name, age, pn, ps)

	log.Printf("path params: id = %v", ctx.Param("id"))

	ctx.Header("foo", "bar")
	ctx.JSON(http.StatusOK, gin.H{
		"code": "200",
		"msg":  "success",
	})
}

func TestFormParamsGin(t *testing.T) {
	resp, ctx, err := NewContextBuilder().
		SetMethod("POST").
		AddPathParam("id", "1").
		AddObjToForms(UserReq{
			PageInfo: PageInfo{
				PageNum:  10,
				PageSize: 15,
			},
			Name: "Jim",
			Age:  20,
		}).
		GinContext()

	if err != nil {
		t.Error(err)
		return
	}

	handleFormGin(ctx)

	t.Log(resp.StatusCode())
	t.Log(resp.Header().Get("foo"))
	ret := new(Result)
	resp.Decode(ret)
	t.Logf("%v", ret)
}

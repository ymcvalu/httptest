# httptest
a simple tool help you write unit test for your http api, support for gin and beego

### example
```go
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

	ginHandler(ctx)

	t.Log(resp.StatusCode(), string(resp.Body()))
```

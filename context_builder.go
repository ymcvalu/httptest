package httptest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego/context"
	"github.com/gin-gonic/gin"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
)

type ContextBuilder struct {
	method   string
	url      string
	params   map[string]string
	query    url.Values
	formDada *multipart.Writer
	Header   map[string]string
	stream   io.Reader
	err      error
}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{
		method: http.MethodGet,
		url:    "http://localhost/api/v1/test",
	}
}

func (c *ContextBuilder) SetHeader(key, val string) *ContextBuilder {
	if c.err != nil {
		return c
	}

	if c.Header == nil {
		c.Header = make(map[string]string)
	}

	c.Header[key] = val

	return c
}

func (c *ContextBuilder) SetHeaders(headers map[string]string) *ContextBuilder {
	if c.err != nil {
		return c
	}

	for k, v := range headers {
		c.SetHeader(k, v)
	}
	return c
}

func (c *ContextBuilder) SetMethod(m string) *ContextBuilder {
	if c.err != nil {
		return c
	}

	c.method = m

	return c
}

func (c *ContextBuilder) SetURL(u string) *ContextBuilder {
	if c.err != nil {
		return c
	}
	c.url = u
	return c
}

// example: /api/v1/user/:id
func (c *ContextBuilder) AddPathParam(key string, val interface{}) *ContextBuilder {
	if c.err != nil {
		return c
	}

	if c.params == nil {
		c.params = make(map[string]string)
	}

	c.params[key] = fmt.Sprintf("%v", val)

	return c
}

// example: /api/v1/user?id=1
func (c *ContextBuilder) AddQuery(key, val string) *ContextBuilder {
	if c.err != nil {
		return c
	}
	if c.query == nil {
		c.query = make(url.Values)
	}
	vals := c.query[key]
	vals = append(vals, val)
	c.query[key] = vals
	return c
}

func (c *ContextBuilder) AddQueries(key map[string]string) *ContextBuilder {
	if c.err != nil {
		return c
	}
	if c.query == nil {
		c.query = make(url.Values)
	}

	for k, v := range key {
		vals := c.query[k]
		vals = append(vals, v)
		c.query[k] = vals
	}
	return c
}

const FORM_DATA_BOUNDRY = "------HttpTest"

func (c *ContextBuilder) lazyInitFormData() {
	if c.hasSetBody() {
		c.err = errors.New("set http body duplicately")
		return
	}

	if c.formDada == nil {
		buf := bytes.NewBuffer(nil)
		c.formDada = multipart.NewWriter(buf)
		c.stream = buf
		c.formDada.SetBoundary(FORM_DATA_BOUNDRY)
		c.SetHeader("Content-Type", "multipart/form-data;boundary="+FORM_DATA_BOUNDRY)
	}
}

func (c *ContextBuilder) AddForm(key, val string) *ContextBuilder {
	if c.err != nil {
		return c
	}

	c.lazyInitFormData()
	if c.err != nil {
		return c
	}

	c.err = c.formDada.WriteField(key, val)
	return c
}

func (c *ContextBuilder) AddForms(kvs map[string]string) *ContextBuilder {
	if c.err != nil {
		return c
	}

	c.lazyInitFormData()
	if c.err != nil {
		return c
	}

	for k, v := range kvs {
		c.err = c.formDada.WriteField(k, v)
		if c.err != nil {
			break
		}
	}
	return c
}

func (c *ContextBuilder) AddFile(key string, filename string, stream io.Reader) *ContextBuilder {
	if c.err != nil {
		return c
	}

	c.lazyInitFormData()
	if c.err != nil {
		return c
	}

	writer, err := c.formDada.CreateFormFile(key, filename)
	if err != nil {
		c.err = err
		return c
	}

	if _, err := io.Copy(writer, stream); err != nil {
		c.err = err
	}

	return c
}

func (c *ContextBuilder) AddFilePath(key, path string) *ContextBuilder {
	if c.err != nil {
		return c
	}

	c.lazyInitFormData()
	if c.err != nil {
		return c
	}

	_, filename := filepath.Split(path)

	fd, err := os.Open(path)
	if err != nil {
		c.err = err
		return c
	}
	defer fd.Close()

	w, err := c.formDada.CreateFormFile(key, filename)
	if err != nil {
		c.err = err
		return c
	}

	_, c.err = io.Copy(w, fd)
	return c
}

func (c *ContextBuilder) SetBody(stream io.Reader) *ContextBuilder {
	if c.err != nil {
		return c
	}

	if c.hasSetBody() {
		c.err = errors.New("set http body duplicately")
		return c
	}

	c.stream = stream
	return c
}

func (c *ContextBuilder) SetJson(obj interface{}) *ContextBuilder {
	if c.hasSetBody() {
		c.err = errors.New("set http body duplicately")
		return c
	}

	c.SetHeader("Content-Type", "application/json")
	objs, _ := json.Marshal(obj)
	c.stream = bytes.NewBuffer(objs)
	return c
}

func (c *ContextBuilder) AddObjToForms(obj interface{}) *ContextBuilder {
	if c.err != nil {
		return c
	}

	params := map[string]string{}
	c.err = obj2map(obj, params)
	if c.err != nil {
		return c
	}

	return c.AddForms(params)
}

func obj2map(obj interface{}, params map[string]string) error {
	typ := reflect.TypeOf(obj)
	val := reflect.ValueOf(obj)

	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("unsupported type[%s] for method AddObjToForms", typ.String())
	}

	for i := 0; i < typ.NumField(); i++ {
		fval := val.Field(i)
		fty := typ.Field(i)

		ftty := fty.Type

		for ftty.Kind() == reflect.Ptr {
			ftty = ftty.Elem()
			fval = fval.Elem()
		}

		if fty.Anonymous && ftty.Kind() == reflect.Struct {
			err := obj2map(fval.Interface(), params)
			if err != nil {
				return err
			}
			continue
		}

		name := fty.Name
		if len(name) == 0 || name[0] > 'Z' || name[0] < 'A' {
			continue
		}

		var val string
		switch ftty.Kind() {
		case reflect.Bool:
			if fval.Bool() {
				val = "true"
			} else {
				val = "false"
			}
		case reflect.Uintptr:
			fallthrough
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val = fmt.Sprintf("%d", fval.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val = fmt.Sprintf("%d", fval.Uint())
		case reflect.String:
			val = fval.String()
		}
		key := fty.Tag.Get("form")
		if key == "" {
			key = fty.Name
		}
		params[key] = val
	}
	return nil
}

func (c *ContextBuilder) respAndreq() (*response, *http.Request, error) {
	if c.err != nil {
		return nil, nil, c.err
	}

	if c.formDada != nil {
		err := c.formDada.Close()
		if err != nil {
			return nil, nil, err
		}
	}

	req := httptest.NewRequest(c.method, c.url, c.stream)
	for k, v := range c.Header {
		req.Header.Add(k, v)
	}

	kvs := req.URL.Query()
	for k, vs := range c.query {
		_vs := kvs[k]
		_vs = append(_vs, vs...)
		kvs[k] = _vs
	}

	req.URL.RawQuery = kvs.Encode()

	resp := &response{
		size:             noWritten,
		ResponseRecorder: httptest.NewRecorder(),
	}

	return resp, req, nil
}

func (c *ContextBuilder) BeegoContext() (HttpResponse, *context.Context, error) {
	resp, req, err := c.respAndreq()
	if err != nil {
		return nil, nil, err
	}

	ctx := context.NewContext() // new a beego context
	ctx.Reset(resp, req)

	for k, v := range c.params {
		if len(k) > 0 && k[0] != ':' {
			k = ":" + k
		}
		ctx.Input.SetParam(k, v)
	}

	return resp, ctx, nil
}

func (c *ContextBuilder) GinContext() (HttpResponse, *gin.Context, error) {
	resp, req, err := c.respAndreq()
	if err != nil {
		return nil, nil, err
	}

	ctx := gin.Context{
		Request: req,
		Writer:  resp,
	}

	params := make(gin.Params, 0, len(c.params))
	for k, v := range c.params {
		params = append(params, gin.Param{
			Key:   k,
			Value: v,
		})
	}
	ctx.Params = params

	return resp, &ctx, nil
}

func (c *ContextBuilder) hasSetBody() bool {
	return c.formDada != nil || c.stream != nil
}

type HttpResponse interface {
	http.Hijacker
	http.Flusher
	http.CloseNotifier
	http.ResponseWriter
	Size() int
	Status() int
	Written() bool
	Pusher() http.Pusher
	StatusCode() int
	WriteString(string) (int, error)
	WriteHeaderNow()
	Body() []byte
	Decode(obj interface{}) error // json unmarshal
	Response() *http.Response
}

const noWritten = -1

var _ HttpResponse = (*response)(nil)

type response struct {
	size int
	*httptest.ResponseRecorder
}

func (r *response) Decode(obj interface{}) error {
	return json.Unmarshal(r.ResponseRecorder.Body.Bytes(), obj)
}

func (r *response) Body() []byte {
	return r.ResponseRecorder.Body.Bytes()
}

func (r *response) Response() *http.Response {
	return r.ResponseRecorder.Result()
}

func (r *response) StatusCode() int {
	return r.Code
}

func (r *response) Status() int {
	return r.Code
}

func (r *response) CloseNotify() <-chan bool {
	return nil
}

func (r *response) Write(buf []byte) (int, error) {
	n, err := r.ResponseRecorder.Write(buf)
	r.size += n
	return n, err
}

func (r *response) WriteString(str string) (int, error) {
	n, err := r.ResponseRecorder.WriteString(str)
	r.size += n
	return n, err
}

func (r *response) WriteHeaderNow() {
	if !r.Written() {
		r.size = 0
	}
}

func (r *response) WriteHeader(code int) {
	if !r.Written() {
		r.size = 0
		r.ResponseRecorder.WriteHeader(code)
	}
}

func (r *response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func (r *response) Pusher() http.Pusher {
	return nil
}

func (r *response) Written() bool {
	return r.size != noWritten
}

func (r *response) Size() int {
	return r.size
}

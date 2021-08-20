package core

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"net/http/httputil"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	HttpHeaderKeyTraceID = "ipalfish-trace-id"

	WildCharacter = ":"

	RoutePath = "req-simple-path"

	HttpHeaderKeyGroup = "ipalfish-group"
	HttpHeaderKeyHead  = "ipalfish-head"

	CookieNameGroup = "ipalfish_group"
	DefaultGroup = ""

	ContextKeyHead = "Head"
)

// HttpServer is the http server, Create an instance of GinServer, by using NewGinServer()
type HttpServer struct {
	*gin.Engine
}

// Context warp gin Context
type Context struct {
	*gin.Context
}

// HandlerFunc ...
type HandlerFunc func(*Context)

// NewHttpServer create http server with gin
func NewHttpServer() *HttpServer {
	// 实例化gin Server
	router := gin.New()
	router.Use(Recovery(), InjectFromRequest(), Metric(), Trace())

	return &HttpServer{router}
}

// Use attachs a global middleware to the router
func (s *HttpServer) Use(middleware ...HandlerFunc) {
	s.Engine.Use(mutilWrap(middleware...)...)
}

// GET is a shortcut for router.Handle("GET", path, handle).
func (s *HttpServer) GET(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.GET(relativePath, ws...)
}

// POST is a shortcut for router.Handle("POST", path, handle).
func (s *HttpServer) POST(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.POST(relativePath, ws...)
}

// PUT is a shortcut for router.Handle("PUT", path, handle).
func (s *HttpServer) PUT(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.PUT(relativePath, ws...)
}

// Any registers a route that matches all the HTTP methods.
func (s *HttpServer) Any(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.Any(relativePath, ws...)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle).
func (s *HttpServer) DELETE(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.DELETE(relativePath, ws...)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle).
func (s *HttpServer) PATCH(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.PATCH(relativePath, ws...)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle).
func (s *HttpServer) OPTIONS(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.OPTIONS(relativePath, ws...)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle).
func (s *HttpServer) HEAD(relativePath string, handlers ...HandlerFunc) {
	ws := append([]gin.HandlerFunc{pathHook(relativePath)}, mutilWrap(handlers...)...)
	s.Engine.HEAD(relativePath, ws...)
}

// Bind checks the Content-Type to select a binding engine automatically
func (c *Context) Bind(obj interface{}) error {
	b := binding.Default(c.Request.Method, c.ContentType())
	return c.MustBindWith(obj, b)
}

func InjectFromRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = extractThriftUtilContextControlFromRequest(ctx, c.Request)
		ctx = extractThriftUtilContextHeadFromRequest(ctx, c.Request)
		c.Request = c.Request.WithContext(ctx)
	}
}

// Metric returns a metric middleware
func Metric() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = contextWithErrCode(ctx,1)
		c.Request = c.Request.WithContext(ctx)

		now := time.Now()
		c.Next()
		dt := time.Since(now)

		errCode := getErrCodeFromContext(c.Request.Context())
		if path, exist := c.Get(RoutePath); exist {
			if fun, ok := path.(string); ok {
				_,_,_ = dt,errCode,fun
				//report metric
				//group, serviceName := GetGroupAndService()
				//_metricAPIRequestCount.With(xprom.LabelGroupName, group, xprom.LabelServiceName, serviceName, xprom.LabelAPI, fun, xprom.LabelErrCode, strconv.Itoa(errCode)).Inc()
				//_metricAPIRequestTime.With(xprom.LabelGroupName, group, xprom.LabelServiceName, serviceName, xprom.LabelAPI, fun, xprom.LabelErrCode, strconv.Itoa(errCode)).Observe(float64(dt / time.Millisecond))
			}
		}
	}
}

// Trace returns a trace middleware
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		/*
		span := xtrace.SpanFromContext(c.Request.Context())
		if span == nil {
			newSpan, ctx := xtrace.StartSpanFromContext(c.Request.Context(), c.Request.RequestURI)
			c.Request.WithContext(ctx)
			span = newSpan
		}
		defer span.Finish()

		if sc, ok := span.Context().(jaeger.SpanContext); ok {
			c.Writer.Header()[HttpHeaderKeyTraceID] = []string{fmt.Sprint(sc.TraceID())}
		}

		c.Next()
		 */
	}
}

// Recovery returns a middleware that recovers from any panics and writes a 500 if there was one.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			var rawReq []byte
			if err := recover(); err != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				if c.Request != nil {
					rawReq, _ = httputil.DumpRequest(c.Request, false)
				}
				pl := fmt.Sprintf("http call panic: %s\n%v\n%s\n", string(rawReq), err, buf)
				fmt.Println(pl)
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}

func pathHook(relativePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		values := strings.Split(relativePath, WildCharacter)
		c.Set(RoutePath, values[0])
	}
}

func mutilWrap(handlers ...HandlerFunc) []gin.HandlerFunc {
	var h = make([]gin.HandlerFunc, len(handlers))
	for k, v := range handlers {
		h[k] = wrap(v)
	}
	return h
}

func wrap(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(&Context{c})
	}
}

func extractThriftUtilContextControlFromRequest(ctx context.Context, req *http.Request) context.Context {
	var group string
	if group = extractRouteGroupFromHost(req); group != "" {
		return injectRouteGroupToContext(ctx, group)
	}

	if group = extractRouteGroupFromHeader(req); group != "" {
		return injectRouteGroupToContext(ctx, group)
	}

	if group = extractRouteGroupFromCookie(req); group != "" {
		return injectRouteGroupToContext(ctx, group)
	}

	return injectRouteGroupToContext(ctx, DefaultGroup)
}

type Head struct {
	Uid        int64             `thrift:"uid,1" json:"uid"`
	Source     int32             `thrift:"source,2" json:"source"`
	Ip         string            `thrift:"ip,3" json:"ip"`
	Region     string            `thrift:"region,4" json:"region"`
	Dt         int32             `thrift:"dt,5" json:"dt"`
	Unionid    string            `thrift:"unionid,6" json:"unionid"`
	Did        string            `thrift:"did,7" json:"did"`
	Zone       int32             `thrift:"zone,8" json:"zone"`
	ZoneName   string            `thrift:"zone_name,9" json:"zone_name"`
	Properties map[string]string `thrift:"properties,10" json:"properties"`
}

func extractThriftUtilContextHeadFromRequest(ctx context.Context, req *http.Request) context.Context {
	// NOTE: 如果已经有了就先不覆盖
	val := ctx.Value(ContextKeyHead)
	if val != nil {
		return ctx
	}

	headJsonString := req.Header.Get(HttpHeaderKeyHead)
	var head Head
	_ = json.Unmarshal([]byte(headJsonString), &head)
	ctx = context.WithValue(ctx, ContextKeyHead, &head)
	return ctx
}

var domainRouteRegexp = regexp.MustCompile(`(?P<group>.+)\.group\..+`)

func extractRouteGroupFromHost(r *http.Request) (group string) {
	matches := domainRouteRegexp.FindStringSubmatch(r.Host)
	names := domainRouteRegexp.SubexpNames()
	for i, _ := range matches {
		if names[i] == "group" {
			group = matches[i]
		}
	}
	return
}
/*
func NewDefaultControl() *Control {
	return &Control{
		Route:  &Route{""},
		Ct:     time.Now().Unix(),
		Et:     0,
		Caller: &Endpoint{"", "", ""},
	}
}
 */
func injectRouteGroupToContext(ctx context.Context, group string) context.Context {
	/*
	control := thriftutil.NewDefaultControl()
	control.Route.Group = group

	return context.WithValue(ctx, xcontext.ContextKeyControl, control)
	 */
	return ctx
}

func extractRouteGroupFromHeader(r *http.Request) (group string) {
	return r.Header.Get(HttpHeaderKeyGroup)
}

func extractRouteGroupFromCookie(r *http.Request) (group string) {
	ck, err := r.Cookie(CookieNameGroup)
	if err == nil {
		group = ck.Value
	}
	return
}

type ErrCode struct {
	int
}

const ErrCodeKey = "ErrCode"

func ContextSetErrCode(ctx context.Context, errCode int) {
	errCodeContext, ok := ctx.Value(ErrCodeKey).(*ErrCode)
	if !ok {
		return
	}
	errCodeContext.int = errCode
	/*
	if span := xtrace.SpanFromContext(ctx); span != nil {
		span.SetTag("errcode", errCode)
	}
	 */
}

func contextWithErrCode(ctx context.Context, errCode int) context.Context {
	/*
	if span := xtrace.SpanFromContext(ctx); span != nil {
		span.SetTag("errcode", errCode)
	}
	 */
	return context.WithValue(ctx, ErrCodeKey, &ErrCode{errCode})
}

func getErrCodeFromContext(ctx context.Context) int {
	errCode, ok := ctx.Value(ErrCodeKey).(*ErrCode)
	if !ok {
		return 1
	}
	return errCode.int
}

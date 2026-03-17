# KRATOS脚手架

脚手架命令：cmd/kratos/main.go


### 1. APP

APP中包括的服务是`transport.Server`接口。




# 中间件

```
         ┌───────────────────┐
         │MIDDLEWARE 1       │
         │ ┌────────────────┐│
         │ │MIDDLEWARE 2    ││
         │ │ ┌─────────────┐││
         │ │ │MIDDLEWARE 3 │││
         │ │ │ ┌─────────┐ │││
REQUEST  │ │ │ │  YOUR   │ │││  RESPONSE
   ──────┼─┼─┼─▷ HANDLER ○─┼┼┼───▷
         │ │ │ └─────────┘ │││
         │ │ └─────────────┘││
         │ └────────────────┘│
         └───────────────────┘
```
```
// http
var opts = []http.ServerOption{
    http.Middleware(
        recovery.Recovery(), // 把中间件按照需要的顺序加入
        tracing.Server(),
        logging.Server(),
    ),
}
http.NewServer(opts...)

//grpc
var opts = []grpc.ServerOption{
    grpc.Middleware(
        recovery.Recovery(),  // 把中间件按照需要的顺序加入
        tracing.Server(),
        logging.Server(),
    ),
}
grpc.NewServer(opts...)
```

获得TRANSPORTER实例：
```
tr, ok := transport.FromServerContext(ctx)

func Middleware1() middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
            if tr, ok := transport.FromServerContext(ctx); ok {
                // Do something on entering
                defer func() {
                // Do something on exiting
                 }()
            }
            return handler(ctx, req)
        }
    }
}
```


# 传输协议

```
type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}
type Transporter interface {
	Kind() Kind             // 代表实现的通讯协议的种类，如内置的HTTP、GRPC，也可以实现其他的类型如MQTT，WEBSOCKET
	Endpoint() string       // 提供的服务终端地址	
	Operation() string      // 用于标识服务的完整方法路径，示例: /helloworld.Greeter/SayHello
	Header() Header         // HTTP的请求头或者GRPC的元数据
}
type Endpointer interface {
	Endpoint() (*url.URL, error) // 用于实现注册到注册中心的终端地址，如果不实现这个方法则不会注册到注册中心
}
```
各个实现协议会将各自的TRANSPORT实例传递给服务端或者客户端
```http.server.filter
func (s *Server) filter() mux.MiddlewareFunc {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

			var (
				ctx    context.Context
				cancel context.CancelFunc
			)

			if s.timeout > 0 {
				ctx, cancel = context.WithTimeout(req.Context(), s.timeout)
			} else {
				ctx, cancel = context.WithCancel(req.Context())
			}

			defer cancel()

			pathTemplate := req.URL.Path

			if route := mux.CurrentRoute(req); route != nil {

				// /path/123 -> /path/{id}

				pathTemplate, _ = route.GetPathTemplate()

			}

			tr := &Transport{
				operation:    pathTemplate,
				pathTemplate: pathTemplate,
				reqHeader:    headerCarrier(req.Header),
				replyHeader:  headerCarrier(w.Header()),
				request:      req,
				response:     w,
			}

			if s.endpoint != nil {
				tr.endpoint = s.endpoint.String()
			}

			tr.request = req.WithContext(transport.NewServerContext(ctx, tr))

			next.ServeHTTP(w, tr.request)

		})

	}

}
```
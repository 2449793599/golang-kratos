package kratos

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport"
)

// AppInfo is application context value.
type AppInfo interface {
	ID() string
	Name() string
	Version() string
	Metadata() map[string]string
	Endpoint() []string
}

// *********************************************************************************************************************
// App is an application components lifecycle manager.
type App struct {
	opts     options
	ctx      context.Context    // 应用上下文2：可取消
	cancel   context.CancelFunc // 应用上下文2：可取消
	mu       sync.Mutex
	instance *registry.ServiceInstance // 服务实例（服务注册用）
}

// New create an application lifecycle manager.
func New(opts ...Option) *App {

	o := options{
		ctx:              context.Background(), // 应用上下文1：默认
		sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT},
		registrarTimeout: 10 * time.Second,
	}

	if id, err := uuid.NewUUID(); err == nil {
		o.id = id.String()
	}

	for _, opt := range opts {
		opt(&o)
	}

	if o.logger != nil {
		log.SetLogger(o.logger)
	}

	ctx, cancel := context.WithCancel(o.ctx) // 应用上下文2：可取消

	return &App{
		ctx:    ctx,
		cancel: cancel,
		opts:   o,
	}

}

// ID returns app instance id.
func (a *App) ID() string { return a.opts.id }

// Name returns service name.
func (a *App) Name() string { return a.opts.name }

// Version returns app version.
func (a *App) Version() string { return a.opts.version }

// Metadata returns service metadata.
func (a *App) Metadata() map[string]string { return a.opts.metadata }

// Endpoint returns endpoints.
func (a *App) Endpoint() []string {

	if a.instance != nil {
		return a.instance.Endpoints
	}

	return nil

}

// *********************************************************************************************************************
// Run executes all OnStart hooks registered with the application's Lifecycle.
func (a *App) Run() error {

	instance, err := a.buildInstance() // 构建服务实例（服务注册用）

	if err != nil {
		return err
	}

	a.mu.Lock()
	a.instance = instance
	a.mu.Unlock()

	sctx := NewContext(a.ctx, a) // 应用上下文3：将整个APP放入到上下文中

	eg, ctx := errgroup.WithContext(sctx) // 应用上下文4：WaitGroup

	wg := sync.WaitGroup{}

	for _, fn := range a.opts.beforeStart {
		if err = fn(sctx); err != nil {
			return err
		}
	}

	octx := NewContext(a.opts.ctx, a)

	for _, srv := range a.opts.servers {

		server := srv

		eg.Go(func() error {

			<-ctx.Done() // 顺序9：wait for stop signal

			stopCtx := octx

			if a.opts.stopTimeout > 0 {

				var cancel context.CancelFunc

				stopCtx, cancel = context.WithTimeout(stopCtx, a.opts.stopTimeout)

				defer cancel()

			}

			return server.Stop(stopCtx)

		})

		wg.Add(1) // 顺序1

		eg.Go(func() error { // 顺序2

			wg.Done() // 顺序3 here is to ensure server start has begun running before register, so defer is not needed

			return server.Start(octx) // 启动服务（先启动服务然后才能注册）

		})

	}

	wg.Wait()

	if a.opts.registrar != nil { // 顺序4：

		rctx, rcancel := context.WithTimeout(ctx, a.opts.registrarTimeout)
		defer rcancel()

		if err = a.opts.registrar.Register(rctx, instance); err != nil { // 服务注册
			return err
		}

	}

	for _, fn := range a.opts.afterStart {
		if err = fn(sctx); err != nil {
			return err
		}
	}

	// ****************************************************************
	c := make(chan os.Signal, 1)

	signal.Notify(c, a.opts.sigs...)

	eg.Go(func() error {
		select {
		case <-ctx.Done(): // 顺序8：
			return nil
		case <-c:
			return a.Stop() // 顺序5：
		}
	})

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) { // 阻塞
		return err
	}

	err = nil

	for _, fn := range a.opts.afterStop {
		err = fn(sctx)
	}

	return err

}

// Stop gracefully stops the application.
func (a *App) Stop() (err error) {

	sctx := NewContext(a.ctx, a)

	for _, fn := range a.opts.beforeStop {
		err = fn(sctx)
	}

	a.mu.Lock()
	instance := a.instance
	a.mu.Unlock()

	if a.opts.registrar != nil && instance != nil { // 顺序6：

		ctx, cancel := context.WithTimeout(NewContext(a.ctx, a), a.opts.registrarTimeout)
		defer cancel()

		if err = a.opts.registrar.Deregister(ctx, instance); err != nil { // 服务注销
			return err
		}

	}

	if a.cancel != nil {
		a.cancel() // 顺序7：
	}

	return err

}

func (a *App) buildInstance() (*registry.ServiceInstance, error) {

	endpoints := make([]string, 0, len(a.opts.endpoints))

	for _, e := range a.opts.endpoints {
		endpoints = append(endpoints, e.String())
	}

	if len(endpoints) == 0 {

		for _, srv := range a.opts.servers { // 没有指定则从传输层服务中获取

			if r, ok := srv.(transport.Endpointer); ok {

				e, err := r.Endpoint()

				if err != nil {
					return nil, err
				}

				endpoints = append(endpoints, e.String())

			}

		}

	}

	return &registry.ServiceInstance{
		ID:        a.opts.id,
		Name:      a.opts.name,
		Version:   a.opts.version,
		Metadata:  a.opts.metadata,
		Endpoints: endpoints,
	}, nil

}

type appKey struct{}

// NewContext returns a new Context that carries value.
func NewContext(ctx context.Context, s AppInfo) context.Context {
	return context.WithValue(ctx, appKey{}, s)
}

// FromContext returns the Transport value stored in ctx, if any.
func FromContext(ctx context.Context) (s AppInfo, ok bool) {

	s, ok = ctx.Value(appKey{}).(AppInfo)

	return

}

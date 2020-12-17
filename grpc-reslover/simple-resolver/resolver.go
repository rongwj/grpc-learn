package resolver

import (
	"context"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	"google.golang.org/grpc/resolver"
)

const simpleScheme = "simple"

func init() {
	resolver.Register(&simpleBuilder{})
}

type endpoint struct {
	Name string
	Addr []string
}

type config struct {
	Endpoint []endpoint
}

type simpleResolver struct {
	name      string
	size      int
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	doResolve chan struct{}
	endpoint  map[string][]string
	cc        resolver.ClientConn
}

func (r *simpleResolver) ResolveNow(resolver.ResolveNowOptions) {
	r.doResolve <- struct{}{}
}

func (r *simpleResolver) watch() {
	defer r.wg.Done()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.doResolve:
		}
		addrs := []resolver.Address{}
		e, ok := r.endpoint[r.name]
		if !ok {
			continue
		}
		for _, a := range e {
			addrs = append(addrs, resolver.Address{
				Addr: a,
				Type: resolver.Backend,
			})
		}
		r.cc.NewAddress(addrs)
	}
}

func (r *simpleResolver) Close() {
	r.cancel()
	r.wg.Wait()
}

type simpleBuilder struct {
}

func (b *simpleBuilder) Build(t resolver.Target, cc resolver.ClientConn,
	_ resolver.BuildOptions) (resolver.Resolver, error) {

	filename := t.Endpoint
	if _, err := os.Stat(filename); err != nil {
		return nil, err
	}
	c := config{}
	if _, err := toml.DecodeFile(filename, &c); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	sr := simpleResolver{
		name:      t.Authority,
		cc:        cc,
		ctx:       ctx,
		cancel:    cancel,
		doResolve: make(chan struct{}),
	}

	sr.endpoint = make(map[string][]string)
	for _, e := range c.Endpoint {
		sr.endpoint[e.Name] = e.Addr
	}
	sr.wg.Add(1)
	go sr.watch()
	sr.ResolveNow(resolver.ResolveNowOptions{})
	return &sr, nil
}

func (*simpleBuilder) Scheme() string {
	return simpleScheme
}

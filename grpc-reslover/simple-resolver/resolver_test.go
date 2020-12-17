package resolver

import (
	"os"
	"sync"
	"testing"

	"google.golang.org/grpc/resolver"
)

type testClientConn struct {
	resolver.ClientConn // For unimplemented functions
	target              string
	m1                  sync.Mutex
	state               resolver.State
	updateStateCalls    int
	errChan             chan error
}

func (t *testClientConn) NewAddress(addrs []resolver.Address) {
	if len(addrs) == 0 {
		return
	}
	t.state.Addresses = make([]resolver.Address, len(addrs))
	copy(t.state.Addresses, addrs)
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestResolve(t *testing.T) {
	tests := []struct {
		target    string
		authority string
		addrWant  []string
	}{
		{
			"./test.toml",
			"one",
			[]string{"127.0.0.1:5000", "127.0.0.1:5001", "127.0.0.1:5002"},
		},
		{
			"./test.toml",
			"two",
			[]string{"127.0.0.2:6000", "127.0.0.2:6001", "127.0.0.2:6002"},
		},
		{
			"./test.toml",
			"three",
			[]string{"127.0.0.3:7000", "127.0.0.3:7001", "127.0.0.3:7002"},
		},
		{
			"./test.toml",
			"aaa",
			nil,
		},
	}
	b := simpleBuilder{}
	for _, test := range tests {
		cc := &testClientConn{target: test.target}
		r, err := b.Build(resolver.Target{
			Scheme:    "simple",
			Authority: test.authority,
			Endpoint:  test.target}, cc, resolver.BuildOptions{})
		if err != nil {
			t.Fatalf("%v\n", err)
		}
		addrs := []string{}
		for _, a := range cc.state.Addresses {
			addrs = append(addrs, a.Addr)
		}
		if false == compareStringSlice(test.addrWant, addrs) {
			t.Fatalf("Get:%v\nWant:%v\n", addrs, test.addrWant)
		}
		r.Close()
	}
}

func compareStringSlice(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

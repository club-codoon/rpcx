package clientselector

import (
	"errors"
	"math/rand"
	"net/rpc"
	"time"

	"github.com/smallnest/rpcx"
)

// ServerPeer is
type ServerPeer struct {
	Network, Address string
	Weight           int
}

// MultiClientSelector is used to select a direct rpc server from a list.
type MultiClientSelector struct {
	Servers            []*ServerPeer
	WeightedServers    []*Weighted
	SelectMode         rpcx.SelectMode
	dailTimeout        time.Duration
	rnd                *rand.Rand
	currentServer      int
	len                int
	HashServiceAndArgs HashServiceAndArgs
	Client             *rpcx.Client
}

// NewMultiClientSelector creates a MultiClientSelector
func NewMultiClientSelector(servers []*ServerPeer, sm rpcx.SelectMode, dailTimeout time.Duration) *MultiClientSelector {
	s := &MultiClientSelector{
		Servers:     servers,
		SelectMode:  sm,
		dailTimeout: dailTimeout,
		rnd:         rand.New(rand.NewSource(time.Now().UnixNano())),
		len:         len(servers)}

	if sm == rpcx.WeightedRoundRobin {
		s.WeightedServers = make([]*Weighted, len(s.Servers))
		for i, ss := range s.Servers {
			s.WeightedServers[i] = &Weighted{Server: ss, Weight: ss.Weight, EffectiveWeight: ss.Weight}
		}
	}

	s.currentServer = s.rnd.Intn(s.len)
	return s
}

func (s *MultiClientSelector) SetClient(c *rpcx.Client) {
	s.Client = c
}

func (s *MultiClientSelector) SetSelectMode(sm rpcx.SelectMode) {
	s.SelectMode = sm
}

func (s *MultiClientSelector) AllClients(clientCodecFunc rpcx.ClientCodecFunc) []*rpc.Client {
	var clients []*rpc.Client

	for _, sv := range s.Servers {
		c, err := rpcx.NewDirectRPCClient(s.Client, clientCodecFunc, sv.Network, sv.Address, s.dailTimeout)
		if err == nil {
			clients = append(clients, c)
		}
	}

	return clients
}

//Select returns a rpc client
func (s *MultiClientSelector) Select(clientCodecFunc rpcx.ClientCodecFunc, options ...interface{}) (*rpc.Client, error) {
	if s.len == 0 {
		return nil, errors.New("No available service")
	}

	if s.SelectMode == rpcx.RandomSelect {
		s.currentServer = s.rnd.Intn(s.len)
		peer := s.Servers[s.currentServer]
		return rpcx.NewDirectRPCClient(s.Client, clientCodecFunc, peer.Network, peer.Address, s.dailTimeout)

	} else if s.SelectMode == rpcx.RoundRobin {
		s.currentServer = (s.currentServer + 1) % s.len //not use lock for performance so it is not precise even
		peer := s.Servers[s.currentServer]
		return rpcx.NewDirectRPCClient(s.Client, clientCodecFunc, peer.Network, peer.Address, s.dailTimeout)
	} else if s.SelectMode == rpcx.ConsistentHash {
		if s.HashServiceAndArgs == nil {
			s.HashServiceAndArgs = JumpConsistentHash
		}
		s.currentServer = s.HashServiceAndArgs(s.len, options...)
		peer := s.Servers[s.currentServer]
		return rpcx.NewDirectRPCClient(s.Client, clientCodecFunc, peer.Network, peer.Address, s.dailTimeout)
	} else if s.SelectMode == rpcx.WeightedRoundRobin {
		best := nextWeighted(s.WeightedServers)
		peer := best.Server.(*ServerPeer)
		return rpcx.NewDirectRPCClient(s.Client, clientCodecFunc, peer.Network, peer.Address, s.dailTimeout)
	}

	return nil, errors.New("not supported SelectMode: " + s.SelectMode.String())
}

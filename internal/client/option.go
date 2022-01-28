/*
 * Copyright 2021 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package client defines the Options of client
package client

import (
	"time"

	"github.com/minogump/kitex/internal/configutil"
	internal_stats "github.com/minogump/kitex/internal/stats"
	"github.com/minogump/kitex/pkg/acl"
	"github.com/minogump/kitex/pkg/circuitbreak"
	connpool2 "github.com/minogump/kitex/pkg/connpool"
	"github.com/minogump/kitex/pkg/diagnosis"
	"github.com/minogump/kitex/pkg/discovery"
	"github.com/minogump/kitex/pkg/endpoint"
	"github.com/minogump/kitex/pkg/event"
	"github.com/minogump/kitex/pkg/http"
	"github.com/minogump/kitex/pkg/loadbalance"
	"github.com/minogump/kitex/pkg/loadbalance/lbcache"
	"github.com/minogump/kitex/pkg/proxy"
	"github.com/minogump/kitex/pkg/remote"
	"github.com/minogump/kitex/pkg/remote/codec"
	"github.com/minogump/kitex/pkg/remote/codec/protobuf"
	"github.com/minogump/kitex/pkg/remote/codec/thrift"
	"github.com/minogump/kitex/pkg/remote/connpool"
	"github.com/minogump/kitex/pkg/remote/trans/netpoll"
	"github.com/minogump/kitex/pkg/remote/trans/nphttp2"
	"github.com/minogump/kitex/pkg/retry"
	"github.com/minogump/kitex/pkg/rpcinfo"
	"github.com/minogump/kitex/pkg/serviceinfo"
	"github.com/minogump/kitex/pkg/stats"
	"github.com/minogump/kitex/pkg/transmeta"
	"github.com/minogump/kitex/pkg/utils"
	"github.com/minogump/kitex/transport"
)

func init() {
	remote.PutPayloadCode(serviceinfo.Thrift, thrift.NewThriftCodec())
	remote.PutPayloadCode(serviceinfo.Protobuf, protobuf.NewProtobufCodec())
}

// Options is used to initialize a client.
type Options struct {
	Cli     *rpcinfo.EndpointBasicInfo
	Svr     *rpcinfo.EndpointBasicInfo
	Configs rpcinfo.RPCConfig
	Locks   *ConfigLocks
	Once    *configutil.OptionOnce

	MetaHandlers []remote.MetaHandler

	RemoteOpt        *remote.ClientOption
	Proxy            proxy.ForwardProxy
	Resolver         discovery.Resolver
	HTTPResolver     http.Resolver
	Balancer         loadbalance.Loadbalancer
	BalancerCacheOpt *lbcache.Options
	PoolCfg          *connpool2.IdleConfig
	ErrHandle        func(error) error
	Targets          string
	CBSuite          *circuitbreak.CBSuite
	Timeouts         rpcinfo.TimeoutProvider

	ACLRules []acl.RejectFunc

	MWBs  []endpoint.MiddlewareBuilder
	IMWBs []endpoint.MiddlewareBuilder

	Bus          event.Bus
	Events       event.Queue
	ExtraTimeout time.Duration

	// DebugInfo should only contains objects that are suitable for json serialization.
	DebugInfo    utils.Slice
	DebugService diagnosis.Service

	// Observability
	TracerCtl  *internal_stats.Controller
	StatsLevel *stats.Level

	// retry policy
	RetryPolicy    *retry.Policy
	RetryContainer *retry.Container

	CloseCallbacks []func() error
}

// Apply applies all options.
func (o *Options) Apply(opts []Option) {
	for _, op := range opts {
		op.F(o, &o.DebugInfo)
	}
}

// Option is the only way to config client.
type Option struct {
	F func(o *Options, di *utils.Slice)
}

// NewOptions creates a new option.
func NewOptions(opts []Option) *Options {
	o := &Options{
		Cli:          &rpcinfo.EndpointBasicInfo{Tags: make(map[string]string)},
		Svr:          &rpcinfo.EndpointBasicInfo{Tags: make(map[string]string)},
		RemoteOpt:    newClientOption(),
		Configs:      rpcinfo.NewRPCConfig(),
		Locks:        NewConfigLocks(),
		Once:         configutil.NewOptionOnce(),
		HTTPResolver: http.NewDefaultResolver(),
		DebugService: diagnosis.NoopService,

		Bus:    event.NewEventBus(),
		Events: event.NewQueue(event.MaxEventNum),

		TracerCtl: &internal_stats.Controller{},
	}
	o.Apply(opts)
	o.MetaHandlers = append(o.MetaHandlers, transmeta.MetainfoClientHandler)

	o.initRemoteOpt()

	if o.RetryContainer != nil && o.DebugService != nil {
		o.DebugService.RegisterProbeFunc(diagnosis.RetryPolicyKey, o.RetryContainer.Dump)
	}

	if o.StatsLevel == nil {
		level := stats.LevelDisabled
		if o.TracerCtl.HasTracer() {
			level = stats.LevelDetailed
		}
		o.StatsLevel = &level
	}
	return o
}

func (o *Options) initRemoteOpt() {
	if o.Configs.TransportProtocol()&transport.GRPC == transport.GRPC {
		o.RemoteOpt.ConnPool = nphttp2.NewConnPool(o.Svr.ServiceName)
		o.RemoteOpt.CliHandlerFactory = nphttp2.NewCliTransHandlerFactory()
	}
	if o.RemoteOpt.ConnPool == nil {
		if o.PoolCfg != nil {
			var zero connpool2.IdleConfig
			if *o.PoolCfg == zero {
				o.RemoteOpt.ConnPool = connpool.NewShortPool(o.Svr.ServiceName)
			} else {
				o.RemoteOpt.ConnPool = connpool.NewLongPool(o.Svr.ServiceName, *o.PoolCfg)
			}
		} else {
			o.RemoteOpt.ConnPool = connpool.NewLongPool(
				o.Svr.ServiceName,
				connpool2.IdleConfig{
					MaxIdlePerAddress: 10,
					MaxIdleGlobal:     100,
					MaxIdleTimeout:    time.Minute,
				},
			)
		}
	}
	pool := o.RemoteOpt.ConnPool
	o.CloseCallbacks = append(o.CloseCallbacks, pool.Close)

	if df, ok := pool.(interface{ Dump() interface{} }); ok {
		o.DebugService.RegisterProbeFunc(diagnosis.ConnPoolKey, df.Dump)
	}
	if r, ok := pool.(remote.ConnPoolReporter); ok && o.RemoteOpt.EnableConnPoolReporter {
		r.EnableReporter()
	}

	if long, ok := pool.(remote.LongConnPool); ok {
		o.Bus.Watch(discovery.ChangeEventName, func(ev *event.Event) {
			ch, ok := ev.Extra.(*discovery.Change)
			if !ok {
				return
			}
			for _, inst := range ch.Removed {
				if addr := inst.Address(); addr != nil {
					long.Clean(addr.Network(), addr.String())
				}
			}
		})
	}
}

func newClientOption() *remote.ClientOption {
	return &remote.ClientOption{
		CliHandlerFactory: netpoll.NewCliTransHandlerFactory(),
		Dialer:            netpoll.NewDialer(),
		Codec:             codec.NewDefaultCodec(),
	}
}

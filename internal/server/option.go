/*
 * Copyright 2021 CloudWeGo
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package server defines the Options of server
package server

import (
	"github.com/minogump/kitex/internal/configutil"
	internal_stats "github.com/minogump/kitex/internal/stats"
	"github.com/minogump/kitex/pkg/acl"
	"github.com/minogump/kitex/pkg/diagnosis"
	"github.com/minogump/kitex/pkg/endpoint"
	"github.com/minogump/kitex/pkg/event"
	"github.com/minogump/kitex/pkg/klog"
	"github.com/minogump/kitex/pkg/limit"
	"github.com/minogump/kitex/pkg/limiter"
	"github.com/minogump/kitex/pkg/proxy"
	"github.com/minogump/kitex/pkg/registry"
	"github.com/minogump/kitex/pkg/remote"
	"github.com/minogump/kitex/pkg/remote/codec"
	"github.com/minogump/kitex/pkg/remote/codec/protobuf"
	"github.com/minogump/kitex/pkg/remote/codec/thrift"
	"github.com/minogump/kitex/pkg/remote/trans/detection"
	"github.com/minogump/kitex/pkg/remote/trans/netpoll"
	"github.com/minogump/kitex/pkg/rpcinfo"
	"github.com/minogump/kitex/pkg/serviceinfo"
	"github.com/minogump/kitex/pkg/stats"
	"github.com/minogump/kitex/pkg/transmeta"
	"github.com/minogump/kitex/pkg/utils"
)

func init() {
	remote.PutPayloadCode(serviceinfo.Thrift, thrift.NewThriftCodec())
	remote.PutPayloadCode(serviceinfo.Protobuf, protobuf.NewProtobufCodec())
}

// Option is the only way to config a server.
type Option struct {
	F func(o *Options, di *utils.Slice)
}

// Options is used to initialize the server.
type Options struct {
	Svr      *rpcinfo.EndpointBasicInfo
	Configs  rpcinfo.RPCConfig
	LockBits int
	Once     *configutil.OptionOnce

	MetaHandlers []remote.MetaHandler

	RemoteOpt *remote.ServerOption
	ErrHandle func(error) error
	Proxy     proxy.BackwardProxy

	// Registry is used for service registry.
	Registry registry.Registry
	// RegistryInfo is used to in registry.
	RegistryInfo *registry.Info

	ACLRules      []acl.RejectFunc
	Limits        *limit.Option
	LimitReporter limiter.LimitReporter

	MWBs []endpoint.MiddlewareBuilder

	Bus    event.Bus
	Events event.Queue

	// DebugInfo should only contains objects that are suitable for json serialization.
	DebugInfo    utils.Slice
	DebugService diagnosis.Service

	// Observability
	Logger     klog.FormatLogger
	TracerCtl  *internal_stats.Controller
	StatsLevel *stats.Level
}

// NewOptions creates a default options.
func NewOptions(opts []Option) *Options {
	o := &Options{
		Svr:          &rpcinfo.EndpointBasicInfo{},
		Configs:      rpcinfo.NewRPCConfig(),
		Once:         configutil.NewOptionOnce(),
		RemoteOpt:    newServerOption(),
		Logger:       klog.DefaultLogger(),
		DebugService: diagnosis.NoopService,

		Bus:    event.NewEventBus(),
		Events: event.NewQueue(event.MaxEventNum),

		TracerCtl: &internal_stats.Controller{},
		Registry:  registry.NoopRegistry,
	}
	ApplyOptions(opts, o)
	o.MetaHandlers = append(o.MetaHandlers, transmeta.MetainfoServerHandler)
	rpcinfo.AsMutableRPCConfig(o.Configs).LockConfig(o.LockBits)
	if o.StatsLevel == nil {
		level := stats.LevelDisabled
		if o.TracerCtl.HasTracer() {
			level = stats.LevelDetailed
		}
		o.StatsLevel = &level
	}
	return o
}

func newServerOption() *remote.ServerOption {
	return &remote.ServerOption{
		TransServerFactory:    netpoll.NewTransServerFactory(),
		SvrHandlerFactory:     detection.NewSvrTransHandlerFactory(),
		Codec:                 codec.NewDefaultCodec(),
		Address:               defaultAddress,
		ExitWaitTime:          defaultExitWaitTime,
		MaxConnectionIdleTime: defaultConnectionIdleTime,
		AcceptFailedDelayTime: defaultAcceptFailedDelayTime,
	}
}

// ApplyOptions applies the given options.
func ApplyOptions(opts []Option, o *Options) {
	for _, op := range opts {
		op.F(o, &o.DebugInfo)
	}
}

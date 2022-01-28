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

package transmeta

import (
	"testing"

	"github.com/minogump/kitex/internal/mocks"
	"github.com/minogump/kitex/internal/test"
	"github.com/minogump/kitex/pkg/remote"
	"github.com/minogump/kitex/pkg/rpcinfo"
	"github.com/minogump/kitex/pkg/serviceinfo"
	"github.com/minogump/kitex/transport"
)

func TestIsTTHeader(t *testing.T) {
	t.Run("with ttheader", func(t *testing.T) {
		ri := rpcinfo.NewRPCInfo(nil, nil, rpcinfo.NewInvocation("", ""), nil, rpcinfo.NewRPCStats())
		msg := remote.NewMessage(nil, mocks.ServiceInfo(), ri, remote.Call, remote.Server)
		msg.SetProtocolInfo(remote.NewProtocolInfo(transport.TTHeader, serviceinfo.Thrift))
		test.Assert(t, isTTHeader(msg))
	})
	t.Run("with ttheader framed", func(t *testing.T) {
		ri := rpcinfo.NewRPCInfo(nil, nil, rpcinfo.NewInvocation("", ""), nil, rpcinfo.NewRPCStats())
		msg := remote.NewMessage(nil, mocks.ServiceInfo(), ri, remote.Call, remote.Server)
		msg.SetProtocolInfo(remote.NewProtocolInfo(transport.TTHeaderFramed, serviceinfo.Thrift))
		test.Assert(t, isTTHeader(msg))
	})
}

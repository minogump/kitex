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

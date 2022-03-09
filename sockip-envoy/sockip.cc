#include "sockip.h"

#include "envoy/buffer/buffer.h"
#include "envoy/network/connection.h"

#include "source/common/common/assert.h"

namespace Envoy {
namespace Filter {

Network::FilterStatus SockIP::onAccept(Network::ListenerFilterCallbacks& cb) {
  ENVOY_LOG(trace, "sockip: connection accepted");
  Network::ConnectionSocket& socket = cb.socket();
  if (socket.addressType() == Network::Address::Type::Ip) {
    auto address = socket.ioHandle().localAddress();
    if (address) {
      // restore the local address to what is actually on the socket
      // since we're handling redirects
      socket.connectionInfoProvider().restoreLocalAddress(address);
    }
  }
  return Network::FilterStatus::Continue;
}

} // namespace Filter
} // namespace Envoy

#include <string>

#include "sockip.h"

#include "envoy/registry/registry.h"
#include "envoy/server/filter_config.h"

namespace Envoy {
namespace Server {
namespace Configuration {

/**
 * Config registration for the sockip filter. @see NamedNetworkFilterConfigFactory.
 */
class SockIPConfigFactory : public NamedListenerFilterConfigFactory {
public:
  Network::ListenerFilterFactoryCb createListenerFilterFactoryFromProto(
    const Protobuf::Message&,
    const Network::ListenerFilterMatcherSharedPtr& listener_filter_matcher,
    ListenerFactoryContext& context) override {
    return [listener_filter_matcher, traffic_direction = context.listenerConfig().direction()](
              Network::ListenerFilterManager& filter_manager) -> void {
      filter_manager.addAcceptFilter(listener_filter_matcher,
                                    std::make_unique<Filter::SockIP>(traffic_direction));
    };
  }

  ProtobufTypes::MessagePtr createEmptyConfigProto() override {
    return ProtobufTypes::MessagePtr{new Envoy::ProtobufWkt::Struct()};
  }

  std::string name() const override { return "sockip.filter"; }
};

/**
 * Static registration for the echo2 filter. @see RegisterFactory.
 */
REGISTER_FACTORY(SockIPConfigFactory, NamedListenerFilterConfigFactory){"sockip"};

} // namespace Configuration
} // namespace Server
} // namespace Envoy

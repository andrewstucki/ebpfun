#pragma once

#include "envoy/network/filter.h"

#include "source/common/common/logger.h"

namespace Envoy {
namespace Filter {

/**
 * Implementation of a basic echo filter.
 */
class SockIP : public Network::ListenerFilter, Logger::Loggable<Logger::Id::filter> {
public:
  SockIP(const envoy::config::core::v3::TrafficDirection& traffic_direction)
      : traffic_direction_(traffic_direction) {
    // [[maybe_unused]] attribute is not supported on GCC for class members. We trivially use the
    // parameter here to silence the warning.
    (void)traffic_direction_;
  }
  // Network::ListenerFilter
  Network::FilterStatus onAccept(Network::ListenerFilterCallbacks& cb) override;

private:
  envoy::config::core::v3::TrafficDirection traffic_direction_;
};

} // namespace Filter
} // namespace Envoy

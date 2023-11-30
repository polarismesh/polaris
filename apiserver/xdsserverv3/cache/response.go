package cache

import (
	"errors"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

type NoReadyXdsResponse struct{
	cachev3.DeltaResponse
}

func (r *NoReadyXdsResponse) GetDeltaRequest() *discovery.DeltaDiscoveryRequest{
	return nil
}

func (r *NoReadyXdsResponse) GetDeltaDiscoveryResponse() (*discovery.DeltaDiscoveryResponse, error){
	return nil, errors.New("node xds not created yet")
}
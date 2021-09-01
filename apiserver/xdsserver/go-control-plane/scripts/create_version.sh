#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

MODULES=(   'clusterservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":clusterservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/cluster/v3"'
            'endpointservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":endpointservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/endpoint/v3"'
            'listenerservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":listenerservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/listener/v3"'
            'routeservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":routeservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/route/v3"'
            'runtimeservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/discovery/v2":runtimeservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/runtime/v3"'
            'secretservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/discovery/v2":secretservice "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/secret/v3"'
            'accessloggrpc "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/accesslog/v2":accessloggrpc "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/accesslog/v3"'
            'discovery "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":discovery "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/discovery/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/resource/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/resource/v3"'
            'cluster "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":cluster "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/cluster/v3"'
            'endpoint "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":endpoint "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/endpoint/v3"'
            'listener "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":listener "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/listener/v3"'
            'route "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2":route "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/route/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2/route":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/route/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/filter/network/http_connection_manager/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2/core":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/core/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/cache/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/cache/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2/endpoint":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/endpoint/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2/auth":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/extensions/transport_sockets/tls/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/api/v2/listener":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/listener/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/accesslog/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/extensions/access_loggers/grpc/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/filter/accesslog/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/accesslog/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/config/filter/network/tcp_proxy/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"'
            'runtime "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/discovery/v2":runtime "github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/runtime/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/discovery/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/service/discovery/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/test/resource/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/test/resource/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/data/accesslog/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/envoy/data/accesslog/v3"'
            '"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/server/v2":"github.com/polarismesh/polaris-server/apiserver/xdsserver/go-control-plane/pkg/server/v3"'
)

workdir="$(dirname "$0")"
. "$workdir/export.sh"

SED="sed"
case "${OSTYPE}" in
    "darwin"*) proto_os="osx"; SED="gsed" ;;
esac

if ! which $SED > /dev/null; then
    printf "Please install $SED"
    exit 1
fi

for dir in "${DIRS[@]}" ; do
    v2dir="$dir/v2"
    v3dir="$dir/v3"
    printf "Deleting %s\n" "$v3dir"
    rm -rf "$v3dir"
    cp -r "$v2dir" "$v3dir"
    FILES=($(ls -p "$v3dir"))
    for file in "${FILES[@]}" ; do
        filepath="$v3dir/$file"
        printf "creating %s\n" "$filepath"
        for module in "${MODULES[@]}" ; do
            KEY=${module%%:*}
            VALUE=${module#*:}
            "$SED" -i.bak "s|$KEY|$VALUE|" "$filepath" && rm "$filepath.bak"
        done
        "$SED" -i.bak '1s|^|// Code generated by create_version. DO NOT EDIT.\n|' "$filepath" && rm "$filepath.bak"
        gofmt -s -w "$filepath"
    done
done

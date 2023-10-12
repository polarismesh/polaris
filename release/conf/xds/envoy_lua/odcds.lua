-- Called on the request path.
function envoy_on_request(request_handle)
    local service_name = request_handle:headers():get(":authority")
    local service_namespace = os.getenv("SIDECAR_NAMESPACE")
    local cluster_name = service_name .. "." .. service_namespace
    request_handle:headers():remove(":authority")
    request_handle:headers():add(":authority", cluster_name)
end

-- Called on the response path.
function envoy_on_response(response_handle)
  -- Do something.
end
package ratelimit

//go:generate go build -o rate-limit-client client/client.go
//go:generate go build -o rate-limit-server-v1 server/v1/server.go
//go:generate go build -o rate-limit-server-v2 server/v2/server.go
//go:generate ./rate-limit-client

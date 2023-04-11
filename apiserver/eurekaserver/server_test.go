package eurekaserver

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_parsePeersToReplicate(t *testing.T) {
	type args struct {
		defaultNamespace  string
		replicatePeerObjs []interface{}
	}

	defaultNamespace := "default"

	tests := []struct {
		name string
		args args
		want map[string][]string
	}{
		{
			name: "empty",
			args: args{
				defaultNamespace:  defaultNamespace,
				replicatePeerObjs: []interface{}{},
			},
			want: map[string][]string{},
		},
		{
			name: "single-default",
			args: args{
				defaultNamespace: defaultNamespace,
				replicatePeerObjs: []interface{}{
					"127.0.0.1:8761",
					"127.0.0.1:8762",
				},
			},
			want: map[string][]string{
				defaultNamespace: {
					"127.0.0.1:8761",
					"127.0.0.1:8762",
				},
			},
		},
		{
			name: "multi-namespace",
			args: args{
				defaultNamespace: defaultNamespace,
				replicatePeerObjs: []interface{}{
					"127.0.0.1:8761",
					"127.0.0.1:8762",
					map[interface{}]interface{}{
						"ns1": []interface{}{
							"127.0.0.1:8763",
							"127.0.0.1:8764",
						},
					},
					map[interface{}]interface{}{
						"ns2": []interface{}{
							"127.0.0.1:8765",
							"127.0.0.1:8766",
						},
					},
				},
			},
			want: map[string][]string{
				defaultNamespace: {
					"127.0.0.1:8761",
					"127.0.0.1:8762",
				},
				"ns1": {
					"127.0.0.1:8763",
					"127.0.0.1:8764",
				},
				"ns2": {
					"127.0.0.1:8765",
					"127.0.0.1:8766",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, parsePeersToReplicate(tt.args.defaultNamespace, tt.args.replicatePeerObjs), "parsePeersToReplicate(%v, %v)", tt.args.defaultNamespace, tt.args.replicatePeerObjs)
		})
	}
}

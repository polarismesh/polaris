package service

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
)

// Test_checkResourceName checks if the resource name is valid.
func Test_checkResourceName(t *testing.T) {
	type args struct {
		name *wrappers.StringValue
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil test", args: args{
			name: nil,
		}, wantErr: true},
		{name: "empty test", args: args{
			name: &wrappers.StringValue{Value: ""},
		}, wantErr: true},
		{name: "illegal treatment", args: args{
			name: &wrappers.StringValue{Value: "a-b-c-d-#"},
		}, wantErr: true},
		{name: "normal treatment", args: args{
			name: &wrappers.StringValue{Value: "a-b-c-d"},
		}, wantErr: false},
		{name: "normal treatment-backslash", args: args{
			name: &wrappers.StringValue{Value: "/a/b/c/d"},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkResourceName(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("checkResourceName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

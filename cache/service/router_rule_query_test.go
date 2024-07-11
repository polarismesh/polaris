/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package service

import (
	"testing"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/model"
)

func Test_queryRoutingRuleV2ByService(t *testing.T) {
	type args struct {
		rule            *model.ExtendRouterConfig
		sourceNamespace string
		sourceService   string
		destNamespace   string
		destService     string
		both            bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "命名空间-或-精确查询",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "test-1",
				sourceService:   "",
				destNamespace:   "test-2",
				destService:     "",
				both:            false,
			},
			want: true,
		},
		{
			name: "命名空间-与-精确查询",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "test-1",
				sourceService:   "",
				destNamespace:   "test-2",
				destService:     "",
				both:            true,
			},
			want: false,
		},
		{
			name: "命名空间-或-模糊查询",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "test*",
				sourceService:   "",
				destNamespace:   "test-1",
				destService:     "",
				both:            false,
			},
			want: true,
		},
		{
			name: "命名空间-与-模糊查询",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "test*",
				sourceService:   "",
				destNamespace:   "tesa*",
				destService:     "",
				both:            true,
			},
			want: false,
		},
		{
			name: "(命名空间精确查询+服务名精确查询)-或",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "test-1",
				sourceService:   "test-1",
				destNamespace:   "test-1",
				destService:     "test-1",
				both:            false,
			},
			want: true,
		},
		{
			name: "(命名空间精确查询+服务名精确查询)-与",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "test-1",
				sourceService:   "test-1",
				destNamespace:   "test-1",
				destService:     "test-1",
				both:            true,
			},
			want: true,
		},
		{
			name: "(命名空间精确查询+服务名精确查询)-与",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "test-1",
				sourceService:   "test-1",
				destNamespace:   "test-1",
				destService:     "test-2",
				both:            true,
			},
			want: false,
		},
		{
			name: "(命名空间模糊+服务名精确查询)-或",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "te*",
				sourceService:   "test-1",
				destNamespace:   "tes*",
				destService:     "test-1",
				both:            false,
			},
			want: true,
		},
		{
			name: "(命名空间模糊+服务名精确查询)-或",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "te*",
				sourceService:   "test-1",
				destNamespace:   "",
				destService:     "",
				both:            false,
			},
			want: true,
		},
		{
			name: "(命名空间模糊+服务名精确查询)-或",
			args: args{
				rule: &model.ExtendRouterConfig{
					RuleRouting: &model.RuleRoutingConfigWrapper{
						RuleRouting: &apitraffic.RuleRoutingConfig{
							Rules: []*apitraffic.SubRuleRouting{
								{
									Sources: []*apitraffic.SourceService{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
									Destinations: []*apitraffic.DestinationGroup{
										{
											Service:   "test-1",
											Namespace: "test-1",
										},
									},
								},
							},
						},
					},
				},
				sourceNamespace: "",
				sourceService:   "",
				destNamespace:   "tese*",
				destService:     "test-1",
				both:            false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := queryRoutingRuleV2ByService(tt.args.rule, tt.args.sourceNamespace, tt.args.sourceService,
				tt.args.destNamespace, tt.args.destService, tt.args.both); got != tt.want {

				t.Errorf("queryRoutingRuleV2ByService() = %v, want %v", got, tt.want)
			}
		})
	}
}

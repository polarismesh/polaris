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

package i18n

import (
	"testing"

	"golang.org/x/text/language"

	api "github.com/polarismesh/polaris/common/api/v1"
)

func init() {
	LoadI18nMessageFile("../../../release/conf/i18n/en.toml")
	LoadI18nMessageFile("../../../release/conf/i18n/zh.toml")
}

func Test_Translate(t *testing.T) {
	type args struct {
		lang    language.Tag
		errCode uint32
		want    string
	}

	tests := []args{
		{
			lang:    language.Chinese,
			errCode: api.ExecuteSuccess,
			want:    "执行成功",
		},
		{
			lang:    language.English,
			errCode: api.ExecuteSuccess,
			want:    "execute success",
		},
		{
			lang:    language.English, // 未知errcode, 则不翻译,即为空字符串
			errCode: 0,
			want:    "",
		},
		{
			lang:    language.Japanese, // 未知的语言, 走默认值, 即英语
			errCode: api.ExecuteSuccess,
			want:    "execute success",
		},
	}
	for _, testItem := range tests {
		if msg, _ := Translate(testItem.errCode, testItem.lang.String()); msg != testItem.want {
			t.Errorf("i18.Translate() = %v, want %v", msg, testItem.want)
		}
	}

}

package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/polarismesh/polaris-server/apiserver/httpserver/i18n"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

func init() {
	i18n.LoadI18nMessageFile("i18n/en.toml")
	i18n.LoadI18nMessageFile("i18n/zh.toml")
}

func Test_i18n(t *testing.T) {
	type args struct {
		hlang string // lang in header
		qlang string // lang in query
		rMsg  string // 实际响应体的resp.info
		hMsg  string // resp header 的msg
		wMsg  string // 期望的msg
	}
	code, codeEnMsg, codeZhMsg := uint32(200000), "execute success", "执行成功"

	testCases := []args{
		// 不支持的语言, 默认英语
		{qlang: "ja", hlang: "ja", hMsg: codeEnMsg, rMsg: codeEnMsg, wMsg: codeEnMsg},
		// 优先处理query指定, 否则按照header走
		{qlang: "en", hlang: "zh", hMsg: codeEnMsg, rMsg: codeEnMsg, wMsg: codeEnMsg},
		{qlang: "", hlang: "zh", hMsg: codeEnMsg, rMsg: codeEnMsg, wMsg: codeZhMsg},
		{qlang: "zh", hlang: "en", hMsg: codeEnMsg, rMsg: codeEnMsg, wMsg: codeZhMsg},
		{qlang: "", hlang: "", hMsg: codeEnMsg, rMsg: codeEnMsg, wMsg: codeEnMsg},
		// 当header 与 resp.info 不一致, 不翻译
		{qlang: "", hlang: "", hMsg: codeEnMsg, rMsg: "another msg", wMsg: "another msg"},
		{qlang: "zh", hlang: "en", hMsg: codeEnMsg, rMsg: "another msg", wMsg: "another msg"},
		{qlang: "", hlang: "en", hMsg: codeEnMsg, rMsg: "another msg", wMsg: "another msg"},
	}
	for _, item := range testCases {
		h := Handler{}
		h.Request = restful.NewRequest(&http.Request{
			Header: map[string][]string{"Accept-Language": {item.hlang}},
			Form:   map[string][]string{"lang": {item.qlang}},
		})
		h.Response = restful.NewResponse(httptest.NewRecorder())
		h.Response.AddHeader(utils.PolarisMessage, item.hMsg)
		resp := api.NewResponse(code)
		resp.Info = &wrappers.StringValue{Value: item.rMsg}
		if msg := h.i18n(resp).GetInfo().Value; msg != item.wMsg {
			t.Errorf("handler.i18n() = %v, want %v", msg, item.wMsg)
		}
	}
}

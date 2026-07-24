package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type selfServiceChannelRepositoryStub struct {
	*merchantRepositoryStub
	create func(context.Context, StoreMerchantChannel) (MerchantChannel, error)
}

type manualModelCheckRepositoryStub struct {
	*merchantRepositoryStub
	channel     MerchantChannel
	models      []ChannelModelBinding
	savedState  ModelHealthState
	savedClass  healthFailureClass
	savedError  string
	saveInvoked bool
}

func (s *manualModelCheckRepositoryStub) Channel(context.Context, int64) (MerchantChannel, error) {
	return s.channel, nil
}

func (s *manualModelCheckRepositoryStub) SyncChannelModels(context.Context, int64, []SyncedChannelModel) error {
	return nil
}

func (s *manualModelCheckRepositoryStub) ChannelModels(context.Context, int64) ([]ChannelModelBinding, error) {
	return s.models, nil
}

func (s *manualModelCheckRepositoryStub) SaveManualModelCheck(_ context.Context, _, bindingID, _ int64, state ModelHealthState, class healthFailureClass, latencyMS int, summary string) (ChannelModelBinding, error) {
	s.savedState = state
	s.savedClass = class
	s.savedError = summary
	s.saveInvoked = true
	return ChannelModelBinding{ID: bindingID, ChannelID: s.channel.ID, CanonicalModelID: 61, Model: "gpt-5.5", UpstreamModelName: "gpt-5.5", MatchStatus: "matched", HealthState: state.State, ConsecutiveRequestFailures: state.ConsecutiveRequestFailures, AutomaticCheckFailures: state.AutomaticCheckFailures, LastCheckLatencyMS: latencyMS, LastCheckError: summary}, nil
}

type manualModelTesterStub struct {
	called  bool
	outcome ChannelTestOutcome
	err     error
}

func (s *manualModelTesterStub) Test(context.Context, ChannelTestRequest) (ChannelTestOutcome, error) {
	return ChannelTestOutcome{}, nil
}

func (s *manualModelTesterStub) TestExisting(_ context.Context, channelID int, model CanonicalModel) (ChannelTestOutcome, error) {
	s.called = true
	if channelID != 41 || model.ID != 61 {
		return ChannelTestOutcome{}, fmt.Errorf("检查参数错误：channel=%d model=%d", channelID, model.ID)
	}
	return s.outcome, s.err
}

func (s *selfServiceChannelRepositoryStub) CreateSelfServiceChannel(ctx context.Context, command StoreMerchantChannel) (MerchantChannel, error) {
	return s.create(ctx, command)
}

type selfServiceAdminStub struct {
	createdModels string
	channelID     int
}

func (s *selfServiceAdminStub) Do(_ context.Context, method, path string, body any) ([]byte, error) {
	switch {
	case method == http.MethodPost && path == "/api/channel/fetch_models":
		return []byte(`{"data":[{"id":"gpt-5.5"},{"id":"gpt-5.6"}]}`), nil
	case method == http.MethodPost && path == "/api/channel/":
		encoded, _ := json.Marshal(body)
		var request struct {
			Channel struct {
				Models string `json:"models"`
			} `json:"channel"`
		}
		_ = json.Unmarshal(encoded, &request)
		s.createdModels = request.Channel.Models
		return []byte(`null`), nil
	case method == http.MethodGet && strings.HasPrefix(path, "/api/channel/search?"):
		return []byte(fmt.Sprintf(`{"items":[{"id":%d}]}`, s.channelID)), nil
	case method == http.MethodDelete && path == fmt.Sprintf("/api/channel/%d", s.channelID):
		return []byte(`null`), nil
	default:
		return nil, fmt.Errorf("未知管理调用 %s %s", method, path)
	}
}

func TestMerchantCreatesFormalChannelWithPendingModelsAfterSync(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	admin := &selfServiceAdminStub{channelID: 41}
	services.AdminNewAPI = admin
	services.Routing = &routingRepositoryStub{listModels: func(context.Context) ([]CanonicalModel, error) {
		return []CanonicalModel{
			{ID: 61, PublicName: "gpt-5.5", Provider: "OpenAI", Status: "active", Price: OfficialPrice{PriceKind: "text"}},
			{ID: 62, PublicName: "gpt-5.6", Provider: "OpenAI", Status: "active", Price: OfficialPrice{PriceKind: "text"}},
		}, nil
	}}
	repository := &selfServiceChannelRepositoryStub{merchantRepositoryStub: &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
	}}}
	repository.create = func(_ context.Context, command StoreMerchantChannel) (MerchantChannel, error) {
		if command.UserID != 7 || command.NewAPIChannelID != 41 || len(command.Models) != 2 {
			t.Fatalf("正式渠道入库=%+v", command)
		}
		return MerchantChannel{ID: 51, UserID: 7, ChannelName: command.ChannelName, Provider: command.Provider, BaseURL: command.BaseURL}, nil
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, roleCommon)
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/channels", strings.NewReader(`{"channel_name":"主线路","provider":"OpenAI","base_url":"https://api.merchant.example","api_key":"sk-upstream-secret"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"id":51`) || strings.Count(response.Body.String(), `"health_state":"pending"`) != 2 {
		t.Fatalf("创建渠道响应=%d %s", response.Code, response.Body.String())
	}
	if admin.createdModels != "gpt-5.5,gpt-5.6" {
		t.Fatalf("核心渠道模型=%q", admin.createdModels)
	}
}

func TestBulkChannelTestEndpointIsRemoved(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/channel-tests", nil)
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound && response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("批量检测旧接口状态码=%d", response.Code)
	}
}

func TestMerchantManualModelCheckRestoresHealthyState(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &manualModelCheckRepositoryStub{
		merchantRepositoryStub: &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
			return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
		}},
		channel: MerchantChannel{ID: 51, MerchantID: 31, NewAPIChannelID: 41},
		models: []ChannelModelBinding{{ID: 71, ChannelID: 51, CanonicalModelID: 61, HealthState: "error", ConsecutiveRequestFailures: 3, AutomaticCheckFailures: 10}},
	}
	tester := &manualModelTesterStub{outcome: ChannelTestOutcome{LatencyMS: 38}}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	services.ChannelTests = tester
	services.Routing = &routingRepositoryStub{listModels: func(context.Context) ([]CanonicalModel, error) {
		return []CanonicalModel{{ID: 61, PublicName: "gpt-5.5", Provider: "OpenAI", Status: "active", Price: OfficialPrice{PriceKind: "text"}}}, nil
	}}
	request := merchantModelCheckRequest(t, services, 7, 51, 71)
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !tester.called || !repository.saveInvoked {
		t.Fatalf("手动检查响应=%d %s tester=%t save=%t", response.Code, response.Body.String(), tester.called, repository.saveInvoked)
	}
	if repository.savedState.State != "healthy" || repository.savedState.ConsecutiveRequestFailures != 0 || repository.savedState.AutomaticCheckFailures != 0 {
		t.Fatalf("恢复状态=%+v", repository.savedState)
	}
}

func TestMerchantManualModelCheckFailureMovesModelToError(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &manualModelCheckRepositoryStub{
		merchantRepositoryStub: &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
			return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
		}},
		channel: MerchantChannel{ID: 51, MerchantID: 31, NewAPIChannelID: 41},
		models:  []ChannelModelBinding{{ID: 71, ChannelID: 51, CanonicalModelID: 61, HealthState: "cooling", ConsecutiveRequestFailures: 3, AutomaticCheckFailures: 4}},
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	services.ChannelTests = &manualModelTesterStub{err: errors.New("upstream timeout")}
	services.Routing = &routingRepositoryStub{listModels: func(context.Context) ([]CanonicalModel, error) {
		return []CanonicalModel{{ID: 61, PublicName: "gpt-5.5", Provider: "OpenAI", Status: "active", Price: OfficialPrice{PriceKind: "text"}}}, nil
	}}
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, merchantModelCheckRequest(t, services, 7, 51, 71))
	if response.Code != http.StatusUnprocessableEntity || repository.savedState.State != "error" || repository.savedClass != healthFailureTemporary || repository.savedError == "" {
		t.Fatalf("失败检查响应=%d %s state=%+v class=%q error=%q", response.Code, response.Body.String(), repository.savedState, repository.savedClass, repository.savedError)
	}
}

func TestMerchantCannotCheckAnotherMerchantsChannelModel(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &manualModelCheckRepositoryStub{
		merchantRepositoryStub: &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
			return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
		}},
		channel: MerchantChannel{ID: 51, MerchantID: 32, NewAPIChannelID: 41},
	}
	tester := &manualModelTesterStub{}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	services.ChannelTests = tester
	services.Routing = &routingRepositoryStub{}
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, merchantModelCheckRequest(t, services, 7, 51, 71))
	if response.Code != http.StatusNotFound || tester.called || repository.saveInvoked {
		t.Fatalf("跨商户检查响应=%d tester=%t save=%t", response.Code, tester.called, repository.saveInvoked)
	}
}

func merchantModelCheckRequest(t *testing.T, services Services, userID int, channelID, bindingID int64) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/merchant/channels/%d/models/%d/check", channelID, bindingID), nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: createPlatformSession(t, services, userID, roleCommon)})
	return request
}

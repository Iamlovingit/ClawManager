package services

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"clawreef/internal/models"
)

func TestCollectUpstreamSetCookies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "hermes_session_at=session-127; Path=/proxy; HttpOnly")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	resp, err := srv.Client().Post(srv.URL+"/auth/password-login", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	got := collectUpstreamSetCookies(resp)
	if len(got) == 0 {
		t.Fatalf("no cookies; Values=%v Map=%v Cookies=%v", resp.Header.Values("Set-Cookie"), resp.Header["Set-Cookie"], resp.Cookies())
	}
}

func TestIsHermesLiteProxyInstanceMatchesTokenInjectionSetup(t *testing.T) {
	instanceToken := "igt_hermes_instance"
	workspacePath := "/workspaces/hermes/user-45/instance-127"
	instanceRepo := newV2LifecycleInstanceRepo()
	instanceRepo.byID[127] = &models.Instance{
		ID:                127,
		UserID:            45,
		Type:              "hermes",
		RuntimeType:       "gateway",
		InstanceMode:      InstanceModeLite,
		Status:            "running",
		AccessToken:       &instanceToken,
		WorkspacePath:     &workspacePath,
		RuntimeGeneration: 5,
	}
	service := NewInstanceProxyService(NewInstanceAccessService())
	service.instanceRepo = instanceRepo
	if !service.isHermesLiteProxyInstance(127, "hermes") {
		inst, _ := instanceRepo.GetByID(127)
		rt, ok := v2RuntimeTypeForInstance(inst)
		t.Fatalf("isHermesLiteProxyInstance=false; runtimeType=%q ok=%v mode=%q runtime=%q", rt, ok, inst.InstanceMode, inst.RuntimeType)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/127/proxy/chat", nil)
	if !shouldBootstrapHermesDashboardSession(req, "/chat") {
		t.Fatal("shouldBootstrap=false")
	}
}

package operations

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MCP, model and skill writes are admin-only, and none of them can carry a
// credential: secrets stay on the pages where the user types them.
func TestBuiltinInfrastructureWrites(t *testing.T) {
	catalog := Builtin()
	cases := []struct {
		id     string
		risk   Risk
		method string
		params []string
	}{
		{
			"mcp_service.create", RiskWrite, http.MethodPost,
			[]string{"name", "usage_instructions", "transport_type", "url", "auth_type", "api_key_header"},
		},
		{
			"mcp_service.update", RiskWrite, http.MethodPut,
			[]string{"mcp_service_id", "name", "usage_instructions", "enabled", "transport_type", "url"},
		},
		{"mcp_service.delete", RiskDestructive, http.MethodDelete, []string{"mcp_service_id"}},
		{"model.copy", RiskWrite, http.MethodPost, []string{"model_id", "display_name"}},
		{"model.update", RiskWrite, http.MethodPatch, []string{"model_id", "display_name", "description"}},
		{"model.delete", RiskDestructive, http.MethodDelete, []string{"model_id"}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			op, ok := catalog.Get(tc.id)
			require.True(t, ok)
			assert.Equal(t, tc.risk, op.Risk)
			assert.Equal(t, tc.method, op.Method)
			assert.Equal(t, types.TenantRoleAdmin, op.Role)
			assert.ElementsMatch(t, tc.params, paramNames(op))
			for _, p := range op.Params {
				secrets := []string{"api_key", "token", "client_secret", "headers", "custom_headers", "envs"}
				assert.NotContains(t, secrets, p.Name)
			}
		})
	}

	create, _ := catalog.Get("mcp_service.create")
	transport, _ := create.Param("transport_type")
	assert.ElementsMatch(t, []string{"sse", "http-streamable"}, transport.Enum, "stdio is never offered")
	auth, _ := create.Param("auth_type")
	assert.Equal(t, "auth_config.auth_type", auth.Key)

}

func TestBuiltinInstallsAlwaysAsk(t *testing.T) {
	catalog := Builtin()
	op, ok := catalog.Get("mcp_service.create")
	require.True(t, ok)
	assert.True(t, op.AlwaysConfirm)
	assert.True(t, Access{Level: types.AssistantAccessAutoWrite}.NeedsApproval(op))
	// Skill install operations were removed from this fork together with their
	// HTTP routes (MGI build).
	_, ok = catalog.Get("skill.install")
	assert.False(t, ok)
}

func TestBuiltinInfrastructureProjections(t *testing.T) {
	catalog := Builtin()
	project := func(id, body string) string {
		t.Helper()
		op, ok := catalog.Get(id)
		require.True(t, ok, id)
		projected, err := op.Project([]byte(body))
		require.NoError(t, err)
		out, err := json.Marshal(projected.Data)
		require.NoError(t, err)
		return string(out)
	}

	assert.JSONEq(t, `{"mcp_service_id":"m1","name":"Docs","description":"","enabled":true,
		"transport_type":"sse","is_builtin":false,"auth_type":"api_key"}`,
		project("mcp_service.get", `{"data":{"id":"m1","name":"Docs","description":"","enabled":true,
		"transport_type":"sse","is_builtin":false,"url":"https://mcp.example/sse",
		"auth_config":{"auth_type":"api_key","api_key_header":"X-Key","custom_headers":{"Authorization":"x"}},
		"credentials":{"api_key":{"configured":true}}}}`))

}

func TestModelCopyOnlyAcceptsSourceAndNewDisplayName(t *testing.T) {
	op, ok := Builtin().Get("model.copy")
	require.True(t, ok)
	require.Equal(t, "model.get", op.Target.Operation)
	call, err := op.Build(map[string]json.RawMessage{"model_id": raw("source"), "display_name": raw("Experiment")})
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/models/source/copy", call.URI())
	assert.JSONEq(t, `{"display_name":"Experiment"}`, string(call.Body))
	for _, params := range []map[string]json.RawMessage{
		{"model_id": raw("source")},
		{"model_id": raw("source"), "display_name": raw("")},
		{"model_id": raw("source"), "display_name": raw("Experiment"), "api_key": raw("secret")},
		{"model_id": raw("source"), "display_name": raw("Experiment"), "name": raw("another-model")},
		{"model_id": raw("source"), "display_name": raw("Experiment"), "is_default": raw(true)},
	} {
		_, err = op.Build(params)
		require.Error(t, err)
	}
}

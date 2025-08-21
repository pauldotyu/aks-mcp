package k8s

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
	k8ssecurity "github.com/Azure/mcp-kubernetes/pkg/security"
	k8stools "github.com/Azure/mcp-kubernetes/pkg/tools"
)

// This test suite verifies config mapping (without mutating input), adapter delegation,
// error propagation, and the current nil-config behavior. Benchmarks provide a baseline
// for detecting performance regressions.

// mustEqual keeps assertions concise with consistent failure messages.
func mustEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

// mustDeepEqual keeps deep-structure assertions concise with consistent messages.
func mustDeepEqual(t *testing.T, got, want interface{}, msg string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", msg, got, want)
	}
}

// fakeExecutor captures inputs and returns preset output/error to observe delegation.
type fakeExecutor struct {
	lastParams map[string]interface{}
	lastCfg    *k8sconfig.ConfigData
	out        string
	err        error
}

var _ k8stools.CommandExecutor = (*fakeExecutor)(nil)

func (f *fakeExecutor) Execute(params map[string]interface{}, cfg *k8sconfig.ConfigData) (string, error) {
	f.lastParams = params
	f.lastCfg = cfg
	return f.out, f.err
}

func TestConvertConfig_MapsFields(t *testing.T) {
	t.Parallel()

	in := &config.ConfigData{
		Timeout:         600,
		Transport:       "stdio",
		Host:            "127.0.0.1",
		Port:            8000,
		AccessLevel:     "readonly",
		AdditionalTools: map[string]bool{"helm": true, "cilium": false},
		AllowNamespaces: "default,platform",
		OTLPEndpoint:    "otel:4317",
	}

	got := ConvertConfig(in)
	if got == nil {
		t.Fatal("ConvertConfig returned nil")
	}

	mustEqual(t, got.Timeout, in.Timeout, "Timeout")
	mustEqual(t, got.Transport, in.Transport, "Transport")
	mustEqual(t, got.Host, in.Host, "Host")
	mustEqual(t, got.Port, in.Port, "Port")
	mustEqual(t, got.AccessLevel, in.AccessLevel, "AccessLevel")
	mustEqual(t, got.OTLPEndpoint, in.OTLPEndpoint, "OTLPEndpoint")
	mustDeepEqual(t, got.AdditionalTools, in.AdditionalTools, "AdditionalTools")
	mustEqual(t, got.AllowNamespaces, in.AllowNamespaces, "AllowNamespaces")

	if got.SecurityConfig == nil {
		t.Fatal("SecurityConfig is nil")
	}
	mustEqual(t, got.SecurityConfig.AccessLevel, k8ssecurity.AccessLevel(in.AccessLevel), "SecurityConfig.AccessLevel")
}

func TestConvertConfig_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	in := &config.ConfigData{
		Timeout:         42,
		Transport:       "stdio",
		Host:            "127.0.0.1",
		Port:            8000,
		AccessLevel:     "readonly",
		AdditionalTools: map[string]bool{"helm": true},
		AllowNamespaces: "default",
		OTLPEndpoint:    "otel:4317",
	}

	// Verify the “no input mutation” guarantee by comparing to a copy.
	orig := *in
	orig.AdditionalTools = map[string]bool{}
	for k, v := range in.AdditionalTools {
		orig.AdditionalTools[k] = v
	}

	_ = ConvertConfig(in)
	mustDeepEqual(t, in, &orig, "input should remain unchanged")
}

func TestConvertConfig_ZeroValueCfg(t *testing.T) {
	t.Parallel()
	// Zero-value config should be accepted (no panic).
	in := &config.ConfigData{}
	_ = ConvertConfig(in)
}

func TestExecutorAdapter_DelegatesAndForwards(t *testing.T) {
	t.Parallel()

	fe := &fakeExecutor{out: "ok"}
	adapter := WrapK8sExecutor(fe)

	params := map[string]interface{}{"k": "v"}
	inCfg := &config.ConfigData{
		Timeout:         10,
		Transport:       "stdio",
		Host:            "127.0.0.1",
		Port:            8000,
		AccessLevel:     "readonly",
		AdditionalTools: map[string]bool{"helm": true},
		AllowNamespaces: "default",
	}

	got, err := adapter.Execute(params, inCfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustEqual(t, got, "ok", "adapter output")
	mustDeepEqual(t, fe.lastParams, params, "params forwarded")

	if fe.lastCfg == nil || fe.lastCfg.SecurityConfig == nil {
		t.Fatalf("expected non-nil converted cfg + SecurityConfig, got %#v", fe.lastCfg)
	}
	mustEqual(t, fe.lastCfg.Port, inCfg.Port, "Port")
	mustEqual(t, fe.lastCfg.AccessLevel, inCfg.AccessLevel, "AccessLevel")
	mustDeepEqual(t, fe.lastCfg.AdditionalTools, inCfg.AdditionalTools, "AdditionalTools")
	mustEqual(t, fe.lastCfg.AllowNamespaces, inCfg.AllowNamespaces, "AllowNamespaces")
	mustEqual(t, fe.lastCfg.SecurityConfig.AccessLevel, k8ssecurity.AccessLevel("readonly"), "SecurityConfig.AccessLevel")
}

func TestExecutorAdapter_PropagatesError(t *testing.T) {
	t.Parallel()

	fe := &fakeExecutor{err: errors.New("boom")}
	adapter := WrapK8sExecutor(fe)

	_, err := adapter.Execute(map[string]interface{}{"x": 1}, &config.ConfigData{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExecutorAdapter_PanicsOnNilConfig_CurrentBehavior(t *testing.T) {
	t.Parallel()

	// Document the current precondition: cfg must be non-nil.
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when cfg is nil")
		}
	}()

	fe := &fakeExecutor{}
	adapter := WrapK8sExecutor(fe)
	_, _ = adapter.Execute(map[string]interface{}{"x": 1}, nil)
}

// BenchmarkConvertConfig tracks drift in allocation/time costs over time.
// Helps detect subtle regressions when config mapping logic evolves.
func BenchmarkConvertConfig(b *testing.B) {
	in := &config.ConfigData{
		Timeout:         600,
		Transport:       "stdio",
		Host:            "127.0.0.1",
		Port:            8000,
		AccessLevel:     "readonly",
		AdditionalTools: map[string]bool{"helm": true, "cilium": false},
		AllowNamespaces: "default,platform",
		OTLPEndpoint:    "otel:4317",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertConfig(in)
	}
}

// BenchmarkExecutorAdapter measures adapter overhead to ensure delegation
// stays cheap and doesn’t become a bottleneck as layers evolve.
func BenchmarkExecutorAdapter(b *testing.B) {
	fe := &fakeExecutor{out: "ok"}
	adapter := WrapK8sExecutor(fe)
	inCfg := &config.ConfigData{
		Timeout:         42,
		Transport:       "stdio",
		Host:            "127.0.0.1",
		Port:            8000,
		AccessLevel:     "readonly",
		AdditionalTools: map[string]bool{"helm": true},
		AllowNamespaces: "default",
	}

	params := map[string]interface{}{"k": "v"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = adapter.Execute(params, inCfg)
	}
}

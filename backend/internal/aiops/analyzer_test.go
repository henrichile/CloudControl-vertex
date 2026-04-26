package aiops_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/etasoft/cloudcontrol/internal/aiops"
)

// mockOllamaServer creates a test HTTP server that simulates Ollama responses.
func mockOllamaServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/generate":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"response": response,
				"done":     true,
			})
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]string{{"name": "llama3:latest"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestOllamaClient_Ping_Success(t *testing.T) {
	srv := mockOllamaServer("")
	defer srv.Close()

	client := aiops.NewClient(srv.URL, "llama3")
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestOllamaClient_Ping_ModelNotFound(t *testing.T) {
	srv := mockOllamaServer("")
	defer srv.Close()

	client := aiops.NewClient(srv.URL, "modelo-que-no-existe")
	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("Ping() should fail when model not found")
	}
}

func TestOllamaClient_Generate(t *testing.T) {
	srv := mockOllamaServer(`{"action": "ok", "reason": "recursos estables"}`)
	defer srv.Close()

	client := aiops.NewClient(srv.URL, "llama3")
	resp, err := client.Generate(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp == "" {
		t.Error("Generate() returned empty response")
	}
}

func TestAnalyzer_AnalyzeMetrics_ScaleUp(t *testing.T) {
	srv := mockOllamaServer(`{"action": "scale_up", "reason": "CPU al 95%, necesita más recursos", "new_cpu_limit": 200, "new_mem_limit_mb": 1024}`)
	defer srv.Close()

	client := aiops.NewClient(srv.URL, "llama3")
	analyzer := aiops.NewAnalyzer(client)

	snap := aiops.MetricsSnapshot{
		ContainerName: "mi-api",
		CPUPercent:    95.5,
		MemUsageMB:    900,
		MemLimitMB:    1024,
		NetRxMB:       10,
		NetTxMB:       5,
	}

	rec, err := analyzer.AnalyzeMetrics(context.Background(), snap)
	if err != nil {
		t.Fatalf("AnalyzeMetrics() error: %v", err)
	}
	if rec.Action != "scale_up" {
		t.Errorf("expected action 'scale_up', got %q", rec.Action)
	}
	if rec.Reason == "" {
		t.Error("Reason should not be empty")
	}
	if rec.RawAnalysis == "" {
		t.Error("RawAnalysis should not be empty")
	}
}

func TestAnalyzer_AnalyzeMetrics_OK(t *testing.T) {
	srv := mockOllamaServer(`{"action": "ok", "reason": "todo está bien", "new_cpu_limit": null, "new_mem_limit_mb": null}`)
	defer srv.Close()

	client := aiops.NewClient(srv.URL, "llama3")
	analyzer := aiops.NewAnalyzer(client)

	snap := aiops.MetricsSnapshot{
		ContainerName: "db",
		CPUPercent:    15.0,
		MemUsageMB:    256,
		MemLimitMB:    1024,
	}

	rec, err := analyzer.AnalyzeMetrics(context.Background(), snap)
	if err != nil {
		t.Fatalf("AnalyzeMetrics() error: %v", err)
	}
	if rec.Action != "ok" {
		t.Errorf("expected action 'ok', got %q", rec.Action)
	}
}

func TestAnalyzer_AuditConfig_FindsIssues(t *testing.T) {
	securityResponse := `{
		"score": 35,
		"findings": [
			{"severity": "critical", "finding": "Contraseña en texto plano", "suggestion": "Usa secrets de Docker", "line_number": 5},
			{"severity": "high", "finding": "Puerto de base de datos expuesto", "suggestion": "Elimina el mapeo de puerto 5432", "line_number": 12}
		]
	}`
	srv := mockOllamaServer(securityResponse)
	defer srv.Close()

	client := aiops.NewClient(srv.URL, "llama3")
	analyzer := aiops.NewAnalyzer(client)

	content := `
services:
  db:
    image: postgres:latest
    environment:
      POSTGRES_PASSWORD: mysecretpassword
    ports:
      - "5432:5432"
`

	result, err := analyzer.AuditConfig(context.Background(), "docker-compose.yml", content)
	if err != nil {
		t.Fatalf("AuditConfig() error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Error("AuditConfig() should return at least one finding")
	}
	if result.Score == 0 {
		t.Error("Score should not be 0")
	}
	if result.RawAnalysis == "" {
		t.Error("RawAnalysis should not be empty")
	}
}

func TestAnalyzer_AnalyzeLogs(t *testing.T) {
	srv := mockOllamaServer("El contenedor muestra errores de conexión repetidos. Recomendación: verificar la conectividad de red.")
	defer srv.Close()

	client := aiops.NewClient(srv.URL, "llama3")
	analyzer := aiops.NewAnalyzer(client)

	logs := `2024-01-01 10:00:00 ERROR connection refused
2024-01-01 10:00:01 ERROR connection refused
2024-01-01 10:00:02 INFO retrying...`

	analysis, err := analyzer.AnalyzeLogs(context.Background(), "mi-servicio", logs)
	if err != nil {
		t.Fatalf("AnalyzeLogs() error: %v", err)
	}
	if analysis == "" {
		t.Error("AnalyzeLogs() returned empty analysis")
	}
	if !strings.Contains(analysis, "conexión") && !strings.Contains(analysis, "contenedor") {
		t.Logf("Analysis: %s", analysis)
	}
}

func TestOllamaClient_Unreachable(t *testing.T) {
	client := aiops.NewClient("http://localhost:19999", "llama3")
	ctx := context.Background()

	if err := client.Ping(ctx); err == nil {
		t.Error("Ping() to unreachable server should return error")
	}

	_, err := client.Generate(ctx, "test")
	if err == nil {
		t.Error("Generate() to unreachable server should return error")
	}
}

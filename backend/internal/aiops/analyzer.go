package aiops

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// MetricsSnapshot holds resource usage for a single container.
type MetricsSnapshot struct {
	ContainerName string
	CPUPercent    float64
	MemUsageMB    float64
	MemLimitMB    float64
	NetRxMB       float64
	NetTxMB       float64
}

// ScalingRecommendation is the structured output of a metrics analysis.
type ScalingRecommendation struct {
	Action      string  `json:"action"` // "scale_up", "scale_down", "ok"
	Reason      string  `json:"reason"`
	NewCPULimit float64 `json:"new_cpu_limit,omitempty"` // percent (0-100)
	NewMemLimMB float64 `json:"new_mem_limit_mb,omitempty"`
	RawAnalysis string  `json:"raw_analysis"`
}

// SecurityFinding represents a single vulnerability or misconfiguration.
type SecurityFinding struct {
	Severity   string `json:"severity"`
	Finding    string `json:"finding"`
	Suggestion string `json:"suggestion"`
	LineNumber int    `json:"line_number,omitempty"`
}

// SecurityAuditResult is the output of a configuration audit.
type SecurityAuditResult struct {
	Findings    []SecurityFinding `json:"findings"`
	RawAnalysis string            `json:"raw_analysis"`
	Score       int               `json:"score"` // 0-100, higher is better
}

// Analyzer performs AIOps tasks using the Ollama client.
type Analyzer struct {
	ollama *Client
}

func NewAnalyzer(ollama *Client) *Analyzer {
	return &Analyzer{ollama: ollama}
}

// AnalyzeMetrics sends container metrics to Ollama and returns a scaling recommendation.
func (a *Analyzer) AnalyzeMetrics(ctx context.Context, snap MetricsSnapshot) (*ScalingRecommendation, error) {
	memPercent := 0.0
	if snap.MemLimitMB > 0 {
		memPercent = (snap.MemUsageMB / snap.MemLimitMB) * 100
	}

	prompt := fmt.Sprintf(`Eres un ingeniero DevOps experto analizando el uso de recursos de contenedores Docker.
Analiza las siguientes métricas y proporciona una recomendación de escalamiento.

Contenedor: %s
Uso de CPU: %.2f%%
Uso de Memoria: %.2f MB / %.2f MB (%.1f%%)
Red Entrada: %.2f MB
Red Salida: %.2f MB

Basándote en estas métricas:
1. Determina si el contenedor necesita escalar HACIA ARRIBA, HACIA ABAJO, u OK como está.
2. Si se necesita escalar, proporciona límites nuevos específicos.
3. Explica el razonamiento brevemente.

Responde en este formato JSON exacto (sin markdown, sin texto extra):
{
  "action": "scale_up|scale_down|ok",
  "reason": "explicación breve",
  "new_cpu_limit": <número o null>,
  "new_mem_limit_mb": <número o null>
}`,
		snap.ContainerName, snap.CPUPercent,
		snap.MemUsageMB, snap.MemLimitMB, memPercent,
		snap.NetRxMB, snap.NetTxMB,
	)

	raw, err := a.ollama.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("ollama metrics analysis failed: %w", err)
	}

	rec := parseScalingRecommendation(raw)
	rec.RawAnalysis = raw
	return rec, nil
}

// AuditConfig sends a config file to Ollama for security analysis.
func (a *Analyzer) AuditConfig(ctx context.Context, fileName, content string) (*SecurityAuditResult, error) {
	prompt := fmt.Sprintf(`Eres un experto en seguridad auditando archivos de configuración de Docker e infraestructura.
Analiza el siguiente archivo en busca de vulnerabilidades de seguridad, configuraciones erróneas y violaciones de mejores prácticas.

Archivo: %s
Contenido:
---
%s
---

Verifica:
- Secretos o credenciales expuestos en variables de entorno
- Exposiciones de puerto peligrosas (ej: puertos de bases de datos expuestos a 0.0.0.0)
- Imágenes sin versiones fijadas (usando etiqueta :latest)
- Ejecución como usuario root
- Límites de recursos faltantes
- Configuraciones de red inseguras
- Opciones de seguridad faltantes (no-new-privileges, filesystem read-only)
- Imágenes base desactualizadas o vulnerables

Para cada hallazgo, proporciona:
- severity: critical|high|medium|low|info
- finding: cuál es el problema
- suggestion: cómo solucionarlo
- line_number: línea aproximada en el archivo (o 0 si no aplica)

Responde en este formato JSON exacto (sin markdown):
{
  "score": <0-100>,
  "findings": [
    {"severity": "high", "finding": "...", "suggestion": "...", "line_number": 5}
  ]
}`,
		fileName, content,
	)

	raw, err := a.ollama.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("ollama security audit failed: %w", err)
	}

	result := parseSecurityAudit(raw)
	result.RawAnalysis = raw
	return result, nil
}

// AnalyzeLogs sends recent container logs to Ollama for anomaly detection.
func (a *Analyzer) AnalyzeLogs(ctx context.Context, containerName, logs string) (string, error) {
	if len(logs) > 8000 {
		logs = logs[len(logs)-8000:] // keep the most recent portion
	}

	prompt := fmt.Sprintf(`Eres un experto DevOps analizando logs de contenedores Docker en busca de errores, advertencias y anomalías.

Contenedor: %s
Logs Recientes:
---
%s
---

Identifica:
1. Errores críticos o bloqueos
2. Advertencias de rendimiento
3. Eventos relevantes para seguridad (fallos de autenticación, acceso inesperado)
4. Patrones recurrentes que indican inestabilidad

Proporciona un resumen conciso (máx 200 palabras) con recomendaciones accionables.`,
		containerName, logs,
	)

	return a.ollama.Generate(ctx, prompt)
}

// parseScalingRecommendation extracts structured data from Ollama's JSON response.
func parseScalingRecommendation(raw string) *ScalingRecommendation {
	rec := &ScalingRecommendation{Action: "ok", Reason: raw}

	jsonStr := extractJSON(raw)
	if jsonStr == "" {
		return rec
	}

	// Simple field extraction without a full JSON parser to avoid circular deps.
	if strings.Contains(jsonStr, `"scale_up"`) {
		rec.Action = "scale_up"
	} else if strings.Contains(jsonStr, `"scale_down"`) {
		rec.Action = "scale_down"
	} else {
		rec.Action = "ok"
	}

	if r := extractStringField(jsonStr, "reason"); r != "" {
		rec.Reason = r
	}

	rec.NewCPULimit = extractFloatField(jsonStr, "new_cpu_limit")
	rec.NewMemLimMB = extractFloatField(jsonStr, "new_mem_limit_mb")

	return rec
}

func parseSecurityAudit(raw string) *SecurityAuditResult {
	result := &SecurityAuditResult{Score: 50}

	jsonStr := extractJSON(raw)
	if jsonStr == "" {
		result.Findings = []SecurityFinding{{
			Severity:   "info",
			Finding:    "Could not parse structured response",
			Suggestion: raw,
		}}
		return result
	}

	// Extract score
	if score := extractFloatField(jsonStr, "score"); score > 0 {
		result.Score = int(score)
	}

	// For full parsing in production, use encoding/json with the proper struct.
	// This lightweight version handles the common case.
	severities := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range severities {
		if strings.Contains(strings.ToLower(jsonStr), fmt.Sprintf(`"severity": "%s"`, sev)) ||
			strings.Contains(strings.ToLower(jsonStr), fmt.Sprintf(`"severity":"%s"`, sev)) {
			finding := SecurityFinding{Severity: sev}
			if f := extractStringField(jsonStr, "finding"); f != "" {
				finding.Finding = f
			}
			if s := extractStringField(jsonStr, "suggestion"); s != "" {
				finding.Suggestion = s
			}
			result.Findings = append(result.Findings, finding)
		}
	}

	if len(result.Findings) == 0 {
		result.Findings = []SecurityFinding{{
			Severity:   "info",
			Finding:    "No specific findings extracted",
			Suggestion: raw,
		}}
	}

	return result
}

var jsonBlockRe = regexp.MustCompile(`(?s)\{.*\}`)

func extractJSON(s string) string {
	match := jsonBlockRe.FindString(s)
	return match
}

func extractStringField(json, field string) string {
	patterns := []string{
		fmt.Sprintf(`"%s": "([^"]*)"`, field),
		fmt.Sprintf(`"%s":"([^"]*)"`, field),
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if m := re.FindStringSubmatch(json); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func extractFloatField(json, field string) float64 {
	patterns := []string{
		fmt.Sprintf(`"%s": ([0-9.]+)`, field),
		fmt.Sprintf(`"%s":([0-9.]+)`, field),
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if m := re.FindStringSubmatch(json); len(m) > 1 {
			var f float64
			fmt.Sscanf(m[1], "%f", &f)
			return f
		}
	}
	return 0
}

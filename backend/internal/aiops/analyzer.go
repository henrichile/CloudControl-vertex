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
	Action      string  `json:"action"`      // "scale_up", "scale_down", "ok"
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

	prompt := fmt.Sprintf(`You are an expert DevOps engineer analyzing Docker container resource usage.
Analyze the following metrics and provide a scaling recommendation.

Container: %s
CPU Usage: %.2f%%
Memory Usage: %.2f MB / %.2f MB (%.1f%%)
Network In: %.2f MB
Network Out: %.2f MB

Based on these metrics:
1. Determine if the container needs resources scaled UP, DOWN, or is OK as-is.
2. If scaling is needed, provide specific new limits.
3. Explain the reasoning briefly.

Respond in this exact JSON format (no markdown, no extra text):
{
  "action": "scale_up|scale_down|ok",
  "reason": "brief explanation",
  "new_cpu_limit": <number or null>,
  "new_mem_limit_mb": <number or null>
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
	prompt := fmt.Sprintf(`You are a security expert auditing Docker and infrastructure configuration files.
Analyze the following file for security vulnerabilities, misconfigurations, and best-practice violations.

File: %s
Content:
---
%s
---

Check for:
- Exposed secrets or credentials in environment variables
- Dangerous port exposures (e.g., database ports exposed to 0.0.0.0)
- Images without pinned versions (using :latest tag)
- Running as root user
- Missing resource limits
- Insecure network configurations
- Missing security options (no-new-privileges, read-only filesystem)
- Outdated or vulnerable base images

For each finding, provide:
- severity: critical|high|medium|low|info
- finding: what the issue is
- suggestion: how to fix it
- line_number: approximate line in the file (or 0 if not applicable)

Respond in this exact JSON format (no markdown):
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

	prompt := fmt.Sprintf(`You are a DevOps expert analyzing Docker container logs for errors, warnings, and anomalies.

Container: %s
Recent Logs:
---
%s
---

Identify:
1. Critical errors or crashes
2. Performance warnings
3. Security-relevant events (failed auth, unexpected access)
4. Recurring patterns that indicate instability

Provide a concise summary (max 200 words) with actionable recommendations.`,
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
			Severity: "info",
			Finding:  "No specific findings extracted",
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

package commands

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func newAIOpsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aiops",
		Short: "Análisis inteligente de contenedores con IA (Ollama)",
	}

	cmd.AddCommand(
		aiopsAnalyzeCmd(),
		aiopsAuditCmd(),
		aiopsLogsCmd(),
	)

	return cmd
}

func aiopsAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze <container-id|nombre>",
		Short: "Analizar métricas de un contenedor con IA y sugerir escalado",
		Example: `  cloudctl aiops analyze my-api
  cloudctl aiops analyze abc123def456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Analizando métricas de '%s' con IA...\n", args[0])

			body := map[string]string{"container_id": args[0]}
			data, code, err := apiClient(http.MethodPost, "/api/v1/aiops/analyze", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			printJSON(data)
			return nil
		},
	}
}

func aiopsAuditCmd() *cobra.Command {
	var projectID, filePath string

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Auditar un archivo de configuración con IA",
		Example: `  cloudctl aiops audit --file docker-compose.yml
  cloudctl aiops audit --file .env --project-id abc123
  cloudctl aiops audit --file nginx/default.conf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("--file es requerido")
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("no se pudo leer %s: %w", filePath, err)
			}

			fmt.Printf("Auditando '%s' con IA...\n", filePath)

			body := map[string]string{
				"file_name":  filePath,
				"content":    string(content),
				"project_id": projectID,
			}

			data, code, err := apiClient(http.MethodPost, "/api/v1/aiops/audit", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			printJSON(data)
			return nil
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Ruta al archivo a auditar (requerido)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "ID del proyecto para guardar los hallazgos")

	return cmd
}

func aiopsLogsCmd() *cobra.Command {
	var tail string

	cmd := &cobra.Command{
		Use:   "logs <container-id|nombre>",
		Short: "Analizar logs recientes de un contenedor con IA",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Analizando logs de '%s' con IA...\n", args[0])

			body := map[string]string{
				"container_id": args[0],
				"tail":         tail,
			}

			data, code, err := apiClient(http.MethodPost, "/api/v1/aiops/logs", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			printJSON(data)
			return nil
		},
	}

	cmd.Flags().StringVarP(&tail, "tail", "n", "200", "Número de líneas de log a analizar")
	return cmd
}

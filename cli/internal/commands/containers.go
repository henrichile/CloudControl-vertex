package commands

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func newContainersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "containers",
		Aliases: []string{"c", "container"},
		Short:   "Gestionar contenedores Docker",
	}

	cmd.AddCommand(
		containersListCmd(),
		containersStartCmd(),
		containersStopCmd(),
		containersRemoveCmd(),
		containersLogsCmd(),
		containersStatsCmd(),
		containersLimitsCmd(),
	)

	return cmd
}

func containersListCmd() *cobra.Command {
	var onlyRunning bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "ps"},
		Short:   "Listar contenedores",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/v1/containers"
			if onlyRunning {
				path += "?running=true"
			}
			data, code, err := apiClient(http.MethodGet, path, nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			printJSON(data)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&onlyRunning, "running", "r", false, "Mostrar solo contenedores en ejecución")
	return cmd
}

func containersStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <id|nombre>",
		Short: "Iniciar un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, code, err := apiClient(http.MethodPost, "/api/v1/containers/"+args[0]+"/start", nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Contenedor %s iniciado.\n", args[0])
			return nil
		},
	}
}

func containersStopCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "stop <id|nombre>",
		Short: "Detener un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]int{"timeout": timeout}
			data, code, err := apiClient(http.MethodPost, "/api/v1/containers/"+args[0]+"/stop", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Contenedor %s detenido.\n", args[0])
			return nil
		},
	}

	cmd.Flags().IntVarP(&timeout, "timeout", "t", 10, "Timeout de parada en segundos")
	return cmd
}

func containersRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "remove <id|nombre>",
		Aliases: []string{"rm"},
		Short:   "Eliminar un contenedor",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/v1/containers/" + args[0]
			if force {
				path += "?force=true"
			}
			data, code, err := apiClient(http.MethodDelete, path, nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Contenedor %s eliminado.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Forzar eliminación aunque esté en ejecución")
	return cmd
}

func containersLogsCmd() *cobra.Command {
	var tail string
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs <id|nombre>",
		Short: "Ver logs de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := fmt.Sprintf("/api/v1/containers/%s/logs?tail=%s", args[0], tail)
			if follow {
				path += "&follow=true"
			}
			data, code, err := apiClient(http.MethodGet, path, nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVarP(&tail, "tail", "n", "100", "Número de líneas a mostrar")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Seguir el stream de logs")
	return cmd
}

func containersStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats <id|nombre>",
		Short: "Ver métricas de recursos de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, code, err := apiClient(http.MethodGet, "/api/v1/containers/"+args[0]+"/stats", nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			printJSON(data)
			return nil
		},
	}
}

func containersLimitsCmd() *cobra.Command {
	var cpuQuota int64
	var memMB int64

	cmd := &cobra.Command{
		Use:   "limits <id|nombre>",
		Short: "Actualizar límites de CPU/RAM de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]int64{
				"cpu_quota": cpuQuota,
				"memory_mb": memMB,
			}
			data, code, err := apiClient(http.MethodPatch, "/api/v1/containers/"+args[0]+"/limits", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Límites actualizados para %s.\n", args[0])
			printJSON(data)
			return nil
		},
	}

	cmd.Flags().Int64Var(&cpuQuota, "cpu-quota", 0, "CPU quota en microsegundos (ej: 50000 = 50% de 1 CPU)")
	cmd.Flags().Int64Var(&memMB, "mem", 0, "Límite de memoria en MB")
	return cmd
}

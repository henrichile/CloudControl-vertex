package commands

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"p", "project"},
		Short:   "Gestionar proyectos Docker Compose",
	}

	cmd.AddCommand(
		projectsListCmd(),
		projectsCreateCmd(),
		projectsUpCmd(),
		projectsDownCmd(),
		projectsDeleteCmd(),
		stacksListCmd(),
	)

	return cmd
}

func projectsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Listar proyectos",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, code, err := apiClient(http.MethodGet, "/api/v1/projects", nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			printJSON(data)
			return nil
		},
	}
}

func projectsCreateCmd() *cobra.Command {
	var name, stack, dbName, dbUser, dbPassword, appPort, domain string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Crear un nuevo proyecto a partir de un stack",
		Example: `  cloudctl projects create --name mi-api --stack fastapi-postgres --db-name mydb
  cloudctl projects create --name blog --stack MERN --port 4000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || stack == "" {
				return fmt.Errorf("--name y --stack son requeridos")
			}
			body := map[string]string{
				"name":        name,
				"stack":       stack,
				"db_name":     dbName,
				"db_user":     dbUser,
				"db_password": dbPassword,
				"app_port":    appPort,
				"domain":      domain,
			}
			data, code, err := apiClient(http.MethodPost, "/api/v1/projects", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Proyecto '%s' creado con stack '%s'.\n", name, stack)
			printJSON(data)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Nombre del proyecto (requerido)")
	cmd.Flags().StringVarP(&stack, "stack", "s", "", "Stack a usar (requerido). Ver: cloudctl projects stacks")
	cmd.Flags().StringVar(&dbName, "db-name", "", "Nombre de la base de datos")
	cmd.Flags().StringVar(&dbUser, "db-user", "", "Usuario de la base de datos")
	cmd.Flags().StringVar(&dbPassword, "db-password", "secret", "Contraseña de la base de datos")
	cmd.Flags().StringVarP(&appPort, "port", "p", "8080", "Puerto de la aplicación")
	cmd.Flags().StringVar(&domain, "domain", "", "Dominio para la aplicación")

	return cmd
}

func projectsUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up <project-id>",
		Short: "Levantar un proyecto (docker compose up)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, code, err := apiClient(http.MethodPost, "/api/v1/projects/"+args[0]+"/up", nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Proyecto %s levantado.\n", args[0])
			printJSON(data)
			return nil
		},
	}
}

func projectsDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down <project-id>",
		Short: "Detener un proyecto (docker compose down)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, code, err := apiClient(http.MethodPost, "/api/v1/projects/"+args[0]+"/down", nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Proyecto %s detenido.\n", args[0])
			return nil
		},
	}
}

func projectsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <project-id>",
		Aliases: []string{"rm"},
		Short:   "Eliminar un proyecto (down + delete)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, code, err := apiClient(http.MethodDelete, "/api/v1/projects/"+args[0], nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			fmt.Printf("Proyecto %s eliminado.\n", args[0])
			return nil
		},
	}
}

func stacksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stacks",
		Short: "Listar stacks disponibles para generar proyectos",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, code, err := apiClient(http.MethodGet, "/api/v1/stacks", nil)
			fatalIfError(err)
			fatalIfNotOK(code, data)
			printJSON(data)
			return nil
		},
	}
}

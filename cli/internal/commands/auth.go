package commands

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Autenticación con Cloud Control",
	}

	cmd.AddCommand(authLoginCmd(), authRegisterCmd())
	return cmd
}

func authLoginCmd() *cobra.Command {
	var email, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Iniciar sesión y obtener token JWT",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || password == "" {
				return fmt.Errorf("--email y --password son requeridos")
			}
			body := map[string]string{"email": email, "password": password}
			data, code, err := apiClient(http.MethodPost, "/api/v1/auth/login", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)

			fmt.Println("Login exitoso.")
			fmt.Printf("Configura tu token:\n  export CC_TOKEN='<token del JSON>'\n  o usa: --token <token>\n")
			printJSON(data)
			return nil
		},
	}

	cmd.Flags().StringVarP(&email, "email", "e", "", "Email del usuario")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Contraseña")
	return cmd
}

func authRegisterCmd() *cobra.Command {
	var email, password string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Registrar un nuevo usuario",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || password == "" {
				return fmt.Errorf("--email y --password son requeridos")
			}
			body := map[string]string{"email": email, "password": password}
			data, code, err := apiClient(http.MethodPost, "/api/v1/auth/register", body)
			fatalIfError(err)
			fatalIfNotOK(code, data)

			_ = viper.WriteConfig()
			fmt.Println("Usuario registrado.")
			printJSON(data)
			return nil
		},
	}

	cmd.Flags().StringVarP(&email, "email", "e", "", "Email del usuario")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Contraseña (mínimo 8 caracteres)")
	return cmd
}

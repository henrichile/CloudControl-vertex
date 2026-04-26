package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	apiURL string
	token  string
)

var RootCmd = &cobra.Command{
	Use:   "cloudctl",
	Short: "Cloud Control CLI — gestión inteligente de contenedores",
	Long: `cloudctl es la interfaz de línea de comandos de Cloud Control.
Permite gestionar contenedores, proyectos y analizar recursos con IA.

Configura el endpoint del API con --api o la variable CC_API_URL.`,
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&apiURL, "api", "", "URL del API de Cloud Control (default: http://localhost:8080)")
	RootCmd.PersistentFlags().StringVar(&token, "token", "", "JWT token de autenticación (o CC_TOKEN)")

	viper.BindPFlag("api_url", RootCmd.PersistentFlags().Lookup("api"))
	viper.BindPFlag("token", RootCmd.PersistentFlags().Lookup("token"))

	RootCmd.AddCommand(
		newContainersCmd(),
		newProjectsCmd(),
		newAIOpsCmd(),
		newAuthCmd(),
	)
}

func initConfig() {
	viper.SetEnvPrefix("CC")
	viper.AutomaticEnv()
	viper.SetDefault("api_url", "http://localhost:8080")

	viper.SetConfigName(".cloudctl")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()
}

// apiClient performs authenticated requests to the Cloud Control API.
func apiClient(method, path string, body interface{}) ([]byte, int, error) {
	base := viper.GetString("api_url")
	url := base + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	t := viper.GetString("token")
	if t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("API unreachable at %s — is the server running?", base)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

func printJSON(data []byte) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(buf.String())
}

func fatalIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func fatalIfNotOK(code int, data []byte) {
	if code >= 400 {
		fmt.Fprintf(os.Stderr, "API error (HTTP %d):\n%s\n", code, string(data))
		os.Exit(1)
	}
}

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gestionar configuración del servidor",
	Long:  `Comandos para gestionar la configuración del servidor Nebula.`,
}

var setGitHubTokenCmd = &cobra.Command{
	Use:   "set-github-token <token>",
	Short: "Configurar token de GitHub para repositorios privados",
	Long: `Configura un token de acceso personal de GitHub para permitir
clonar repositorios privados durante el despliegue.

El token necesita el permiso "repo" para acceder a repositorios privados.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := args[0]

		body, _ := json.Marshal(map[string]string{"token": token})
		req, err := http.NewRequest("PUT", GetServerURL()+"/api/v1/settings/github-token", bytes.NewBuffer(body))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+GetToken())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error al conectar con el servidor: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Error: el servidor respondió con código %d\n", resp.StatusCode)
			os.Exit(1)
		}

		fmt.Println("Token de GitHub configurado correctamente")
	},
}

var getGitHubTokenCmd = &cobra.Command{
	Use:   "get-github-token",
	Short: "Ver estado del token de GitHub",
	Long:  `Muestra si hay un token de GitHub configurado en el servidor.`,
	Run: func(cmd *cobra.Command, args []string) {
		req, err := http.NewRequest("GET", GetServerURL()+"/api/v1/settings/github-token", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		req.Header.Set("Authorization", "Bearer "+GetToken())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error al conectar con el servidor: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Error: el servidor respondió con código %d\n", resp.StatusCode)
			os.Exit(1)
		}

		var result struct {
			Configured bool `json:"configured"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "Error al leer respuesta: %v\n", err)
			os.Exit(1)
		}

		if result.Configured {
			fmt.Println("Token de GitHub: Configurado")
		} else {
			fmt.Println("Token de GitHub: No configurado")
		}
	},
}

var deleteGitHubTokenCmd = &cobra.Command{
	Use:   "unset-github-token",
	Short: "Eliminar token de GitHub",
	Long:  `Elimina el token de GitHub configurado en el servidor.`,
	Run: func(cmd *cobra.Command, args []string) {
		req, err := http.NewRequest("DELETE", GetServerURL()+"/api/v1/settings/github-token", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		req.Header.Set("Authorization", "Bearer "+GetToken())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error al conectar con el servidor: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Error: el servidor respondió con código %d\n", resp.StatusCode)
			os.Exit(1)
		}

		fmt.Println("Token de GitHub eliminado correctamente")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(setGitHubTokenCmd)
	configCmd.AddCommand(getGitHubTokenCmd)
	configCmd.AddCommand(deleteGitHubTokenCmd)
}

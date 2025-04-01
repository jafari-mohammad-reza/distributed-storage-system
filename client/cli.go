package client

import (
	"fmt"
	"log/slog"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/spf13/cobra"
)

var cfg *pkg.ClientConfig

func init() {
	config, err := pkg.GetClientConfig()
	if err != nil {
		slog.Error("Error init client config", "err", err.Error())
	}
	cfg = config
}

var rootCmd = &cobra.Command{
	Use:   "dss",
	Short: "distributed storage system,",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available Commands:")
		for _, c := range cmd.Commands() {
			fmt.Printf("  %-10s %s\n", c.Name(), c.Short)
		}
	},
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload file to storage",
	Run: func(cmd *cobra.Command, args []string) {
		if err := AuthGuard(); err != nil {
			fmt.Println("error authenticating:", err.Error())
			return
		}
		filePath := cmd.Flag("path").Value.String()
		if filePath == "" {
			fmt.Println("invalid path")
			return
		}
		if err := UploadFile(filePath); err != nil {
			fmt.Println("error uploading file", err.Error())
		}
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "authenticate to your account",
	Run: func(cmd *cobra.Command, args []string) {
		var email, password string
		fmt.Print("Enter Email: ")
		fmt.Scanln(&email)

		fmt.Print("Enter Password: ")
		fmt.Scanln(&password)

		if email == "" || password == "" {
			fmt.Println("email and password are required")
			return
		}
		if err := Auth(email, password); err != nil {
			fmt.Println("error authenticating", err.Error())
		}
	},
}

var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "revoke your token",
	Run: func(cmd *cobra.Command, args []string) {
		if err := RevokeToken(); err != nil {
			fmt.Println("revoke token error", err.Error())
		}
	},
}

// commands that will exist:
// download filePath or fileHash for specific verion
// upload filePath for uploading the file
// list for list user files in storages
// delete filePath or hash for deleting the file
// sync to sync storage files with system

func InitCli() error {
	uploadCmd.PersistentFlags().StringP("path", "p", "", "file to upload")
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(revokeCmd)
	return rootCmd.Execute()
}

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

func printUploads(uploads []pkg.ListUploadsResult) {
	for _, upload := range uploads {
		fmt.Printf("ID: %s\n", upload.ID)
		fmt.Printf("File: %s\n", upload.FileName)
		fmt.Printf("Directory: %s\n\n", upload.Directory)
		fmt.Println("Versions:")
		for _, version := range upload.Versions {
			fmt.Printf("  - ID: %s\n", version.ID)
			fmt.Printf("    Created At: %s\n", version.CreatedAt)
		}
		fmt.Printf("\nCreated At: %s\n", upload.CreatedAt)
	}
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list your uploaded files",
	Run: func(cmd *cobra.Command, args []string) {
		if err := AuthGuard(); err != nil {
			fmt.Println("error authenticating:", err.Error())
			return
		}
		result, err := ListUploads()
		if err != nil {
			fmt.Sprintf("error fetching list of uploads %s", err.Error())
			return
		}
		printUploads(result)
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download file you want with version you want",
	Run: func(cmd *cobra.Command, args []string) {
		if err := AuthGuard(); err != nil {
			fmt.Println("error authenticating:", err.Error())
			return
		}
 		id := cmd.Flag("id").Value.String()
		version := cmd.Flag("version").Value.String()
		output := cmd.Flag("output").Value.String()
		if id == ""{
			fmt.Println("id can not be empty")
			return 
		}
		err := DownloadFile(id , version , output)
		if err != nil {
			fmt.Println("error downloading file" , err.Error())
			return 
		}
		fmt.Println("file downloaded successfully")
	},
}

// commands that will exist:
// download filePath or fileHash for specific version
// upload filePath for uploading the file
// list for list user files in storages
// delete filePath or hash for deleting the file
// sync to sync storage files with system

func InitCli() error {
	uploadCmd.PersistentFlags().StringP("path", "p", "", "file to upload")
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(revokeCmd)
	rootCmd.AddCommand(listCmd)
	downloadCmd.PersistentFlags().StringP("id", "", "", "fileId to download")
	downloadCmd.PersistentFlags().StringP("version", "v", "", "version to download")
	downloadCmd.PersistentFlags().StringP("output", "o", "", "where to store downloaded file")
	rootCmd.AddCommand(downloadCmd)
	return rootCmd.Execute()
}

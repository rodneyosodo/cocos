package cli

import (
	"log"

	"github.com/spf13/cobra"
	agentsdk "github.com/ultravioletrs/agent/pkg/sdk"
)

func NewDatasetsCmd(sdk agentsdk.SDK) *cobra.Command {
	return &cobra.Command{
		Use:   "upload-dataset",
		Short: "Upload a dataset CSV file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			datasetFile := args[0]

			log.Println("Uploading dataset CSV:", datasetFile)

			_, err := sdk.UploadDataset(datasetFile)
			if err != nil {
				log.Println("Error uploading dataset:", err)
				return
			}

			log.Println("Dataset uploaded successfully!")
		},
	}
}

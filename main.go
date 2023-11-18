package main

import (
	"certmaster/models"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/alexflint/go-arg"
)

type CreateCmd struct {
	Config string `help:"File name for certificate creation config"`
	JSON   string `help:"JSON string for certificate creation config"`
}
type LambdaCmd struct {
}

var args struct {
	Create *CreateCmd `arg:"subcommand:create" help:"Create a new certificate"`
	Lambda *LambdaCmd `arg:"subcommand:lambda" help:"Run AWS Lambda function to create cert"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	arg.MustParse(&args)

	switch {
	case args.Create != nil:
		jsonInput := args.Create.JSON

		if args.Create.Config != "" {
			data, err := os.ReadFile(args.Create.Config)
			if err != nil {
				panic(err)
			}
			jsonInput = string(data)
		}

		var request models.CertRequest
		err := json.Unmarshal([]byte(jsonInput), &request)
		if err != nil {
			log.Fatal(err)
		}

		err = createCert(&request)
		if err != nil {
			log.Fatal(err)
		}
	case args.Lambda != nil:
		lambda.Start(HandleLambdaEvent)
	}

}

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type SSMParameter struct {
	ARN              string    `json:"arn"`
	DataType         string    `json:"dataType"`
	LastModifiedDate time.Time `json:"lastModifiedDate"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Value            string    `json:"value"`
	Version          int64     `json:"version"`
}

func (p *SSMParameter) Print() {
	fmt.Printf("ARN: %s\n", p.ARN)
	fmt.Printf("DataType: %s\n", p.DataType)
	fmt.Printf("LastModifiedDate: %s\n", p.LastModifiedDate)
	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Type: %s\n", p.Type)
	fmt.Printf("Value: %s\n", p.Value)
	fmt.Printf("Version: %d\n", p.Version)
}

func SSMGet(ctx context.Context, cfg aws.Config, param string) (SSMParameter, error) {
	ssmClient := ssm.NewFromConfig(cfg)

	output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(param),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return SSMParameter{}, err
	}

	var p SSMParameter

	p.ARN = *output.Parameter.ARN
	p.DataType = *output.Parameter.DataType
	p.LastModifiedDate = *output.Parameter.LastModifiedDate
	p.Name = *output.Parameter.Name
	p.Type = string(output.Parameter.Type)
	p.Value = *output.Parameter.Value
	p.Version = output.Parameter.Version

	return p, nil
}

func SSMList(ctx context.Context, cfg aws.Config, path string, recursive ...bool) ([]SSMParameter, error) {
	ssmClient := ssm.NewFromConfig(cfg)
	paginator := ssm.NewGetParametersByPathPaginator(ssmClient, &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(recursive[0]),
		WithDecryption: aws.Bool(true),
	})
	var params []SSMParameter
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range output.Parameters {
			var param SSMParameter

			param.ARN = *p.ARN
			param.DataType = *p.DataType
			param.LastModifiedDate = *p.LastModifiedDate
			param.Name = *p.Name
			param.Type = string(p.Type)
			param.Value = *p.Value
			param.Version = p.Version

			params = append(params, param)
		}
	}

	return params, nil
}

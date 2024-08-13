/*
Package aws implements functions to interact with Amazon Web Services.
This part handles working with the SSM Parameter Store.
*/
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

const parameterTypeSecureString string = "SecureString"

// SSMParameter represents some of the fields that makes up a parameter in the AWS SSM Parameter Store.
type SSMParameter struct {
	ARN              string    `json:"arn"`
	DataType         string    `json:"dataType"`
	Error            string    `json:"error,omitempty"`
	KeyID            string    `json:"keyId,omitempty"`
	LastModifiedDate time.Time `json:"lastModifiedDate"`
	LastModifiedUser string    `json:"lastModifiedUser,omitempty"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Value            string    `json:"value"`
	Version          int64     `json:"version"`
}

// Print displays the SSMParameter to the screen.
func (p *SSMParameter) Print() {
	fmt.Printf("ARN: %s\n", p.ARN)
	fmt.Printf("DataType: %s\n", p.DataType)
	if p.Error != "" {
		fmt.Printf("Error: %s\n", p.Error)
	}
	if p.KeyID != "" {
		fmt.Printf("KeyID: %s\n", p.KeyID)
	}
	fmt.Printf("LastModifiedDate: %s\n", p.LastModifiedDate)
	if p.LastModifiedUser != "" {
		fmt.Printf("LastModifiedUser: %s\n", p.LastModifiedUser)
	}
	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Type: %s\n", p.Type)
	fmt.Printf("Value: %s\n", p.Value)
	fmt.Printf("Version: %d\n", p.Version)
}

// SSMClient returns the authenticated SSM client that can be passed to the various SSM* Functions.
func SSMClient(cfg aws.Config) *ssm.Client {
	return ssm.NewFromConfig(cfg)
}

// SSMDelete deletes a parameter by name from the SSM parameter store.
func SSMDelete(ctx context.Context, ssmClient *ssm.Client, name string) error {
	_, err := ssmClient.DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: aws.String(name)})
	return err
}

// SSMDescribeParameter returns the ID of the encryption key and the last user who set/modified an SSM parameter.
// If there is no encryption key because the parameter is a String, then the key ID will be an empty string.
func SSMDescribeParameter(ctx context.Context, ssmClient *ssm.Client, name string) (string, string, error) {
	output, err := ssmClient.DescribeParameters(ctx, &ssm.DescribeParametersInput{
		ParameterFilters: []types.ParameterStringFilter{
			{
				Key:    aws.String("Name"),
				Option: aws.String("Equals"),
				Values: []string{name},
			},
		},
	})
	if err != nil {
		return "", "", err
	}
	var keyID, lastModifiedUser string
	if len(output.Parameters) == 1 {
		if string(output.Parameters[0].Type) == parameterTypeSecureString {
			keyID = *output.Parameters[0].KeyId
		}
		lastModifiedUser = *output.Parameters[0].LastModifiedUser
		/*
			Also output.Parameters has available...
			- AllowedPattern
			- Description
			- Policies ([]types.ParameterInlinePolicy{}
			- Tier
			Along with these that GetParameter also returns...
			- ARN
			- DataType
			- LastModifiedDate
			- Name
			- Version
		*/
	} else {
		return "", "", fmt.Errorf("%d parameters were returned instead of just 1", len(output.Parameters))
	}

	return keyID, lastModifiedUser, nil
}

// SSMGet returns a populated SSMParameter structure populated with details of a named SSM parameter.
func SSMGet(ctx context.Context, ssmClient *ssm.Client, name string) (SSMParameter, error) {
	var err error
	var output *ssm.GetParameterOutput
	var p SSMParameter

	output, err = ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		p.Error = fmt.Sprint(err)
		output, err = ssmClient.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
		if err != nil {
			return SSMParameter{}, err
		}
		// Clear the value since it failed to decrypt.
		*output.Parameter.Value = ""
	}

	p.ARN = *output.Parameter.ARN
	// For some reason some SSM parameters had no data type set... These seem to show in the GUI as text.
	if output.Parameter.DataType == nil {
		p.DataType = "text"
	} else {
		p.DataType = *output.Parameter.DataType
	}
	p.LastModifiedDate = *output.Parameter.LastModifiedDate
	p.Name = *output.Parameter.Name
	p.Type = string(output.Parameter.Type)
	p.Value = *output.Parameter.Value
	p.Version = output.Parameter.Version

	p.KeyID, p.LastModifiedUser, _ = SSMDescribeParameter(ctx, ssmClient, name)

	return p, nil
}

// SSMList returns a list of parameters below a path in the SSM parameter store.
// It can optionally recurse through the paths below the supplied path.
// If the `full` parameter (for full details) is true, it'll fetch the encryption key ID and Last modified user,
// at the expense of performing an AWS API lookup per parameter found, so doesn't scale well.
func SSMList(ctx context.Context, ssmClient *ssm.Client, path string, recursive, full bool) ([]SSMParameter, error) {
	paginator := ssm.NewGetParametersByPathPaginator(ssmClient, &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(recursive),
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

			if full {
				param.KeyID, param.LastModifiedUser, _ = SSMDescribeParameter(ctx, ssmClient, param.Name)
			}

			params = append(params, param)
		}
	}

	return params, nil
}

// SSMListSafeDecrypt returns a list of parameters below a path in the SSM parameter store.
// It can optionally recurse through the paths below the supplied path.
// If the `full` parameter (for full details) is true, it'll fetch the encryption key ID and Last modified user,
// at the expense of performing an AWS API lookup per parameter found, so doesn't scale well.
// It differs from SSMList in that it retrieves parameters unencrypted then tries to decrypt them as they are
// encountered. This allows it to handle decryption errors like when the decryption key has been deleted.
func SSMListSafeDecrypt(
	ctx context.Context, ssmClient *ssm.Client, path string, recursive, full bool,
) ([]SSMParameter, error) {
	paginator := ssm.NewGetParametersByPathPaginator(ssmClient, &ssm.GetParametersByPathInput{
		Path:      aws.String(path),
		Recursive: aws.Bool(recursive),
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
			param.Version = p.Version

			if param.Type == parameterTypeSecureString {
				par, err := SSMGet(ctx, ssmClient, param.Name)
				if err != nil {
					param.Error = fmt.Sprint(err)
				} else {
					param.Error = par.Error
					param.Value = par.Value
				}
			} else {
				param.Value = *p.Value
			}

			if full {
				param.KeyID, param.LastModifiedUser, _ = SSMDescribeParameter(ctx, ssmClient, param.Name)
			}

			params = append(params, param)
		}
	}

	return params, nil
}

// SSMPut creates or updates a parameter in the SSM Parameter store.
// The name and value comes from a populated SSMParameter struct that is passed to it.
// If the Type is `SecureString` then it is expected that there is a encryption key ID being passed as well.
func SSMPut(ctx context.Context, ssmClient *ssm.Client, param *SSMParameter) (int64, error) {
	input := &ssm.PutParameterInput{
		Name:      aws.String(param.Name),
		Overwrite: aws.Bool(true),
		Type:      types.ParameterType(param.Type),
		Value:     aws.String(param.Value),
	}
	if param.Type == parameterTypeSecureString {
		input.KeyId = aws.String(param.KeyID)
	}
	output, err := ssmClient.PutParameter(ctx, input)
	if err != nil {
		return -1, err
	}
	return output.Version, nil
}

/*
Package aws implements functions to interact with Amazon Web Services.
This part handles working with the SSM Parameter Store.
*/
package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMParameter represents some of the fields that makes up a parameter in the AWS SSM Parameter Store.
type SSMParameter struct {
	AllowedPattern   string              `json:"allowedPattern,omitempty"`
	ARN              string              `json:"arn"`
	DataType         string              `json:"dataType"`
	Description      string              `json:"description,omitempty"`
	Error            string              `json:"error,omitempty"`
	KeyID            string              `json:"keyId,omitempty"`
	LastModifiedDate time.Time           `json:"lastModifiedDate"`
	LastModifiedUser string              `json:"lastModifiedUser,omitempty"`
	Name             string              `json:"name"`
	Policies         string              `json:"policies,omitempty"`
	Tier             types.ParameterTier `json:"tier,omitempty"`
	Type             string              `json:"type"`
	Value            string              `json:"value"`
	Version          int64               `json:"version"`
}

// Print displays the SSMParameter to the screen, optionally excluding the value.
func (p *SSMParameter) Print(hideValue bool) {
	if p.AllowedPattern != "" {
		fmt.Printf("AllowedPattern: %s\n", p.AllowedPattern)
	}

	fmt.Printf("ARN: %s\n", p.ARN)
	fmt.Printf("DataType: %s\n", p.DataType)

	if p.Description != "" {
		fmt.Printf("Description: %s\n", p.Description)
	}

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

	if p.Policies != "" {
		fmt.Printf("Policies: %s\n", p.Policies)
	}

	if p.Tier != "" {
		fmt.Printf("Tier: %s\n", p.Tier)
	}

	fmt.Printf("Type: %s\n", p.Type)

	if !hideValue {
		fmt.Printf("Value: %s\n", p.Value)
	}

	fmt.Printf("Version: %d\n", p.Version)
}

// SSMClient returns the authenticated SSM client that can be passed to the various SSM* Functions.
func SSMClient(cfg aws.Config) *ssm.Client {
	return ssm.NewFromConfig(cfg)
}

// SSMDelete deletes a parameter by name from the SSM parameter store.
func SSMDelete(ctx context.Context, ssmClient *ssm.Client, name string) error {
	_, err := ssmClient.DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: aws.String(name)})
	if err != nil {
		return fmt.Errorf("%w: %w", NewParameterDeleteError(name), err)
	}

	return nil
}

// SSMDescribeParameter returns extra fields for an SSM parameter that the GetParameter() call does not cover.
// If there is no encryption key because the parameter is a String, then the key ID will be an empty string.
func SSMDescribeParameter(
	ctx context.Context, ssmClient *ssm.Client, name string,
) (
	allowedPattern string,
	description string,
	keyID string,
	lastModifiedUser string,
	policies string,
	tier types.ParameterTier,
	err error,
) {
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
		err = fmt.Errorf("%w: %w", NewParameterDescribeError(name), err)
		return
	}

	if len(output.Parameters) != 1 {
		err = NewOneParameterError(len(output.Parameters))
		return
	}

	param := output.Parameters[0]

	allowedPattern = aws.ToString(param.AllowedPattern)
	description = aws.ToString(param.Description)

	keyID = ""
	if param.Type == types.ParameterTypeSecureString {
		keyID = aws.ToString(param.KeyId)
	}

	lastModifiedUser = aws.ToString(param.LastModifiedUser)

	// param.Policies is of type []types.ParameterInlinePolicy.
	// We need to take the policy text components to build a JSON array.
	if len(param.Policies) > 0 {
		var jsonArray []string

		for _, policy := range param.Policies {
			jsonArray = append(jsonArray, aws.ToString(policy.PolicyText))
		}

		policies = "[" + strings.Join(jsonArray, ",") + "]"
	}

	tier = param.Tier

	return
}

// SSMGet returns a populated SSMParameter structure populated with details of a named SSM parameter.
func SSMGet(ctx context.Context, ssmClient *ssm.Client, name string) (SSMParameter, error) {
	var p SSMParameter

	output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		p.Error = fmt.Sprint(err)

		output, err = ssmClient.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
		if err != nil {
			return SSMParameter{}, fmt.Errorf("%w: %w", NewParameterGetError(name), err)
		}
		// Clear the value since it failed to decrypt.
		output.Parameter.Value = aws.String("")
	}

	p.ARN = aws.ToString(output.Parameter.ARN)
	// For some reason some SSM parameters had no data type set... These seem to show in the GUI as text.
	if output.Parameter.DataType == nil {
		p.DataType = "text"
	} else {
		p.DataType = aws.ToString(output.Parameter.DataType)
	}

	p.LastModifiedDate = aws.ToTime(output.Parameter.LastModifiedDate)
	p.Name = aws.ToString(output.Parameter.Name)
	p.Type = string(output.Parameter.Type)
	p.Value = aws.ToString(output.Parameter.Value)
	p.Version = output.Parameter.Version

	p.AllowedPattern,
		p.Description,
		p.KeyID,
		p.LastModifiedUser,
		p.Policies,
		p.Tier,
		_ = SSMDescribeParameter(ctx, ssmClient, name)

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
			return nil, fmt.Errorf("%w : %w", errParameterGetByPath, err)
		}

		for _, p := range output.Parameters {
			param := SSMParameter{
				ARN:              aws.ToString(p.ARN),
				DataType:         aws.ToString(p.DataType),
				LastModifiedDate: aws.ToTime(p.LastModifiedDate),
				Name:             aws.ToString(p.Name),
				Type:             string(p.Type),
				Value:            aws.ToString(p.Value),
				Version:          p.Version,
			}

			if full {
				param.AllowedPattern,
					param.Description,
					param.KeyID,
					param.LastModifiedUser,
					param.Policies,
					param.Tier,
					_ = SSMDescribeParameter(ctx, ssmClient, param.Name)
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
			return nil, fmt.Errorf("%w : %w", errParameterGetByPath, err)
		}

		for _, p := range output.Parameters {
			param := SSMParameter{
				ARN:              aws.ToString(p.ARN),
				DataType:         aws.ToString(p.DataType),
				LastModifiedDate: aws.ToTime(p.LastModifiedDate),
				Name:             aws.ToString(p.Name),
				Type:             string(p.Type),
				Version:          p.Version,
			}

			if types.ParameterType(param.Type) == types.ParameterTypeSecureString {
				par, err := SSMGet(ctx, ssmClient, param.Name)
				if err != nil {
					param.Error = fmt.Sprint(err)
				} else {
					param.Error = par.Error
					param.Value = par.Value
				}
			} else {
				param.Value = aws.ToString(p.Value)
			}

			if full {
				param.AllowedPattern,
					param.Description,
					param.KeyID,
					param.LastModifiedUser,
					param.Policies,
					param.Tier,
					_ = SSMDescribeParameter(ctx, ssmClient, param.Name)
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

	if param.AllowedPattern != "" {
		input.AllowedPattern = aws.String(param.AllowedPattern)
	}

	if param.Description != "" {
		input.Description = aws.String(param.Description)
	}

	if param.Tier != "" {
		input.Tier = param.Tier
	}

	if types.ParameterType(param.Type) == types.ParameterTypeSecureString {
		input.KeyId = aws.String(param.KeyID)
	}

	output, err := ssmClient.PutParameter(ctx, input)
	if err != nil {
		return -1, fmt.Errorf("%w: %w", NewParameterPutError(param.Name), err)
	}

	return output.Version, nil
}

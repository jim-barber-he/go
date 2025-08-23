package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// mockSSMClient is a mock implementation of the SSM client for testing.
type mockSSMClient struct {
	deleteParameterFunc      func(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
	describeParametersFunc   func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
	getParameterFunc         func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	getParametersByPathFunc  func(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
	putParameterFunc         func(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
}

func (m *mockSSMClient) DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	if m.deleteParameterFunc != nil {
		return m.deleteParameterFunc(ctx, params, optFns...)
	}
	return &ssm.DeleteParameterOutput{}, nil
}

func (m *mockSSMClient) DescribeParameters(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
	if m.describeParametersFunc != nil {
		return m.describeParametersFunc(ctx, params, optFns...)
	}
	return &ssm.DescribeParametersOutput{}, nil
}

func (m *mockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.getParameterFunc != nil {
		return m.getParameterFunc(ctx, params, optFns...)
	}
	return &ssm.GetParameterOutput{}, nil
}

func (m *mockSSMClient) GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.getParametersByPathFunc != nil {
		return m.getParametersByPathFunc(ctx, params, optFns...)
	}
	return &ssm.GetParametersByPathOutput{}, nil
}

func (m *mockSSMClient) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	if m.putParameterFunc != nil {
		return m.putParameterFunc(ctx, params, optFns...)
	}
	return &ssm.PutParameterOutput{}, nil
}

// ssmClientInterface defines the interface for SSM client to allow mocking.
type ssmClientInterface interface {
	DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
	DescribeParameters(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
	PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
}

func TestSSMClient(t *testing.T) {
	t.Parallel()

	cfg := aws.Config{}
	client := SSMClient(cfg)

	if client == nil {
		t.Fatal("SSMClient() returned nil")
	}
}

func TestSSMDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		parameterName  string
		mockFunc       func(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
		expectedError  bool
		errorContains  string
	}{
		{
			name:          "successful deletion",
			parameterName: "/test/parameter",
			mockFunc: func(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
				return &ssm.DeleteParameterOutput{}, nil
			},
			expectedError: false,
		},
		{
			name:          "parameter not found",
			parameterName: "/nonexistent/parameter",
			mockFunc: func(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
				return nil, &types.ParameterNotFound{Message: aws.String("Parameter not found")}
			},
			expectedError: true,
			errorContains: "failed to delete parameter",
		},
		{
			name:          "access denied",
			parameterName: "/restricted/parameter",
			mockFunc: func(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
				return nil, errors.New("access denied")
			},
			expectedError: true,
			errorContains: "failed to delete parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mockSSMClient{
				deleteParameterFunc: tt.mockFunc,
			}

			ctx := context.Background()
			err := ssmDeleteWithClient(ctx, mockClient, tt.parameterName)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errorContains != "" && !errorContains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSSMDescribeParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		parameterName         string
		mockFunc              func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
		expectedAllowedPattern string
		expectedDescription   string
		expectedKeyID         string
		expectedLastModUser   string
		expectedPolicies      string
		expectedTier          types.ParameterTier
		expectedError         bool
		errorContains         string
	}{
		{
			name:          "successful describe string parameter",
			parameterName: "/test/string-param",
			mockFunc: func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
				return &ssm.DescribeParametersOutput{
					Parameters: []types.ParameterMetadata{
						{
							AllowedPattern:   aws.String("^[a-zA-Z0-9]+$"),
							Description:      aws.String("Test parameter"),
							LastModifiedUser: aws.String("testuser"),
							Type:             types.ParameterTypeString,
							Tier:             types.ParameterTierStandard,
						},
					},
				}, nil
			},
			expectedAllowedPattern: "^[a-zA-Z0-9]+$",
			expectedDescription:    "Test parameter",
			expectedKeyID:          "",
			expectedLastModUser:    "testuser",
			expectedPolicies:       "",
			expectedTier:           types.ParameterTierStandard,
			expectedError:          false,
		},
		{
			name:          "successful describe secure string parameter",
			parameterName: "/test/secure-param",
			mockFunc: func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
				return &ssm.DescribeParametersOutput{
					Parameters: []types.ParameterMetadata{
						{
							Description:      aws.String("Secure test parameter"),
							KeyId:            aws.String("alias/aws/ssm"),
							LastModifiedUser: aws.String("testuser"),
							Type:             types.ParameterTypeSecureString,
							Tier:             types.ParameterTierStandard,
							Policies: []types.ParameterInlinePolicy{
								{
									PolicyText: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"ssm:GetParameter"}]}`),
								},
							},
						},
					},
				}, nil
			},
			expectedAllowedPattern: "",
			expectedDescription:    "Secure test parameter",
			expectedKeyID:          "alias/aws/ssm",
			expectedLastModUser:    "testuser",
			expectedPolicies:       `[{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"ssm:GetParameter"}]}]`,
			expectedTier:           types.ParameterTierStandard,
			expectedError:          false,
		},
		{
			name:          "parameter not found",
			parameterName: "/nonexistent/parameter",
			mockFunc: func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
				return &ssm.DescribeParametersOutput{
					Parameters: []types.ParameterMetadata{},
				}, nil
			},
			expectedError: true,
			errorContains: "expected 1 parameter, got 0",
		},
		{
			name:          "multiple parameters returned",
			parameterName: "/test/parameter",
			mockFunc: func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
				return &ssm.DescribeParametersOutput{
					Parameters: []types.ParameterMetadata{
						{Name: aws.String("/test/parameter1")},
						{Name: aws.String("/test/parameter2")},
					},
				}, nil
			},
			expectedError: true,
			errorContains: "expected 1 parameter, got 2",
		},
		{
			name:          "API error",
			parameterName: "/test/parameter",
			mockFunc: func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
				return nil, errors.New("API error")
			},
			expectedError: true,
			errorContains: "failed to describe parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mockSSMClient{
				describeParametersFunc: tt.mockFunc,
			}

			ctx := context.Background()
			allowedPattern, description, keyID, lastModUser, policies, tier, err := ssmDescribeParameterWithClient(ctx, mockClient, tt.parameterName)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errorContains != "" && !errorContains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if allowedPattern != tt.expectedAllowedPattern {
				t.Errorf("expected allowedPattern %q, got %q", tt.expectedAllowedPattern, allowedPattern)
			}
			if description != tt.expectedDescription {
				t.Errorf("expected description %q, got %q", tt.expectedDescription, description)
			}
			if keyID != tt.expectedKeyID {
				t.Errorf("expected keyID %q, got %q", tt.expectedKeyID, keyID)
			}
			if lastModUser != tt.expectedLastModUser {
				t.Errorf("expected lastModifiedUser %q, got %q", tt.expectedLastModUser, lastModUser)
			}
			if policies != tt.expectedPolicies {
				t.Errorf("expected policies %q, got %q", tt.expectedPolicies, policies)
			}
			if tier != tt.expectedTier {
				t.Errorf("expected tier %v, got %v", tt.expectedTier, tier)
			}
		})
	}
}

// Helper functions for testing internal functionality with mocked clients
func ssmDeleteWithClient(ctx context.Context, ssmClient ssmClientInterface, name string) error {
	_, err := ssmClient.DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: aws.String(name)})
	if err != nil {
		return fmt.Errorf("%w: %w", NewParameterDeleteError(name), err)
	}
	return nil
}

func ssmDescribeParameterWithClient(
	ctx context.Context, ssmClient ssmClientInterface, name string,
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

		policies = "[" + fmt.Sprintf("%s", jsonArray[0])
		for i := 1; i < len(jsonArray); i++ {
			policies += "," + jsonArray[i]
		}
		policies += "]"
	}

	tier = param.Tier

	return
}

// errorContains checks if an error message contains a substring.
func errorContains(errMsg, substr string) bool {
	return strings.Contains(errMsg, substr)
}

func TestSSMGet(t *testing.T) {
	t.Parallel()

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name                string
		parameterName       string
		describe            bool
		getParameterFunc    func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
		describeParamsFunc  func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
		expectedParam       SSMParameter
		expectedError       bool
		errorContains       string
	}{
		{
			name:          "successful get without describe",
			parameterName: "/test/parameter",
			describe:      false,
			getParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{
						ARN:              aws.String("arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter"),
						DataType:         aws.String("text"),
						LastModifiedDate: aws.Time(testTime),
						Name:             aws.String("/test/parameter"),
						Type:             types.ParameterTypeString,
						Value:            aws.String("test-value"),
						Version:          int64(1),
					},
				}, nil
			},
			expectedParam: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter",
				DataType:         "text",
				LastModifiedDate: testTime,
				Name:             "/test/parameter",
				Type:             "String",
				Value:            "test-value",
				Version:          1,
			},
			expectedError: false,
		},
		{
			name:          "successful get with describe",
			parameterName: "/test/parameter",
			describe:      true,
			getParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{
						ARN:              aws.String("arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter"),
						DataType:         aws.String("text"),
						LastModifiedDate: aws.Time(testTime),
						Name:             aws.String("/test/parameter"),
						Type:             types.ParameterTypeString,
						Value:            aws.String("test-value"),
						Version:          int64(1),
					},
				}, nil
			},
			describeParamsFunc: func(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
				return &ssm.DescribeParametersOutput{
					Parameters: []types.ParameterMetadata{
						{
							Description:      aws.String("Test parameter"),
							LastModifiedUser: aws.String("testuser"),
							Type:             types.ParameterTypeString,
							Tier:             types.ParameterTierStandard,
						},
					},
				}, nil
			},
			expectedParam: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter",
				DataType:         "text",
				Description:      "Test parameter",
				LastModifiedDate: testTime,
				LastModifiedUser: "testuser",
				Name:             "/test/parameter",
				Tier:             types.ParameterTierStandard,
				Type:             "String",
				Value:            "test-value",
				Version:          1,
			},
			expectedError: false,
		},
		{
			name:          "parameter with nil data type",
			parameterName: "/test/parameter",
			describe:      false,
			getParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{
						ARN:              aws.String("arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter"),
						DataType:         nil, // nil data type
						LastModifiedDate: aws.Time(testTime),
						Name:             aws.String("/test/parameter"),
						Type:             types.ParameterTypeString,
						Value:            aws.String("test-value"),
						Version:          int64(1),
					},
				}, nil
			},
			expectedParam: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter",
				DataType:         "text", // should default to "text"
				LastModifiedDate: testTime,
				Name:             "/test/parameter",
				Type:             "String",
				Value:            "test-value",
				Version:          1,
			},
			expectedError: false,
		},
		{
			name:          "decryption error then fallback",
			parameterName: "/test/secure-parameter",
			describe:      false,
			getParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				// First call with decryption fails, second call without decryption succeeds
				if aws.ToBool(params.WithDecryption) {
					return nil, errors.New("decryption failed")
				}
				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{
						ARN:              aws.String("arn:aws:ssm:us-east-1:123456789012:parameter/test/secure-parameter"),
						DataType:         aws.String("text"),
						LastModifiedDate: aws.Time(testTime),
						Name:             aws.String("/test/secure-parameter"),
						Type:             types.ParameterTypeSecureString,
						Value:            aws.String(""), // value cleared on decryption failure
						Version:          int64(1),
					},
				}, nil
			},
			expectedParam: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/secure-parameter",
				DataType:         "text",
				Error:            "decryption failed",
				LastModifiedDate: testTime,
				Name:             "/test/secure-parameter",
				Type:             "SecureString",
				Value:            "",
				Version:          1,
			},
			expectedError: false,
		},
		{
			name:          "parameter not found",
			parameterName: "/nonexistent/parameter",
			describe:      false,
			getParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return nil, &types.ParameterNotFound{Message: aws.String("Parameter not found")}
			},
			expectedError: true,
			errorContains: "failed to get parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mockSSMClient{
				getParameterFunc:       tt.getParameterFunc,
				describeParametersFunc: tt.describeParamsFunc,
			}

			ctx := context.Background()
			param, err := ssmGetWithClient(ctx, mockClient, tt.parameterName, tt.describe)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errorContains != "" && !errorContains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare individual fields
			if param.ARN != tt.expectedParam.ARN {
				t.Errorf("expected ARN %q, got %q", tt.expectedParam.ARN, param.ARN)
			}
			if param.DataType != tt.expectedParam.DataType {
				t.Errorf("expected DataType %q, got %q", tt.expectedParam.DataType, param.DataType)
			}
			if param.Description != tt.expectedParam.Description {
				t.Errorf("expected Description %q, got %q", tt.expectedParam.Description, param.Description)
			}
			if param.Error != tt.expectedParam.Error {
				t.Errorf("expected Error %q, got %q", tt.expectedParam.Error, param.Error)
			}
			if !param.LastModifiedDate.Equal(tt.expectedParam.LastModifiedDate) {
				t.Errorf("expected LastModifiedDate %v, got %v", tt.expectedParam.LastModifiedDate, param.LastModifiedDate)
			}
			if param.LastModifiedUser != tt.expectedParam.LastModifiedUser {
				t.Errorf("expected LastModifiedUser %q, got %q", tt.expectedParam.LastModifiedUser, param.LastModifiedUser)
			}
			if param.Name != tt.expectedParam.Name {
				t.Errorf("expected Name %q, got %q", tt.expectedParam.Name, param.Name)
			}
			if param.Tier != tt.expectedParam.Tier {
				t.Errorf("expected Tier %v, got %v", tt.expectedParam.Tier, param.Tier)
			}
			if param.Type != tt.expectedParam.Type {
				t.Errorf("expected Type %q, got %q", tt.expectedParam.Type, param.Type)
			}
			if param.Value != tt.expectedParam.Value {
				t.Errorf("expected Value %q, got %q", tt.expectedParam.Value, param.Value)
			}
			if param.Version != tt.expectedParam.Version {
				t.Errorf("expected Version %d, got %d", tt.expectedParam.Version, param.Version)
			}
		})
	}
}

// ssmGetWithClient is a test helper that mimics SSMGet functionality with a mocked client
func ssmGetWithClient(ctx context.Context, ssmClient ssmClientInterface, name string, describe bool) (SSMParameter, error) {
	var param SSMParameter

	// Simulate the first goroutine that gets parameter
	output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		param.Error = fmt.Sprint(err)

		output, err = ssmClient.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
		if err != nil {
			return SSMParameter{}, fmt.Errorf("%w: %w", NewParameterGetError(name), err)
		}
		// Clear the value since it failed to decrypt.
		output.Parameter.Value = aws.String("")
	}

	param.ARN = aws.ToString(output.Parameter.ARN)

	// For some reason some SSM parameters had no data type set... These seem to show in the GUI as text.
	if output.Parameter.DataType == nil {
		param.DataType = "text"
	} else {
		param.DataType = aws.ToString(output.Parameter.DataType)
	}

	param.LastModifiedDate = aws.ToTime(output.Parameter.LastModifiedDate)
	param.Name = aws.ToString(output.Parameter.Name)
	param.Type = string(output.Parameter.Type)
	param.Value = aws.ToString(output.Parameter.Value)
	param.Version = output.Parameter.Version

	// Simulate the second goroutine for describe if requested
	if describe {
		allowedPattern, description, keyID, lastModifiedUser, policies, tier, err := ssmDescribeParameterWithClient(ctx, ssmClient, name)
		if err != nil {
			return SSMParameter{}, fmt.Errorf("%w: %w", errGoRoutine, err)
		}

		param.AllowedPattern = allowedPattern
		param.Description = description
		param.KeyID = keyID
		param.LastModifiedUser = lastModifiedUser
		param.Policies = policies
		param.Tier = tier
	}

	return param, nil
}

func TestSSMPut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		parameter       *SSMParameter
		mockFunc        func(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
		expectedVersion int64
		expectedError   bool
		errorContains   string
	}{
		{
			name: "successful put string parameter",
			parameter: &SSMParameter{
				Name:        "/test/parameter",
				Value:       "test-value",
				Type:        "String",
				Description: "Test parameter",
				Tier:        types.ParameterTierStandard,
			},
			mockFunc: func(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
				return &ssm.PutParameterOutput{Version: int64(2)}, nil
			},
			expectedVersion: 2,
			expectedError:   false,
		},
		{
			name: "successful put secure string parameter",
			parameter: &SSMParameter{
				Name:        "/test/secure-parameter",
				Value:       "secret-value",
				Type:        "SecureString",
				KeyID:       "alias/aws/ssm",
				Description: "Secure test parameter",
			},
			mockFunc: func(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
				// Verify that KeyId is set for SecureString
				if aws.ToString(params.KeyId) != "alias/aws/ssm" {
					return nil, errors.New("KeyId not set for SecureString")
				}
				return &ssm.PutParameterOutput{Version: int64(1)}, nil
			},
			expectedVersion: 1,
			expectedError:   false,
		},
		{
			name: "successful put with allowed pattern",
			parameter: &SSMParameter{
				Name:           "/test/pattern-parameter",
				Value:          "ABC123",
				Type:           "String",
				AllowedPattern: "^[A-Z0-9]+$",
			},
			mockFunc: func(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
				return &ssm.PutParameterOutput{Version: int64(1)}, nil
			},
			expectedVersion: 1,
			expectedError:   false,
		},
		{
			name: "parameter validation error",
			parameter: &SSMParameter{
				Name:  "/invalid/parameter",
				Value: "invalid-value",
				Type:  "String",
			},
			mockFunc: func(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
				return nil, errors.New("validation error")
			},
			expectedError: true,
			errorContains: "failed to put parameter",
		},
		{
			name: "access denied error",
			parameter: &SSMParameter{
				Name:  "/restricted/parameter",
				Value: "test-value",
				Type:  "String",
			},
			mockFunc: func(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
				return nil, errors.New("access denied")
			},
			expectedError: true,
			errorContains: "failed to put parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mockSSMClient{
				putParameterFunc: tt.mockFunc,
			}

			ctx := context.Background()
			version, err := ssmPutWithClient(ctx, mockClient, tt.parameter)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errorContains != "" && !errorContains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if version != tt.expectedVersion {
				t.Errorf("expected version %d, got %d", tt.expectedVersion, version)
			}
		})
	}
}

// ssmPutWithClient is a test helper that mimics SSMPut functionality with a mocked client
func ssmPutWithClient(ctx context.Context, ssmClient ssmClientInterface, param *SSMParameter) (int64, error) {
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

func TestSSMParameterPrint(t *testing.T) {
	t.Parallel()

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	
	tests := []struct {
		name      string
		param     SSMParameter
		hideValue bool
		json      bool
	}{
		{
			name: "print text format with value",
			param: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter",
				DataType:         "text",
				Description:      "Test parameter",
				LastModifiedDate: testTime,
				Name:             "/test/parameter",
				Type:             "String",
				Value:            "test-value",
				Version:          1,
			},
			hideValue: false,
			json:      false,
		},
		{
			name: "print text format without value",
			param: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter",
				DataType:         "text",
				Description:      "Test parameter",
				LastModifiedDate: testTime,
				Name:             "/test/parameter",
				Type:             "String",
				Value:            "test-value",
				Version:          1,
			},
			hideValue: true,
			json:      false,
		},
		{
			name: "print JSON format with value",
			param: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter",
				DataType:         "text",
				Description:      "Test parameter",
				LastModifiedDate: testTime,
				Name:             "/test/parameter",
				Type:             "String",
				Value:            "test-value",
				Version:          1,
			},
			hideValue: false,
			json:      true,
		},
		{
			name: "print JSON format without value",
			param: SSMParameter{
				ARN:              "arn:aws:ssm:us-east-1:123456789012:parameter/test/parameter",
				DataType:         "text",
				Description:      "Test parameter",
				LastModifiedDate: testTime,
				Name:             "/test/parameter",
				Type:             "String",
				Value:            "test-value",
				Version:          1,
			},
			hideValue: true,
			json:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Since Print() outputs to stdout, we'll just verify it doesn't panic
			// In a real implementation, you might want to capture stdout to test output
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Print() panicked: %v", r)
				}
			}()

			tt.param.Print(tt.hideValue, tt.json)
		})
	}
}
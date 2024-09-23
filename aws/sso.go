/*
Package aws implements functions to interact with Amazon Web Services.
This part handles AWS SSO logins.
*/
package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/browser"
)

// LoginSessionDetails is for passing AWS Profile and Region options to the Login function.
type LoginSessionDetails struct {
	Profile string
	Region  string
}

type ssoCacheData struct {
	StartURL              string    `json:"startUrl"`
	Region                string    `json:"region"`
	AccessToken           string    `json:"accessToken"`
	ExpiresAt             time.Time `json:"expiresAt"`
	ClientID              string    `json:"clientId"`
	ClientSecret          string    `json:"clientSecret"`
	RegistrationExpiresAt time.Time `json:"registrationExpiresAt"`
	RefreshToken          string    `json:"refreshToken,omitempty"`
}

// withSharedConfigProfileAndRegion is a helper function to construct functional options that sets Profile and Region
// on config.LoadOptions.
func withSharedConfigProfileAndRegion(profile, region string) config.LoadOptionsFunc {
	return func(o *config.LoadOptions) error {
		o.Region = region
		o.SharedConfigProfile = profile
		return nil
	}
}

// Login gets a session to AWS, optionally specifying an AWS Profile & Region to use via the LoginSessionDetails option.
// If the session in the on-disk cache files are invalid, then perform the AWS SSO workflow to have the user login.
func Login(ctx context.Context, details *LoginSessionDetails) aws.Config {
	var cfg aws.Config
	var err error

	switch {
	case details.Profile != "" && details.Region != "":
		cfg, err = config.LoadDefaultConfig(
			ctx, withSharedConfigProfileAndRegion(details.Profile, details.Region),
		)
	case details.Profile != "":
		cfg, err = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(details.Profile))
	case details.Region != "":
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(details.Region))
	default:
		cfg, err = config.LoadDefaultConfig(ctx)
	}
	if err != nil {
		log.Panicf("failed to load AWS config: %v", err)
	}

	// Check if the AWS SSO session is valid.
	if _, err = sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err == nil {
		// Session is valid.
		return cfg
	}

	// Session is not valid, so need to perform an AWS SSO login.
	if err := ssoLogin(ctx, cfg); err != nil {
		log.Panicf("failed to perform AWS SSO login: %v", err)
	}

	/* Hmmm I don't have to fetch cfg again. It seems independent of the SSO sign-in...
	   I can just return the one I got even if AWS SSO login hasn't been performed yet after an aws sso logout.
	   I've found out that all I need is to write that updated cache file above.
	   If I don't write the cache file then later calls result in errors like:
		   failed to refresh cached credentials, failed to read cached SSO token file,
		   open /home/jim/.aws/sso/cache/94c8c21d08740f5da9eaa38d1f175c592692f0d1.json:
		   no such file or directory

	cfg, err = config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Panic(err)
	}
	*/

	return cfg
}

// ssoLogin performs the workflow required for an AWS SSO login.
// It will open a web browser for the AWS SSO with the appropriate client code.
// Once the user has performed the AWS SSO login, the details of the session are written to the same on-disk cache
// that the AWS CLI would write to. The AWS SDK uses this file automatically.
func ssoLogin(ctx context.Context, cfg aws.Config) error {
	// Recurse from assumed roles to the parent role until we find the configuration containing the SSO login details.
	sharedConfig := checkSharedConfig(ctx, getSharedConfig(&cfg))

	// Possibly these could be of use later?
	// ssoAccountId = sharedConfig.SSOAccountID
	// ssoRegion = sharedConfig.SSOSession.SSORegion

	ssoStartURL := sharedConfig.SSOSession.SSOStartURL
	ssooidcClient := ssooidc.NewFromConfig(cfg)

	clientName, err := ssoGetClientName(sharedConfig)
	if err != nil {
		return fmt.Errorf("%w: %w", errGetClientName, err)
	}

	registerClient, err := ssooidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(clientName),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errRegisterClient, err)
	}

	deviceAuth, err := ssooidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerClient.ClientId,
		ClientSecret: registerClient.ClientSecret,
		StartUrl:     aws.String(ssoStartURL),
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errStartDeviceAuth, err)
	}

	authURL := aws.ToString(deviceAuth.VerificationUriComplete)
	fmt.Fprintf(os.Stderr, "If your browser doesn't open, then open the following URL:\n%s\n\n", authURL)
	if err := browser.OpenURL(authURL); err != nil {
		return fmt.Errorf("%w: %w", errOpenBrowser, err)
	}

	// Check every 2 seconds up to 1 minute for the browser login to be completed.
	token, err := ssoTokenWait(ctx, ssooidcClient, registerClient, deviceAuth)
	if err != nil {
		return fmt.Errorf("%w: %w", errGetToken, err)
	}

	var refreshToken string
	if token.RefreshToken != nil {
		refreshToken = *token.RefreshToken
	}

	cacheData := ssoCacheData{
		StartURL:              ssoStartURL,
		Region:                sharedConfig.Region,
		AccessToken:           *token.AccessToken,
		ExpiresAt:             time.Unix(time.Now().Unix()+int64(token.ExpiresIn), 0).UTC(),
		ClientID:              *registerClient.ClientId,
		ClientSecret:          *registerClient.ClientSecret,
		RegistrationExpiresAt: time.Unix(registerClient.ClientSecretExpiresAt, 0).UTC(),
		RefreshToken:          refreshToken,
	}

	cacheFilePath, err := getCacheFilePath(sharedConfig.SSOSessionName, ssoStartURL)
	if err != nil {
		return fmt.Errorf("%w: %w", errGetCachePath, err)
	}

	if err := writeCacheFile(cacheFilePath, &cacheData); err != nil {
		return fmt.Errorf("%w: %w", errWriteCacheFile, err)
	}

	return nil
}

func ssoGetClientName(sharedConfig config.SharedConfig) (string, error) {
	if sharedConfig.RoleSessionName != "" {
		return sharedConfig.RoleSessionName, nil
	}

	osUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("%w: %w", errOSUserNotFound, err)
	}

	return fmt.Sprintf("%s-%s-%s", osUser, sharedConfig.Profile, sharedConfig.SSORoleName), nil
}

func ssoTokenWait(
	ctx context.Context,
	ssooidcClient *ssooidc.Client,
	registerClient *ssooidc.RegisterClientOutput,
	deviceAuth *ssooidc.StartDeviceAuthorizationOutput,
) (*ssooidc.CreateTokenOutput, error) {
	var createTokenErr error
	timeout := time.Minute
	sleepTime := 2 * time.Second
	startTime := time.Now()

	token := new(ssooidc.CreateTokenOutput)
	for time.Since(startTime) < timeout {
		token, createTokenErr = ssooidcClient.CreateToken(
			ctx, &ssooidc.CreateTokenInput{
				ClientId:     registerClient.ClientId,
				ClientSecret: registerClient.ClientSecret,
				DeviceCode:   deviceAuth.DeviceCode,
				GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
				// TODO: Work out how to use the following instead of DeviceCode and the above GrantType?
				// Is this the key to getting refreshToken in my SSO cached credentials?
				// GrantType:    aws.String("refresh_token"),
			},
		)
		if createTokenErr == nil {
			return token, nil
		}
		if strings.Contains(createTokenErr.Error(), "AuthorizationPendingException") {
			time.Sleep(sleepTime)
		}
	}
	if createTokenErr != nil {
		return nil, errSSOTimeout
	}
	return token, nil
}

// checkSharedConfig checks for a valid shared config from the user's AWS Profile to see if it has valid SSO session
// details. If not, and it references a source profile, then load that and call this function again (recurse) to
// check that, and so on. Eventually you'll hit a valid profile, or you'll get to the top-level where there is no
// valid SSO session details at which point it has to give up.
func checkSharedConfig(ctx context.Context, sharedConfig config.SharedConfig) config.SharedConfig {
	if sharedConfig.SSOSession != nil {
		return sharedConfig
	}

	if sharedConfig.SourceProfileName == "" {
		log.Panic("Current AWS Profile does not support AWS SSO")
	}

	// Check the source profile.
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(sharedConfig.SourceProfileName))
	if err != nil {
		log.Panicf("failed to load source profile %s: %v", sharedConfig.SourceProfileName, err)
	}

	return checkSharedConfig(ctx, getSharedConfig(&cfg))
}

// getSharedConfig extracts the shared config from the slice of interfaces contained in the aws.Config struct.
func getSharedConfig(cfg *aws.Config) config.SharedConfig {
	for _, cs := range cfg.ConfigSources {
		// Use type assertion to match the interface that is of type `config.SharedConfig`.
		if sc, ok := cs.(config.SharedConfig); ok {
			return sc
		}
	}

	return config.SharedConfig{}
}

// getCacheFilePath returns the on-disk path of the cache file containing the AWS SSO session credentials.
func getCacheFilePath(ssoSessionName, ssoStartURL string) (string, error) {
	var cacheFilePath string
	var err error

	// Determine the cache file path based on the provided SSO session name or start URL.
	if ssoSessionName != "" {
		cacheFilePath, err = ssocreds.StandardCachedTokenFilepath(ssoSessionName)
	} else {
		cacheFilePath, err = ssocreds.StandardCachedTokenFilepath(ssoStartURL)
	}

	if err != nil {
		return "", fmt.Errorf("%w: %w", errGetCachePath, err)
	}
	return cacheFilePath, nil
}

// writeCacheFile writes the contents of the valid credentials received after an AWS SSO login to a file.
// It is expected that the correct cache file path is passed in as retrieved via the getCacheFilePath() function.
func writeCacheFile(cacheFilePath string, cacheFileData *ssoCacheData) error {
	marshaledJSON, err := json.Marshal(cacheFileData)
	if err != nil {
		return fmt.Errorf("%w: %w", errMarshalJSON, err)
	}

	dir, _ := path.Split(cacheFilePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("%w: %w", NewCreateDirError(dir), err)
	}

	if err := os.WriteFile(cacheFilePath, marshaledJSON, 0o600); err != nil {
		return fmt.Errorf("%w: %w", NewWriteCacheFileError(cacheFilePath), err)
	}

	return nil
}

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

// Login gets a session to AWS, optionally specifying an AWS Profile to use.
// If the session in the on-disk cache files are invalid, then perform the AWS SSO workflow to have the user login.
func Login(ctx context.Context, awsProfile ...string) aws.Config {
	// cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("ap-southeast-2"))
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(awsProfile[0]))
	if err != nil {
		log.Panic(err)
	}

	// Check if the AWS SSO session is valid.
	if _, err = sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err == nil {
		// Session is valid.
		return cfg
	}

	// Session is not valid, so need to perform an AWS SSO login.
	ssoLogin(ctx, cfg)

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
func ssoLogin(ctx context.Context, cfg aws.Config) {
	// Recurse from assumed roles to the parent role until we find the configuration containing the SSO login details.
	sharedConfig := checkSharedConfig(ctx, getSharedConfig(&cfg))

	// Possibly these could be of use later?
	// ssoAccountId = sharedConfig.SSOAccountID
	// ssoRegion = sharedConfig.SSOSession.SSORegion

	ssoStartURL := sharedConfig.SSOSession.SSOStartURL

	ssooidcClient := ssooidc.NewFromConfig(cfg)

	var clientName string
	if sharedConfig.RoleSessionName != "" {
		clientName = sharedConfig.RoleSessionName
	} else {
		osUser, err := user.Current()
		if err != nil {
			log.Panic(err)
		}
		clientName = fmt.Sprintf("%s-%s-%s", osUser, sharedConfig.Profile, sharedConfig.SSORoleName)
	}

	registerClient, err := ssooidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(clientName),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		log.Panic(err)
	}

	deviceAuth, err := ssooidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerClient.ClientId,
		ClientSecret: registerClient.ClientSecret,
		StartUrl:     aws.String(ssoStartURL),
	})
	if err != nil {
		log.Panic(err)
	}

	authURL := aws.ToString(deviceAuth.VerificationUriComplete)
	fmt.Fprintf(os.Stderr, "If your browser doesn't open, then open the following URL:\n%s\n\n", authURL)
	err = browser.OpenURL(authURL)
	if err != nil {
		log.Panic(err)
	}

	// Check every 2 seconds up to 1 minute for the browser login to be completed.
	var createTokenErr error
	timeout := time.Minute
	sleepTime := 2 * time.Second
	startTime := time.Now()
	timeDelta := time.Since(startTime)
	token := new(ssooidc.CreateTokenOutput)
	for timeDelta < timeout {
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
			break
		}
		if strings.Contains(createTokenErr.Error(), "AuthorizationPendingException") {
			time.Sleep(sleepTime)
			timeDelta = time.Since(startTime)
		}
	}
	if createTokenErr != nil {
		log.Panic("SSO login attempt timed out")
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
		log.Panic(err)
	}

	err = writeCacheFile(cacheFilePath, &cacheData)
	if err != nil {
		log.Panic(err)
	}
}

// checkSharedConfig checks for a valid shared config from the user's AWS Profile to see if it has valid SSO session
// details. If not, and it references a source profile, then load that and call this function again (recurse) to
// check that, and so on. Eventually you'll hit a valid profile, or you'll get to the top-level where there is no
// valid SSO session details at which point it has to give up.
func checkSharedConfig(ctx context.Context, sc config.SharedConfig) config.SharedConfig {
	sharedConfig := sc
	if sharedConfig.SSOSession == nil {
		if sharedConfig.SourceProfileName == "" {
			log.Panic("Current AWS Profile does not support AWS SSO")
		}

		// Check the source profile.
		cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(sharedConfig.SourceProfileName))
		if err != nil {
			log.Panic(err)
		}
		sharedConfig = checkSharedConfig(ctx, getSharedConfig(&cfg))
	}
	return sharedConfig
}

// getSharedConfig extracts the shared config from the slice of interfaces contained in the aws.Config struct.
func getSharedConfig(cfg *aws.Config) config.SharedConfig {
	var sharedConfig config.SharedConfig
	for _, cs := range cfg.ConfigSources {
		// Use type assertion to match the interface that is of type `config.SharedConfig`.
		if sc, ok := cs.(config.SharedConfig); ok {
			sharedConfig = sc
		}
	}

	return sharedConfig
}

// getCacheFilePath returns the on-disk path of the cache file containing the AWS SSO session credentials.
func getCacheFilePath(ssoSessionName, ssoStartURL string) (string, error) {
	var cacheFilePath string
	var err error

	if ssoSessionName != "" {
		cacheFilePath, err = ssocreds.StandardCachedTokenFilepath(ssoSessionName)
	} else {
		cacheFilePath, err = ssocreds.StandardCachedTokenFilepath(ssoStartURL)
	}

	if err != nil {
		return "", err
	}
	return cacheFilePath, nil
}

// writeCacheFile writes the contents of the valid credentials received after an AWS SSO login to a file.
// It is expected that the correct cache file path is passed in as retrieved via the getCacheFilePath() function.
func writeCacheFile(cacheFilePath string, cacheFileData *ssoCacheData) error {
	marshaledJSON, err := json.Marshal(cacheFileData)
	if err != nil {
		return err
	}
	dir, _ := path.Split(cacheFilePath)
	err = os.MkdirAll(dir, 0o700)
	if err != nil {
		return err
	}

	err = os.WriteFile(cacheFilePath, marshaledJSON, 0o600)
	if err != nil {
		return err
	}
	return nil
}

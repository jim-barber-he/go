/*
Package aws implements functions to interact with Amazon Web Services.
This part handles AWS SSO logins.
*/
package aws

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
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

// loadConfig is a helper function to load the AWS configuration with the provided details.
func loadConfig(ctx context.Context, details *LoginSessionDetails) aws.Config {
	var (
		cfg aws.Config
		err error
	)

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

	return cfg
}

// Login gets a session to AWS, optionally specifying an AWS Profile & Region to use via the LoginSessionDetails option.
// If the session in the on-disk cache files are invalid, then perform the AWS SSO workflow to have the user login.
func Login(ctx context.Context, details *LoginSessionDetails, clientName string) aws.Config {
	// Load the AWS configuration based on the provided details.
	cfg := loadConfig(ctx, details)

	// Check if the AWS SSO session is valid.
	if _, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err == nil {
		// Session is valid.
		return cfg
	}

	// Recurse from assumed roles to the parent role until we find the configuration containing the SSO login details.
	sharedConfig := checkSharedConfig(ctx, getSharedConfig(&cfg))

	// Try to refresh token before full login.
	if cachePath, err := getCacheFilePath(sharedConfig.SSOSessionName, sharedConfig.SSOSession.SSOStartURL); err == nil {
		if cache, err := readCacheFile(cachePath); err == nil {
			if time.Now().UTC().After(cache.ExpiresAt) {
				// Expired, try refresh
				if cache.RefreshToken != "" {
					refreshed, err := refreshSSOToken(ctx, cache, cfg)
					if err == nil {
						_ = writeCacheFile(cachePath, refreshed)

						return cfg
					}

					var errInvalidGrant *types.InvalidGrantException
					if errors.As(err, &errInvalidGrant) {
						log.Println("Token expired, falling back to login")
					} else {
						log.Printf("Token refresh failed, falling back to login: %v", err)
					}
				}
			} else {
				// Still valid — skip login
				return cfg
			}
		}
	}

	// Session is not valid, so need to perform an AWS SSO login.
	if err := ssoLogin(ctx, cfg, sharedConfig, clientName); err != nil {
		log.Panicf("failed to perform AWS SSO login: %v", err)
	}

	/* I don't have to fetch cfg again. It seems independent of the SSO sign-in...
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

// refreshSSOToken attempts to refresh the AWS SSO token using the refresh token stored in the cache.
func refreshSSOToken(ctx context.Context, cache *ssoCacheData, cfg aws.Config) (*ssoCacheData, error) {
	ssooidcClient := ssooidc.NewFromConfig(cfg)

	token, err := ssooidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     aws.String(cache.ClientID),
		ClientSecret: aws.String(cache.ClientSecret),
		GrantType:    aws.String("refresh_token"),
		RefreshToken: aws.String(cache.RefreshToken),
	})
	if err != nil {
		return nil, fmt.Errorf("refresh token failed: %w", err)
	}

	cache.AccessToken = *token.AccessToken
	cache.ExpiresAt = time.Now().UTC().Add(time.Duration(token.ExpiresIn) * time.Second)

	if token.RefreshToken != nil {
		cache.RefreshToken = *token.RefreshToken
	}

	return cache, nil
}

// ssoLogin performs the workflow required for an AWS SSO login.
// It will open a web browser for the AWS SSO login flow.
// It tries the PKCE (Proof Key for Code Exchange) flow, and on failure, falls back to the Device Authorization flow.
// Once the user has performed the AWS SSO login, the details of the session are written to the same on-disk cache
// that the AWS CLI would write to. The AWS SDK uses this file automatically.
func ssoLogin(ctx context.Context, cfg aws.Config, sharedConfig config.SharedConfig, clientName string) error {
	// Possibly these could be of use later?
	// ssoAccountId = sharedConfig.SSOAccountID
	// ssoRegion = sharedConfig.SSOSession.SSORegion

	// Try PKCE flow, and fall back to the device authorization flow if it fails.
	cacheData, err := ssoLoginWithPKCE(ctx, cfg, sharedConfig, clientName)
	if err != nil {
		log.Printf("PKCE login failed, falling back to device authorization flow: %v", err)

		cacheData, err = ssoLoginWithDeviceAuthorization(ctx, cfg, sharedConfig, clientName)
		if err != nil {
			return fmt.Errorf("failed to perform AWS SSO login: %w", err)
		}
	}

	cacheFilePath, err := getCacheFilePath(sharedConfig.SSOSessionName, cacheData.StartURL)
	if err != nil {
		return fmt.Errorf("%w: %w", errGetCachePath, err)
	}

	if err := writeCacheFile(cacheFilePath, cacheData); err != nil {
		return fmt.Errorf("%w: %w", errWriteCacheFile, err)
	}

	return nil
}

// ssoLoginWithPKCE attempts to perform an AWS SSO login using the PKCE (Proof Key for Code Exchange) flow.
func ssoLoginWithPKCE(
	ctx context.Context, cfg aws.Config, sharedConfig config.SharedConfig, clientName string,
) (*ssoCacheData, error) {
	ssoStartURL := sharedConfig.SSOSession.SSOStartURL
	ssooidcClient := ssooidc.NewFromConfig(cfg)

	expectedState := uuid.NewString()

	redirectURI, codeChan := localCallbackServer(ctx, expectedState)

	registerClient, err := ssooidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName:   aws.String(clientName),
		ClientType:   aws.String("public"),
		GrantTypes:   []string{"authorization_code", "refresh_token"},
		IssuerUrl:    aws.String(ssoStartURL),
		RedirectUris: []string{redirectURI},
		Scopes:       []string{"sso:account:access"},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRegisterClient, err)
	}

	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	authURL := fmt.Sprintf(
		"https://oidc.ap-southeast-2.amazonaws.com/authorize"+
			"?response_type=code"+
			"&client_id=%s"+
			"&redirect_uri=%s"+
			"&state=%s"+
			"&code_challenge_method=S256"+
			"&scopes=%s"+
			"&code_challenge=%s",
		url.QueryEscape(*registerClient.ClientId),
		url.QueryEscape(redirectURI),
		expectedState,
		url.QueryEscape("sso:account:access"),
		codeChallenge,
	)

	fmt.Fprintf(os.Stderr, "Opening browser for login. If it doesn't open, go to:\n%s\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("%w: %w", errOpenBrowser, err)
	}

	// Read the authorization code from the local callback server.
	code := <-codeChan

	// Create the OIDC token.
	token, err := ssooidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     registerClient.ClientId,
		ClientSecret: registerClient.ClientSecret,
		GrantType:    aws.String("authorization_code"),
		Code:         aws.String(code),
		RedirectUri:  aws.String(redirectURI),
		CodeVerifier: aws.String(codeVerifier),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGetToken, err)
	}

	var refreshToken string
	if token.RefreshToken != nil {
		refreshToken = *token.RefreshToken
	}

	return &ssoCacheData{
		StartURL:              ssoStartURL,
		Region:                sharedConfig.Region,
		AccessToken:           *token.AccessToken,
		ExpiresAt:             time.Now().UTC().Add(time.Duration(token.ExpiresIn) * time.Second),
		ClientID:              *registerClient.ClientId,
		ClientSecret:          *registerClient.ClientSecret,
		RegistrationExpiresAt: time.Unix(registerClient.ClientSecretExpiresAt, 0).UTC(),
		RefreshToken:          refreshToken,
	}, nil
}

// generateCodeVerifier creates a high-entropy PKCE code_verifier.
func generateCodeVerifier() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"

	const strLen = 64

	char := make([]byte, strLen)
	for idx := range char {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			log.Panicf("failed to generate random index: %v", err)
		}

		char[idx] = charset[num.Int64()]
	}

	return string(char)
}

// generateCodeChallenge creates a PKCE code_challenge from the code_verifier using SHA-256 hashing.
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// localCallbackServer starts a local HTTP server to handle the callback from the AWS SSO login.
// It listens on a random port and returns the redirect URI and a channel to receive the authorization code.
func localCallbackServer(ctx context.Context, expectedState string) (string, <-chan string) {
	const httpReadTimeout = 5 * time.Second

	var lc net.ListenConfig

	codeChan := make(chan string, 1)

	// Create a TCP listener on 127.0.0.1 on a random unused port.
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to start listener: %v", err)
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		log.Fatalf("listener.Addr() was not a *net.TCPAddr")
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", addr.Port)

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux, ReadHeaderTimeout: httpReadTimeout}

	mux.HandleFunc("/oauth/callback", func(writer http.ResponseWriter, reader *http.Request) {
		state := reader.URL.Query().Get("state")
		if state != expectedState {
			http.Error(writer, "State mismatch", http.StatusBadRequest)
			log.Printf("State mismatch: expected %s, got %s", expectedState, state)

			return
		}

		code := reader.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintln(writer, "Login successful. You may now close this window.")

			codeChan <- code
		} else {
			http.Error(writer, "No code found", http.StatusBadRequest)
		}

		// Close the server and log any error.
		go func() {
			if err := server.Close(); err != nil {
				log.Printf("error closing local callback server: %v", err)
			}
		}()
	})

	go func() {
		if err := server.Serve(listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Printf("error with local callback server: %v", err)
			}
		}
	}()

	return redirectURI, codeChan
}

// ssoLoginWithDeviceAuthorization attempts to perform an AWS SSO login using the Device-Authorization flow.
func ssoLoginWithDeviceAuthorization(
	ctx context.Context, cfg aws.Config, sharedConfig config.SharedConfig, clientName string,
) (*ssoCacheData, error) {
	ssoStartURL := sharedConfig.SSOSession.SSOStartURL
	ssooidcClient := ssooidc.NewFromConfig(cfg)

	registerClient, err := ssooidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(clientName),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRegisterClient, err)
	}

	deviceAuth, err := ssooidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerClient.ClientId,
		ClientSecret: registerClient.ClientSecret,
		StartUrl:     aws.String(ssoStartURL),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errStartDeviceAuth, err)
	}

	authURL := aws.ToString(deviceAuth.VerificationUriComplete)
	fmt.Fprintf(os.Stderr, "If your browser doesn't open, then open the following URL:\n%s\n\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("%w: %w", errOpenBrowser, err)
	}

	// Wait a while for the browser login to be completed.
	token, err := ssoTokenWait(ctx, ssooidcClient, registerClient, deviceAuth)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGetToken, err)
	}

	var refreshToken string
	if token.RefreshToken != nil {
		refreshToken = *token.RefreshToken
	}

	return &ssoCacheData{
		StartURL:              ssoStartURL,
		Region:                sharedConfig.Region,
		AccessToken:           *token.AccessToken,
		ExpiresAt:             time.Unix(time.Now().Unix()+int64(token.ExpiresIn), 0).UTC(),
		ClientID:              *registerClient.ClientId,
		ClientSecret:          *registerClient.ClientSecret,
		RegistrationExpiresAt: time.Unix(registerClient.ClientSecretExpiresAt, 0).UTC(),
		RefreshToken:          refreshToken,
	}, nil
}

// ssoTokenWait polls for the token in the Device Authorization flow after the user has logged in via the web browser.
// It checks every 2 seconds up to 1 minute for the browser login to be completed.
func ssoTokenWait(
	ctx context.Context,
	ssooidcClient *ssooidc.Client,
	registerClient *ssooidc.RegisterClientOutput,
	deviceAuth *ssooidc.StartDeviceAuthorizationOutput,
) (*ssooidc.CreateTokenOutput, error) {
	const sleepTime = 2 * time.Second

	var createTokenErr error

	timeout := time.Minute
	startTime := time.Now()

	token := new(ssooidc.CreateTokenOutput)
	for time.Since(startTime) < timeout {
		token, createTokenErr = ssooidcClient.CreateToken(
			ctx, &ssooidc.CreateTokenInput{
				ClientId:     registerClient.ClientId,
				ClientSecret: registerClient.ClientSecret,
				DeviceCode:   deviceAuth.DeviceCode,
				GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
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
	var (
		cacheFilePath string
		err           error
	)

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

// readCacheFile reads the contents of the valid credentials from a file containing the results of an AWS SSO login.
// It is expected that the correct cache file path is passed in as retrieved via the getCacheFilePath() function.
func readCacheFile(cacheFilePath string) (*ssoCacheData, error) {
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, err
	}

	var cache ssoCacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
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

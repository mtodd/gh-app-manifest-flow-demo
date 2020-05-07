package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"gopkg.in/yaml.v2"
)

var templates *template.Template
var servercfg serverConfig
var appcfg appConfig

const serverCfgPath = "config/server.yml"
const manifestPath = "config/manifest.yml"
const appCfgPath = "config/app.yml"

func main() {
	var err error

	// load server config (includes app manifest, GitHub endpoints, and local server)
	servercfg, err = loadServerConfig()
	if err != nil {
		log.Fatal("could not load server config", err)
		return
	}
	log.Println(fmt.Sprintf("server cfg: %v", servercfg))

	// try to load the config for the GitHub App if it has been successfully registered already
	appcfg, err = loadAppConfig()
	if err != nil {
		log.Println(fmt.Sprintf("could not load app config: %s", err))
		log.Println("starting server in app registration mode")
	}
	log.Println(fmt.Sprintf("app cfg: %v", appcfg))

	// load templates
	if err := loadTemplates(); err != nil {
		log.Fatal("could not load templates", err)
		return
	}

	// setup routes
	http.HandleFunc("/favicon.ico", notFoundHandler)
	http.HandleFunc("/registered", registerCallbackHandler)
	http.HandleFunc("/installed", installCallbackHandler)
	http.HandleFunc("/", indexHandler)

	// start serving requests
	log.Println("listening on port 5000")
	log.Fatal(http.ListenAndServe(":5000", nil))
}

///////////////////////////////////////////////////////////////////////////////
// ENDPOINTS
///////////////////////////////////////////////////////////////////////////////

// GET /
//
// Landing page to initiate app registration and installation flows.
//
// Once the GitHub App is registered and installed, we perform a number of API requests and display
// their results inline.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var installedcfg installedIndexPage

	// if the app has been registered and installed, demonstrate a few API calls as the GitHub App
	installedcfg, err = fetchInstalledDetails(appcfg)
	if err != nil {
		log.Printf("[nonfatal] unable to fetch installation details: %s", err)
	}

	// e.g. https://github.com/settings/apps/new (to initiate app registration via our app manifest)
	registerURL := buildGitHubURL(servercfg.CreateAppPath, servercfg)
	// e.g. https://github.com/apps/my-app-slug/installations/new (once the app is registered but not yet installed)
	installURL := buildGitHubURL(fmt.Sprintf(servercfg.InstallPath, servercfg.GitHubAppsPrefix, appcfg.Slug), servercfg)

	page := indexPage{"Register your application", registerURL, installURL, installedcfg, appcfg, servercfg}

	t := templates.Lookup("index.html.tpl")
	err = t.Execute(w, page)
	if err != nil {
		log.Println(err)
	}
}

// required for the index template
type indexPage struct {
	Title        string
	CreateAppURL string
	InstallURL   string
	installedIndexPage
	appConfig
	serverConfig
}

// only used in the index template when the app is registered and installed
type installedIndexPage struct {
	AppAuthedJSON         string
	AccessTokenJSON       string
	AccessToken           accessToken
	InstallationReposJSON string
}
type accessToken struct {
	Token       string            `json:"token"`
	ExpiresAt   string            `json:"expires_at"`
	Permissions map[string]string `json:"permissions"`
}

// GET /registered?code=<code>
//
// The callback endpoint GitHub redirects back to after the GitHub App registration request is created by the user.
//
// This endpoint is responsible for performing an API call to convert the pending registration request into a fully registered GitHub App
// and subsequently retrieve the registered app's secrets/details (App ID, secret, PEM, etc).
//
// The app manifest's `redirect_url` field instructs GitHub to redirect back to this endpoint.
//
// https://developer.github.com/apps/building-github-apps/creating-github-apps-from-a-manifest/#2-github-redirects-people-back-to-your-site
func registerCallbackHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// The `?code=`` query param is set from GitHub when the request to create the GitHub App via manifest is accepted.
	// NOTE: We need to take this code and perform an API request to convert the requested app manifest into a registered GitHub App.
	query := r.URL.Query()
	code := query.Get("code")

	// Only try to process app registration callback when we have the registration code and the app isn't already configured.
	if code != "" && appcfg.AppID == "" {
		cfg, err := fetchAppRegistration(code)
		if err != nil {
			log.Println(err)
			http.Redirect(w, r, "/", 302)
			return
		}
		appcfg = cfg

		// persist the registered application details to config/app.yml
		writeAppConfig(cfg)
	}

	// e.g. https://github.com/apps/%s/installations/new
	installURL := buildGitHubURL(fmt.Sprintf(servercfg.InstallPath, servercfg.GitHubAppsPrefix, appcfg.Slug), servercfg)
	page := registeredCallbackPage{"GitHub App registered", code, installURL, appcfg, servercfg}

	t := templates.Lookup("registered.html.tpl")
	err = t.Execute(w, page)
	if err != nil {
		log.Println(err)
	}
}

type registeredCallbackPage struct {
	Title      string
	Code       string
	InstallURL string
	appConfig
	serverConfig
}

// GET /installed
//
// Users are redirected here once the registered GitHub App is installed.
//
// The app manifest's `setup_url` field instructs GitHub to redirect back to this endpoint after installation.
//
// If installation is successful, the endpoint will include an `installation_id` query parameter, which we persist
// before redirecting back to the homepage.
func installCallbackHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	query := r.URL.Query()
	installationIDStr := query.Get("installation_id")
	installationID, err := strconv.Atoi(installationIDStr)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/registered", 302)
		return
	}

	// update app config to include installation ID
	appcfg.InstallationID = installationID
	writeAppConfig(appcfg)

	http.Redirect(w, r, "/", 302)
}

// GET /favicon.ico should 404
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

///////////////////////////////////////////////////////////////////////////////
// API requests
///////////////////////////////////////////////////////////////////////////////

// API request to convert the app registration request into an actual app registration.
// https://developer.github.com/apps/building-github-apps/creating-github-apps-from-a-manifest/#3-you-exchange-the-temporary-code-to-retrieve-the-app-configuration
func fetchAppRegistration(code string) (appConfig, error) {
	var cfg appConfig

	// e.g. https://api.github.com/app-manifests/:code/conversions
	url := buildGitHubAPIURL(fmt.Sprintf(servercfg.FetchRegistrationPath, code), servercfg)

	log.Printf("POST %s", url)
	res, err := http.Post(url, "application/json", nil)
	if err != nil {
		return cfg, err
	}
	defer res.Body.Close()

	// We expect a 201 Created response status.
	if res.StatusCode != http.StatusCreated {
		return cfg, fmt.Errorf("fetch app registration failed: %v", res.Status)
	}

	// The response body is JSON.
	err = json.NewDecoder(res.Body).Decode(&cfg)
	return cfg, err
}

// Performs a series of API calls as the GitHub App once it's been installed.
func fetchInstalledDetails(appcfg appConfig) (installedIndexPage, error) {
	var cfg installedIndexPage

	// Make sure the app is registered and installed.
	if appcfg.AppID == "" {
		return cfg, fmt.Errorf("app not registered")
	}
	if appcfg.InstallationID == 0 {
		return cfg, fmt.Errorf("app not installed")
	}

	// Fetch details about the application, testing basic GitHub App authentication with a JWT.
	appauth, err := fetchApp(appcfg)
	if err != nil {
		return cfg, fmt.Errorf("unable to authenticate as the app: %s (%s)", err, appauth)
	}
	cfg.AppAuthedJSON = appauth

	// Create an access token for the GitHub App to perform API calls for the installation target (User, Organization, Repositories, etc).
	token, err := fetchAccessToken(appcfg)
	if err != nil {
		return cfg, fmt.Errorf("unable to fetch access token for installation (%d): %s (%s)", appcfg.InstallationID, err, token)
	}
	cfg.AccessTokenJSON = token
	err = json.Unmarshal([]byte(token), &cfg.AccessToken)

	// Use the access token to perform an API call, fetching the list of repositories that the GitHub App was installed on.
	// If the list is empty, that's OK: it might not've been installed on any (because it might not be an app with permissions
	// on repositories). This is an easy API call to demonstrate using an access token.
	repos, err := fetchInstallationRepos(cfg.AccessToken)
	if err != nil {
		return cfg, fmt.Errorf("unable to fetch installed repositories (%d): %s (%s)", appcfg.InstallationID, err, repos)
	}
	cfg.InstallationReposJSON = repos

	return cfg, err
}

// API request to fetch the currently authenticated app.
// https://developer.github.com/v3/apps/#get-the-authenticated-github-app
func fetchApp(cfg appConfig) (string, error) {
	var b []byte

	// e.g. https://api.github.com/app
	url := buildGitHubAPIURL("/app", servercfg)
	client := &http.Client{}

	log.Printf("GET %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// The custom media type with this `Accept` header is required to enable this endpoint.
	req.Header.Add("Accept", `application/vnd.github.machine-man-preview+json`)

	// Authentication for this endpoint requires a JWT token:
	// https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/#authenticating-as-a-github-app
	jwt, err := makejwt(cfg)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// body is JSON (but we don't parse it here; we only use it for display purposes)
	b, err = ioutil.ReadAll(res.Body)

	// Expect 200 OK response.
	if res.StatusCode != http.StatusOK {
		return string(b), fmt.Errorf("fetch app failed: %v (%v)", res.Status, string(b))
	}

	return string(b), err
}

// API request to create an installation token.
// https://developer.github.com/v3/apps/#create-a-new-installation-token
func fetchAccessToken(cfg appConfig) (string, error) {
	var b []byte

	// e.g. https://api.github.com/app/installations/:installation_id/access_tokens
	url := buildGitHubAPIURL(fmt.Sprintf("/app/installations/%d/access_tokens", cfg.InstallationID), servercfg)
	client := &http.Client{}

	log.Printf("POST %s", url)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", `application/vnd.github.machine-man-preview+json`)

	// Authentication for this endpoint requires a JWT token:
	// https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/#authenticating-as-a-github-app
	jwt, err := makejwt(cfg)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err = ioutil.ReadAll(res.Body)

	// Expect 201 Created response.
	if res.StatusCode != http.StatusCreated {
		return string(b), fmt.Errorf("fetch access token failed: %v (%v)", res.Status, string(b))
	}

	return string(b), err
}

// API request to create an installation token.
// https://developer.github.com/v3/apps/installations/#list-repositories
func fetchInstallationRepos(at accessToken) (string, error) {
	var b []byte

	// e.g. https://api.github.com/installation/repositories
	url := buildGitHubAPIURL("/installation/repositories", servercfg)
	client := &http.Client{}

	log.Printf("GET %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", `application/vnd.github.machine-man-preview+json`)

	// Authenticate with the access token created via the API call from `fetchAccessToken`.
	// Request is scoped to that installation.
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", at.Token))

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err = ioutil.ReadAll(res.Body)

	// Expect 200 OK response.
	if res.StatusCode != http.StatusOK {
		return string(b), fmt.Errorf("fetch installation repos failed: %v (%v)", res.Status, string(b))
	}

	return string(b), err
}

///////////////////////////////////////////////////////////////////////////////
// Request Utilities
///////////////////////////////////////////////////////////////////////////////

func buildGitHubURL(path string, cfg serverConfig) string {
	return fmt.Sprintf("%s://%s%s", cfg.GitHubScheme, cfg.GitHubHost, path)
}

func buildGitHubAPIURL(path string, cfg serverConfig) string {
	return fmt.Sprintf("%s://%s%s%s", cfg.GitHubScheme, cfg.GitHubAPIHost, cfg.GitHubAPIPrefix, path)
}

func buildLocalURL(path string, cfg serverConfig) string {
	return fmt.Sprintf("%s://%s%s", cfg.Scheme, cfg.Host, path)
}

// To authenticate as the GitHub App (to create access tokens, for example), requires a JWT:
// https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/#authenticating-as-a-github-app
func makejwt(cfg appConfig) (string, error) {
	iss := cfg.ID // use the GitHub App's numeric ID (not the App ID/Client ID)
	now := time.Now()
	dur := 10 * time.Minute // this token will last 10 minutes
	iat := int(now.Unix())
	exp := int(now.Add(dur).Unix())

	pk, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(cfg.PEM))
	if err != nil {
		return "", err
	}

	method := jwt.GetSigningMethod("RS256")

	claims := make(jwt.MapClaims)
	claims["iss"] = iss
	claims["iat"] = iat
	claims["exp"] = exp

	err = claims.Valid()
	if err != nil {
		return "", err
	}

	tok := jwt.NewWithClaims(method, claims)
	signed, err := tok.SignedString(pk)

	return signed, err
}

///////////////////////////////////////////////////////////////////////////////
// CONFIG
///////////////////////////////////////////////////////////////////////////////

// Server settings.
type serverConfig struct {
	Scheme                string `yaml:"scheme"`
	Host                  string `yaml:"host"`
	GitHubScheme          string `yaml:"gh_scheme"`
	GitHubHost            string `yaml:"gh_host"`
	GitHubAppsPrefix      string `yaml:"gh_apps_prefix"`
	GitHubAPIHost         string `yaml:"gh_api_host"`
	GitHubAPIPrefix       string `yaml:"gh_api_prefix"`
	CreateAppPath         string `yaml:"create_app_path"`
	FetchRegistrationPath string `yaml:"fetch_registration_path"`
	InstallPath           string `yaml:"install_path"`
	Manifest              manifest
}

// Registration details for the GitHub App used to execute requests as the newly registered app.
type appConfig struct {
	AppID          string `json:"client_id" yaml:"app_id"`
	AppSecret      string `json:"client_secret" yaml:"app_secret"`
	PEM            string `json:"pem" yaml:"pem"`
	ID             int    `json:"id" yaml:"id"`
	NodeID         string `json:"node_id" yaml:"node_id"`
	Slug           string `json:"slug" yaml:"slug"`
	InstallationID int    `json:"installation_id" yaml:"installation_id"`
}

// App registration manifest.
type manifest struct {
	Name               string            `yaml:"name" json:"name"`
	URL                string            `yaml:"url" json:"url"`
	RedirectURL        string            `yaml:"redirect_url" json:"redirect_url"`
	SetupURL           string            `yaml:"setup_url" json:"setup_url"`
	Public             bool              `yaml:"public" json:"public"`
	DefaultPermissions map[string]string `yaml:"default_permissions" json:"default_permissions"`
	HookAttributes     manifestHookAttrs `yaml:"hook_attributes" json:"hook_attributes"`
	DefaultEvents      []string          `yaml:"default_events" json:"default_events"`
	FormValue          string            `json:"-"` // exclude from JSON output
}
type manifestHookAttrs struct {
	URL    string `yaml:"url" json:"url"`
	Active bool   `yaml:"active" json:"active"`
}

// Load config/server.yml and set default configuration values.
func loadServerConfig() (serverConfig, error) {
	// setup defaults
	cfg := serverConfig{
		Scheme: "http",
		Host:   "localhost:5000",

		GitHubScheme:     "https",
		GitHubHost:       "github.com",
		GitHubAPIHost:    "api.github.com",
		GitHubAPIPrefix:  "",
		GitHubAppsPrefix: "apps",

		// GitHub endpoints
		CreateAppPath:         "/settings/apps/new",
		FetchRegistrationPath: "/app-manifests/%s/conversions",
		InstallPath:           "/%s/%s/installations/new",
	}

	yamlstr, err := ioutil.ReadFile(serverCfgPath)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(yamlstr, &cfg)
	if err != nil {
		return cfg, err
	}

	// load and inject manifest into server config
	man, err := loadManifest(cfg)
	if err != nil {
		return serverConfig{}, err
	}
	cfg.Manifest = man

	return cfg, nil
}

// load the app manifest from config/manifest.yml
func loadManifest(cfg serverConfig) (manifest, error) {
	var man manifest

	yamlstr, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return man, err
	}
	err = yaml.Unmarshal(yamlstr, &man)
	if err != nil {
		return man, err
	}

	// update manifest to include post-installation redirect_url parameter
	man.RedirectURL = buildLocalURL("/registered", cfg)
	man.SetupURL = buildLocalURL("/installed", cfg)

	// update manifest to include JSON
	man.FormValue, err = man.AsJSON()

	return man, err
}

// helper to generate JSON version of the app manifest for embedding in the form.
func (m manifest) AsJSON() (string, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

// reads registered/installed app details from config/app.yml
// NOTE: failing to load the config should not be a fatal failure as this file
// should only be created when the app is registered.
func loadAppConfig() (appConfig, error) {
	var cfg appConfig

	yamlstr, err := ioutil.ReadFile(appCfgPath)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(yamlstr, &cfg)
	return cfg, err
}

// used to persist app registration and installation details to config/app.yml
func writeAppConfig(cfg appConfig) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	ioutil.WriteFile(appCfgPath, b, 0600)

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// TEMPLATING
///////////////////////////////////////////////////////////////////////////////

func loadTemplates() error {
	var err error
	templates, err = template.ParseGlob("./templates/*.html.tpl")
	return err
}

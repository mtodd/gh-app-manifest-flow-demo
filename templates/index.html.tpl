<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
		<h1>GitHub App <tt>{{ .Manifest.Name }}</tt></h1>
		{{ if .AppID }}
			<h3>
				is registered
				{{ if .InstallationID }} and installed{{ end }}
			</h3>
		{{ else }}
			<h3>
				is not yet registered or installed
			</h3>
		{{ end }}

		<h2>
			{{ if .AppID }}✅{{else}}⚠️{{end}}
			Step 1: Register GitHub App
		</h2>
		<div>
			{{ if .AppID }}
				GitHub App has been registered with these details:
				<div>Slug: <tt>{{ .Slug }}</tt></div>
				<div>App ID: <tt>{{ .AppID }}</tt> (<tt>{{ .ID }}</tt>)</div>
				<div>Secret: <tt>{{ .AppSecret }}</tt></div>
				<details>
					<summary>PEM</summary>
					<tt>{{ .PEM }}</tt>
				</details>
			{{ else }}
				<form action="{{ .CreateAppURL }}" method="post">
					<input type="hidden" name="redirect_url" value="{{ .Manifest.RedirectURL }}">
					<button class="btn btn-outline" name="manifest" id="manifest" value="{{ .Manifest.FormValue }}">
						Register GitHub App
					</button>
				</form>
			{{ end }}
		</div>

		<h2>
			{{ if .AppID }}{{ if .InstallationID }}✅{{else}}⚠️{{end}}{{else}}🛑{{end}}
			Step 2: Install GitHub App
		</h2>
		<div>
			{{ if .InstallationID }}
				<div>Installation ID: <tt>{{ .InstallationID }}</tt></div>
			{{ else }}
				{{ if .AppID }}
					<a href="{{ .InstallURL }}" class="btn btn-outline">Install GitHub App</a>
				{{ else }}
					Registering the GitHub App in <strong>Step 1</strong> is required before installing the app.
				{{ end }}
			{{ end }}
		</div>

		<h2>
			{{ if and (.AppID) (.InstallationID) }}✅{{else}}🛑{{end}}
			Step 3: Use the GitHub App
		</h2>
		<div>
			{{ if and (.AppID) (.InstallationID) }}
				{{ if .AppAuthedJSON }}
				<details>
					<summary>Authenticating as GitHub App (<a href="https://developer.github.com/v3/apps/#get-the-authenticated-github-app">endpoint</a>)</summary>
					<div>NOTE: This request authenticates as the GitHub App, but does not authenticate for acecss against the installation
						target.</div>
					<tt>{{ .AppAuthedJSON }}</tt>
				</details>
				{{ end }}
				{{ if .AccessToken }}
				<details>
					<summary>Requesting access token for installation of GitHub App (<a href="https://developer.github.com/v3/apps/#create-a-new-installation-token">endpoint</a>)</summary>
					<div>Access token: <tt>{{ .AccessToken.Token }}</tt> (expires: <tt>{{ .AccessToken.ExpiresAt }}</tt>)</div>
					<tt>{{ .AccessTokenJSON }}</tt>
				</details>
				{{ end }}
				{{ if .InstallationReposJSON }}
				<details>
					<summary>Authenticating with access token (<a href="https://developer.github.com/v3/apps/installations/#list-repositories">endpoint</a>)</summary>
					<div>NOTE: This request authenticates using the newly created access token. The list of repositories may be empty but
						that does not mean it failed.</div>
					<tt>{{ .InstallationReposJSON }}</tt>
				</details>
				{{ end }}
			{{ else }}
				App must be registered and installed first.
			{{ end }}
		</div>

	</body>
</html>

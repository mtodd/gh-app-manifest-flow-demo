<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
		{{ if .AppID }}
			<h1>
				GitHub App <tt>{{ .Manifest.Name }}</tt> is registered
				{{ if .InstallationID }} and installed{{ end }}
			</h1>
			<div>
				<div>
					{{ if .InstallationID }}
						Installation ID: <tt>{{ .InstallationID }}</tt>
					{{ else }}
						<h2>
							Step 2: Install GitHub App <tt>{{ .Manifest.Name }}</tt>
						</h2>
						<a href="{{ .InstallURL }}" class="btn btn-outline">Install GitHub App</a>
					{{ end }}
				</div>
				<div>Slug: <tt>{{ .Slug }}</tt></div>
				<div>App ID: <tt>{{ .AppID }}</tt> (<tt>{{ .ID }}</tt>)</div>
				<div>Secret: <tt>{{ .AppSecret }}</tt></div>
				{{ if .AppAuthedJSON }}
					<details>
						<summary>Authenticating as GitHub App (<a href="https://developer.github.com/v3/apps/#get-the-authenticated-github-app">endpoint</a>)</summary>
						<div>NOTE: This request authenticates as the GitHub App, but does not authenticate for acecss against the installation target.</div>
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
						<div>NOTE: This request authenticates using the newly created access token. The list of repositories may be empty but that does not mean it failed.</div>
						<tt>{{ .InstallationReposJSON }}</tt>
					</details>
				{{ end }}
			</div>
		{{ else }}
			<h1>
				GitHub App <tt>{{ .Manifest.Name }}</tt> is not yet registered or installed
			</h1>
			<h2>
				Step 1: Register GitHub App <tt>{{ .Manifest.Name }}</tt>
			</h2>
			<form action="{{ .CreateAppURL }}" method="post">
				<input type="hidden" name="redirect_url" value="{{ .Manifest.RedirectURL }}">
				<button class="btn btn-outline" name="manifest" id="manifest" value="{{ .Manifest.FormValue }}">Register GitHub App</button>
			</form>
			<div>Step 2 will be to install the registered GitHub App.</div>
		{{ end }}
	</body>
</html>

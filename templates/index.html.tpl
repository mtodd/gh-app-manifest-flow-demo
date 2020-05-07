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
				<div>Slug: <tt>{{ .Slug }}</tt></div>
				<div>App ID: <tt>{{ .AppID }}</tt> (<tt>{{ .ID }}</tt>)</div>
				<div>Secret: <tt>{{ .AppSecret }}</tt></div>
				<div>
					{{ if .InstallationID }}
						Installation ID: <tt>{{ .InstallationID }}</tt>
					{{ else }}
						<a href="{{ .InstallURL }}" class="btn btn-outline">Install GitHub App</a>
					{{ end }}
				</div>
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
				Register GitHub App <tt>{{ .Manifest.Name }}</tt>
			</h1>
			<form action="{{ .CreateAppURL }}" method="post" target="_blank">
				<input type="hidden" name="redirect_url" value="{{ .Manifest.RedirectURL }}">
				<button class="btn btn-outline" name="manifest" id="manifest" value="{{ .Manifest.FormValue }}">Register GitHub App</button>
			</form>
		{{ end }}
	</body>
</html>

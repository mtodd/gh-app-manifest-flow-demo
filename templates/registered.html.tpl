<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
		<h1>
			GitHub App <tt>{{ .Manifest.Name }}</tt> is registered
			{{ if .InstallationID }} and installed{{ end }}
		</h1>
		{{ if .InstallationID }}
			The application has been installed!
			<a href="/" class="btn btn-outline">Return home</a>
		{{ else }}
			<h2>
				Step 2: Install GitHub App <tt>{{ .Manifest.Name }}</tt>
			</h2>
			<a href="{{ .InstallURL }}" class="btn btn-outline">Install GitHub App</a>
	  {{ end }}
		<div>
	    <div>code: <tt>{{ .Code }}</tt></div>
			<div>id: <tt>{{ .Slug }}</tt></div>
			<div>id: <tt>{{ .AppID }}</tt></div>
			<div>secret: <tt>{{ .AppSecret }}</tt></div>
			<div>pem: <tt>{{ .PEM }}</tt></div>
		</div>
	</body>
</html>

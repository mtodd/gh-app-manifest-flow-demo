<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
    <h1>Registered!</h1>
		{{ if .InstallationID }}
			The application has been installed!
			<a href="/" class="btn btn-outline">Return home</a>
		{{ else }}
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

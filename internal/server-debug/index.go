package server_debug

import (
	"html/template"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type page struct {
	Path        string
	Description string
}

type indexPage struct {
	pages []page
}

func newIndexPage() *indexPage {
	return &indexPage{}
}

func (i *indexPage) addPage(path string, description string) {
	i.pages = append(i.pages, page{
		Path:        path,
		Description: description,
	})
}

func (i *indexPage) handler(eCtx echo.Context) error {
	return template.Must(template.New("index").Parse(`
<html>
	<title>Chat Service Debug</title>
<body>
	<h2>Chat Service Debug</h2>
	<ul>
		{{ range $page := .Pages }}
		<li><a href="{{ $page.Path }}">{{ $page.Path }}</a> {{ $page.Description }}</li>
		{{ end }}
	</ul>

	<h2>Log Level</h2>
	<form onSubmit="putLogLevel()">
		<select id="log-level-select">
			{{ range $level := .Levels }}
				{{ if eq $.LogLevel $level }}
					<option value="{{ $level }}" selected>{{ $level }}</option>
				{{ else }}
					<option value="{{ $level }}">{{ $level }}</option>
				{{ end }}
			{{ end }}
		</select>
		<input type="submit" value="Change"></input>
	</form>
	
	<script>
		function putLogLevel() {
			const req = new XMLHttpRequest();
			req.open('PUT', '/log/level', false);
			req.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
			req.onload = function() { window.location.reload(); };
			req.send('level='+document.getElementById('log-level-select').value);
		};
	</script>
</body>
</html>
`)).Execute(eCtx.Response(), struct {
		Pages    []page
		LogLevel string
		Levels   []string
	}{
		Pages:    i.pages,
		LogLevel: zap.L().Level().String(),
		Levels:   []string{"debug", "info", "warn", "error"},
	})
}

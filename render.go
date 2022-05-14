package main

import (
	"bufio"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		ex, _ := os.Executable()
		return fmt.Errorf("Usage:\n\t%v OUTPUT_DIRECTORY", ex)
	}
	dir := args[0]

	s := bufio.NewScanner(os.Stdin)
	for i := 0; s.Scan(); i++ {
		fields := strings.Fields(s.Text())
		if len(fields) != 2 {
			return fmt.Errorf("invalid input line %d, must contain 2 fields: %v", i+1, s.Text())
		}
		d := description{
			Dir:        dir,
			Module:     fields[0],
			ImportPath: fields[1],
			GitRepo:    repos[fields[0]],
		}
		if err := render(d); err != nil {
			return fmt.Errorf("failed to render %v", d.ImportPath)
		}
	}
	return s.Err()
}

func render(d description) error {
	p := d.Path()
	_ = os.MkdirAll(filepath.Dir(p), 0755)

	fh, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	return tmpl.Execute(fh, d)
}

type description struct {
	Dir        string
	Module     string
	ImportPath string
	GitRepo    string
}

func (d description) Path() string {
	relative := strings.TrimPrefix(d.ImportPath, "gotest.tools")
	return filepath.Join(d.Dir, relative, "index.html")
}

var repos = map[string]string{
	"gotest.tools/gotestsum": "https://github.com/gotestyourself/gotestsum",
	"gotest.tools/v3":        "https://github.com/gotestyourself/gotest.tools",
	"gotest.tools":           "https://github.com/gotestyourself/gotest.tools",
}

var tmpl = template.Must(template.New("vanity").Parse(`
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Module}} git {{.GitRepo}}">
<meta http-equiv="refresh" content="0; url=https://pkg.go.dev/{{.ImportPath}}">
</head>
<body>
Nothing to see here; <a href="https://pkg.go.dev/{{.ImportPath}}">see the package on pkg.go.dev</a>.
</body>
</html>
`))

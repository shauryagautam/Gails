package framework

import (
	"bufio"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"go.uber.org/zap"
)

type StackFrame struct {
	Function string
	File     string
	Line     int
	Code     []string
	IsUser   bool
}

type ErrorPageData struct {
	ErrorType      string
	Message        string
	File           string
	Line           int
	Column         int
	StackTrace     []StackFrame
	RequestMethod  string
	RequestURL     string
	RequestHeaders http.Header
	Env            string
	GoVersion      string
	GailsVersion   string
}

func DevErrorHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := ErrorPageData{
		ErrorType:      fmt.Sprintf("%T", err),
		Message:        fmt.Sprintf("%v", err),
		RequestMethod:  r.Method,
		RequestURL:     r.URL.String(),
		RequestHeaders: r.Header,
		Env:            os.Getenv("APP_ENV"),
		GoVersion:      runtime.Version(),
		GailsVersion:   "v1.0.0",
	}

	stack := debug.Stack()
	data.StackTrace = parseStackTrace(stack)

	if len(data.StackTrace) > 0 {
		data.File = data.StackTrace[0].File
		data.Line = data.StackTrace[0].Line
	}

	funcs := template.FuncMap{
		"contains": strings.Contains,
	}

	tmpl := template.Must(template.New("error").Funcs(funcs).Parse(errorTemplate))
	tmpl.Execute(w, data)
}

func ProdErrorHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"error": "Something went wrong"}`)
	Log.Error("Panic recovered", zap.Any("panic", err))
}

func parseStackTrace(stack []byte) []StackFrame {
	lines := strings.Split(string(stack), "\n")
	var frames []StackFrame
	// debug.Stack() output starts with "goroutine ... [running]:" and then pairs of lines: function and file:line
	for i := 1; i < len(lines)-1; i += 2 {
		line1 := strings.TrimSpace(lines[i])
		line2 := strings.TrimSpace(lines[i+1])
		if line1 == "" || line2 == "" {
			continue
		}

		// Parse file and line from line2 (e.g., "/path/to/file.go:line +0xabc")
		parts := strings.Split(line2, ":")
		if len(parts) < 2 {
			continue
		}
		file := parts[0]
		lineParts := strings.Split(parts[1], " ")
		var lineNum int
		fmt.Sscanf(lineParts[0], "%d", &lineNum)

		frame := StackFrame{
			Function: line1,
			File:     file,
			Line:     lineNum,
			IsUser:   !strings.Contains(file, "runtime/") && !strings.Contains(file, "github.com/shaurya/gails/framework"),
		}

		if frame.IsUser {
			frame.Code = getSourceSnippet(file, lineNum)
		}

		frames = append(frames, frame)
	}
	return frames
}

func getSourceSnippet(file string, line int) []string {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()

	var snippet []string
	scanner := bufio.NewScanner(f)
	currentLine := 0
	start := line - 5
	end := line + 5

	for scanner.Scan() {
		currentLine++
		if currentLine >= start && currentLine <= end {
			prefix := "  "
			if currentLine == line {
				prefix = "> "
			}
			snippet = append(snippet, fmt.Sprintf("%d: %s%s", currentLine, prefix, scanner.Text()))
		}
		if currentLine > end {
			break
		}
	}
	return snippet
}

const errorTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Gails Error: {{.Message}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #fdfdfd; color: #333; margin: 0; padding: 20px; }
        .container { max-width: 1000px; margin: 0 auto; }
        header { border-bottom: 2px solid #e2e2e2; padding-bottom: 20px; margin-bottom: 30px; }
        h1 { color: #d9534f; margin: 0 0 10px 0; font-size: 24px; }
        .message { font-size: 18px; color: #444; }
        .file-info { font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace; font-size: 14px; margin-top: 10px; color: #666; }
        .section { margin-bottom: 30px; border: 1px solid #e2e2e2; border-radius: 4px; overflow: hidden; background: #fff; }
        .section-header { background: #f7f7f7; padding: 10px 15px; border-bottom: 1px solid #e2e2e2; font-weight: bold; display: flex; justify-content: space-between; align-items: center; }
        .section-body { padding: 0; }
        pre { margin: 0; padding: 15px; font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace; font-size: 13px; line-height: 1.5; overflow-x: auto; }
        .snippet .line { display: block; }
        .snippet .highlight { background: #feebeb; color: #b94a48; font-weight: bold; width: 100%; display: inline-block; }
        .stack-trace .frame { padding: 10px 15px; border-bottom: 1px solid #f0f0f0; cursor: pointer; }
        .stack-trace .frame:hover { background: #f9f9f9; }
        .stack-trace .frame.framework { color: #999; }
        .stack-trace .func { font-weight: bold; }
        .stack-trace .file { font-size: 12px; margin-top: 4px; }
        table { width: 100%; border-collapse: collapse; font-size: 13px; }
        th, td { text-align: left; padding: 8px 15px; border-bottom: 1px solid #eee; }
        th { background: #fcfcfc; color: #777; width: 30%; }
        .copy-btn { font-size: 12px; padding: 4px 8px; border: 1px solid #ccc; background: #fff; border-radius: 3px; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>{{.ErrorType}}</h1>
            <div class="message">{{.Message}}</div>
            <div class="file-info">{{.File}}:{{.Line}}</div>
        </header>

        <div class="section">
            <div class="section-header">Source</div>
            <div class="section-body">
                <pre class="snippet">{{range .StackTrace}}{{if .IsUser}}{{range .Code}}{{ if (contains . "> ") }}<span class="highlight">{{.}}</span>{{else}}<span class="line">{{.}}</span>{{end}}{{end}}{{end}}{{break}}{{end}}</pre>
            </div>
        </div>

        <div class="section">
            <div class="section-header">Stack Trace <button class="copy-btn" onclick="copyError()">Copy</button></div>
            <div class="section-body stack-trace">
                {{range .StackTrace}}
                <div class="frame {{if not .IsUser}}framework{{end}}">
                    <div class="func">{{.Function}}</div>
                    <div class="file">{{.File}}:{{.Line}}</div>
                </div>
                {{end}}
            </div>
        </div>

        <div class="section">
            <div class="section-header">Request Info</div>
            <div class="section-body">
                <table>
                    <tr><th>Method</th><td>{{.RequestMethod}}</td></tr>
                    <tr><th>URL</th><td>{{.RequestURL}}</td></tr>
                </table>
            </div>
        </div>
        
        <div class="section">
            <div class="section-header">Request Headers</div>
            <div class="section-body">
                <table>
                    {{range $k, $v := .RequestHeaders}}
                    <tr><th>{{$k}}</th><td>{{range $v}}{{.}}<br>{{end}}</td></tr>
                    {{end}}
                </table>
            </div>
        </div>

        <div class="section">
            <div class="section-header">App Info</div>
            <div class="section-body">
                <table>
                    <tr><th>Environment</th><td>{{.Env}}</td></tr>
                    <tr><th>Gails Version</th><td>{{.GailsVersion}}</td></tr>
                </table>
            </div>
        </div>
    </div>
    <script>
        function copyError() {
            const text = document.querySelector('header').innerText + "\n\n" + 
                        document.querySelector('.stack-trace').innerText;
            navigator.clipboard.writeText(text);
            alert('Copied to clipboard');
        }
        function contains(str, substr) {
            return str.indexOf(substr) !== -1;
        }
    </script>
</body>
</html>
`

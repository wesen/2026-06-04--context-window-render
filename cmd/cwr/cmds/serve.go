package cmds

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/context-window-render/pkg/cwr/dsl"
	"github.com/go-go-golems/context-window-render/pkg/cwr/renderer/ascii"
	"github.com/go-go-golems/context-window-render/pkg/cwr/renderer/svg"
	"github.com/go-go-golems/context-window-render/pkg/cwr/theme"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
)

// ServeCommand starts an HTTP server with one page per YAML diagram,
// showing SVG, ASCII, and DSL source side by side.
type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	Dir  string `glazed:"dir"`
	Port int    `glazed:"port"`
}

func NewServeCommand() (*ServeCommand, error) {
	glazedSection, err := settings.NewGlazedSchema(
		settings.WithOutputSectionOptions(
			schema.WithDefaults(map[string]interface{}{
				"output": "table",
			}),
		),
	)
	if err != nil {
		return nil, err
	}

	cmdDesc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Serve diagrams: one page per YAML with SVG+ASCII+DSL side by side"),
		cmds.WithLong(`
Start an HTTP server that renders all YAML diagrams in a directory.
Each diagram gets its own page with SVG, ASCII, and YAML source shown
side by side. Diagrams are re-rendered when YAML files change —
just refresh the browser.

Routes:
  /              — Index listing all diagrams
  /d/{name}      — Diagram page: SVG + ASCII + YAML side by side
  /svg/{name}    — Raw SVG output
  /yaml/{name}   — Raw YAML source

Examples:
  cwr serve --dir examples --port 8080
  cwr serve --port 3000
`),
		cmds.WithFlags(
			fields.New(
				"dir",
				fields.TypeString,
				fields.WithDefault("examples"),
				fields.WithHelp("Directory containing YAML diagram files"),
			),
			fields.New(
				"port",
				fields.TypeInteger,
				fields.WithDefault(8080),
				fields.WithHelp("HTTP port to listen on"),
			),
		),
		cmds.WithSections(glazedSection),
	)

	return &ServeCommand{CommandDescription: cmdDesc}, nil
}

func (c *ServeCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	settings := &ServeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	srv := newGalleryServer(settings.Dir, settings.Port)
	return srv.run(ctx)
}

// --- rendered diagram cache ---

type renderedDiagram struct {
	Name  string
	File  string
	SVG   string
	ASCII string
	YAML  string
	Error string
}

type galleryServer struct {
	dir      string
	port     int
	mu       sync.RWMutex
	diagrams map[string]renderedDiagram
	modTimes map[string]time.Time
}

func newGalleryServer(dir string, port int) *galleryServer {
	return &galleryServer{
		dir:      dir,
		port:     port,
		diagrams: make(map[string]renderedDiagram),
		modTimes: make(map[string]time.Time),
	}
}

func (s *galleryServer) run(ctx context.Context) error {
	s.renderAll()
	go s.watchFiles(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/d/", s.handleDiagram)
	mux.HandleFunc("/svg/", s.handleRawSVG)
	mux.HandleFunc("/yaml/", s.handleRawYAML)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Context Window Render:\n")
	fmt.Printf("  http://localhost%s/          — Index\n", addr)
	fmt.Printf("  http://localhost%s/d/{name}  — Diagram (SVG+ASCII+YAML)\n", addr)
	fmt.Printf("Watching %s for changes (refresh browser to update)\n", s.dir)

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()
	return server.ListenAndServe()
}

// --- render pipeline ---

func (s *galleryServer) renderAll() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		return
	}

	th := theme.DefaultTheme()
	newDiagrams := make(map[string]renderedDiagram)
	newModTimes := make(map[string]time.Time)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		path := filepath.Join(s.dir, entry.Name())
		info, _ := entry.Info()
		newModTimes[path] = info.ModTime()

		rd := renderedDiagram{Name: name, File: path}

		data, err := os.ReadFile(path)
		if err != nil {
			rd.Error = err.Error()
			newDiagrams[name] = rd
			continue
		}
		rd.YAML = string(data)

		diagram, err := dsl.ParseDiagram(data)
		if err != nil {
			rd.Error = err.Error()
			newDiagrams[name] = rd
			continue
		}

		if svgContent, err := svg.NewRenderer(th).Render(diagram); err != nil {
			rd.Error = err.Error()
		} else {
			rd.SVG = svgContent
		}

		if asciiContent, err := ascii.NewRenderer(th).Render(diagram); err != nil {
			if rd.Error == "" {
				rd.Error = err.Error()
			}
		} else {
			rd.ASCII = asciiContent
		}

		newDiagrams[name] = rd
	}

	s.mu.Lock()
	s.diagrams = newDiagrams
	s.modTimes = newModTimes
	s.mu.Unlock()
}

func (s *galleryServer) watchFiles(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkChanges()
		}
	}
}

func (s *galleryServer) checkChanges() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	changed := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(s.dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		s.mu.RLock()
		oldTime, exists := s.modTimes[path]
		s.mu.RUnlock()
		if !exists || info.ModTime().After(oldTime) {
			changed = true
			break
		}
	}
	if changed {
		fmt.Println("YAML changed — re-rendering...")
		s.renderAll()
	}
}

func (s *galleryServer) sortedNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.diagrams))
	for n := range s.diagrams {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// --- HTTP handlers ---

func (s *galleryServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	names := s.sortedNames()
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("index").Parse(indexTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"Names":    names,
		"Diagrams": s.diagrams,
	})
}

func (s *galleryServer) handleDiagram(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/d/")
	s.mu.RLock()
	rd, ok := s.diagrams[name]
	s.mu.RUnlock()
	if !ok {
		http.Error(w, "diagram not found", 404)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("diagram").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}).Parse(diagramTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"RD":    rd,
		"Names": s.sortedNames(),
	})
}

func (s *galleryServer) handleRawSVG(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/svg/")
	s.mu.RLock()
	rd, ok := s.diagrams[name]
	s.mu.RUnlock()
	if !ok {
		http.Error(w, "not found", 404)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write([]byte(rd.SVG))
}

func (s *galleryServer) handleRawYAML(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/yaml/")
	s.mu.RLock()
	rd, ok := s.diagrams[name]
	s.mu.RUnlock()
	if !ok {
		http.Error(w, "not found", 404)
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Write([]byte(rd.YAML))
}

// --- Templates ---

const commonStyle = `
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
    background: #FFFFFF; color: #000000; padding: 24px; line-height: 1.6;
  }
  h1 { font-size: 20px; font-weight: bold; margin-bottom: 4px; }
  .subtitle { font-size: 11px; color: #888888; margin-bottom: 24px; }
  nav { margin-bottom: 24px; font-size: 12px; display: flex; flex-wrap: wrap; gap: 6px 14px; }
  nav a { color: #000000; text-decoration: none; }
  nav a:hover { text-decoration: underline; }
  nav .current { font-weight: bold; }
  .error {
    font-size: 12px; color: #CC0000; padding: 8px;
    border: 1px solid #CC0000; background: #FFF0F0; margin-bottom: 16px;
  }
  .panels {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 20px;
    align-items: start;
  }
  .panel { min-width: 0; overflow: hidden; }
  .panel-label {
    font-size: 10px; color: #888888; text-transform: uppercase;
    letter-spacing: 1px; margin-bottom: 6px; border-bottom: 1px solid #CCCCCC; padding-bottom: 3px;
  }
  .svg-wrap { border: 1px solid #000000; display: inline-block; max-width: 100%; overflow: hidden; }
  .svg-wrap svg { display: block; max-width: 100%; height: auto; }
  .ascii-wrap {
    border: 1px solid #CCCCCC; padding: 12px;
    font-size: 12px; white-space: pre; overflow-x: auto;
  }
  .yaml-wrap {
    border: 1px solid #CCCCCC; padding: 12px;
    font-size: 11px; white-space: pre; overflow-x: auto; overflow-y: auto;
    max-height: 600px; background: #F8F8F8;
  }
  .index-list { list-style: none; }
  .index-list li { margin-bottom: 4px; }
  .index-list a { color: #000000; text-decoration: none; }
  .index-list a:hover { text-decoration: underline; }
  .index-list .file { color: #888888; font-size: 11px; margin-left: 8px; }
`

const indexTemplate = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Context Window Render</title>
<style>` + commonStyle + `</style></head>
<body>
<h1>Context Window Render</h1>
<div class="subtitle">YAML DSL for LLM context window diagrams</div>
<ul class="index-list">
{{range $name := .Names}}
{{with $rd := index $.Diagrams $name}}
<li><a href="/d/{{$rd.Name}}">{{$rd.Name}}</a><span class="file">{{$rd.File}}</span>
{{if $rd.Error}}<span style="color:#CC0000"> — error</span>{{end}}</li>
{{end}}
{{end}}
</ul>
</body></html>`

const diagramTemplate = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>{{.RD.Name}} — Context Window Render</title>
<style>` + commonStyle + `</style></head>
<body>
<h1>{{.RD.Name}}</h1>
<div class="subtitle">{{.RD.File}}</div>
<nav>
{{range .Names}}
<a href="/d/{{.}}">{{.}}</a>
{{end}}
</nav>

{{if .RD.Error}}<div class="error">Error: {{.RD.Error}}</div>{{end}}

<div class="panels">
  <div class="panel">
    <div class="panel-label">SVG</div>
    {{if .RD.SVG}}<div class="svg-wrap">{{.RD.SVG | safeHTML}}</div>{{end}}
  </div>
  <div class="panel">
    <div class="panel-label">ASCII</div>
    {{if .RD.ASCII}}<div class="ascii-wrap">{{.RD.ASCII}}</div>{{end}}
  </div>
  <div class="panel">
    <div class="panel-label">DSL</div>
    {{if .RD.YAML}}<div class="yaml-wrap">{{.RD.YAML}}</div>{{end}}
  </div>
</div>

</body></html>`

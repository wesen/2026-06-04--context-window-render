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

// ServeCommand starts an HTTP server that serves SVG and ASCII galleries
// and re-renders when YAML files change.
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
		cmds.WithShort("Serve SVG and ASCII galleries with hot-reload"),
		cmds.WithLong(`
Start an HTTP server that renders all YAML diagrams in a directory.
SVG and ASCII are served on separate pages. Diagrams are re-rendered
automatically when YAML files change — just refresh the browser.

Routes:
  /         — Index with links to both galleries
  /svg      — SVG gallery
  /ascii    — ASCII gallery
  /svg/{name} — Single SVG diagram
  /yaml/{name} — Raw YAML source

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

// --- gallery server ---

type galleryServer struct {
	dir      string
	port     int
	mu       sync.RWMutex
	diagrams map[string]renderedDiagram // sorted by name
	modTimes map[string]time.Time
}

type renderedDiagram struct {
	Name  string
	File  string
	SVG   string
	ASCII string
	YAML  string
	Error string
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
	mux.HandleFunc("/svg", s.handleSVGGallery)
	mux.HandleFunc("/ascii", s.handleASCIIGallery)
	mux.HandleFunc("/svg/", s.handleSingleSVG)
	mux.HandleFunc("/yaml/", s.handleSingleYAML)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Context Window Render gallery:\n")
	fmt.Printf("  http://localhost%s/       — Index\n", addr)
	fmt.Printf("  http://localhost%s/svg     — SVG gallery\n", addr)
	fmt.Printf("  http://localhost%s/ascii   — ASCII gallery\n", addr)
	fmt.Printf("Watching %s for changes (refresh browser to see updates)\n", s.dir)

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()
	return server.ListenAndServe()
}

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

		svgR := svg.NewRenderer(th)
		if svgContent, err := svgR.Render(diagram); err != nil {
			rd.Error = err.Error()
		} else {
			rd.SVG = svgContent
		}

		asciiR := ascii.NewRenderer(th)
		if asciiContent, err := asciiR.Render(diagram); err != nil {
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
	for name := range s.diagrams {
		names = append(names, name)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, indexPage, s.port)
}

func (s *galleryServer) handleSVGGallery(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := s.sortedNames()

	data := map[string]interface{}{
		"Names":    names,
		"Diagrams": s.diagrams,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("svg").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}).Parse(svgGalleryTemplate))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *galleryServer) handleASCIIGallery(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := s.sortedNames()

	data := map[string]interface{}{
		"Names":    names,
		"Diagrams": s.diagrams,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("ascii").Parse(asciiGalleryTemplate))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *galleryServer) handleSingleSVG(w http.ResponseWriter, r *http.Request) {
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

func (s *galleryServer) handleSingleYAML(w http.ResponseWriter, r *http.Request) {
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

// --- HTML templates ---

const commonStyle = `
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
    background: #FFFFFF;
    color: #000000;
    padding: 40px;
    line-height: 1.6;
  }
  h1 { font-size: 22px; font-weight: bold; margin-bottom: 4px; letter-spacing: -0.5px; }
  .subtitle { font-size: 11px; color: #888888; margin-bottom: 32px; }
  nav { margin-bottom: 32px; font-size: 12px; }
  nav a { color: #000000; text-decoration: underline; margin-right: 16px; }
  nav a:hover { color: #444444; }
  nav .current { font-weight: bold; text-decoration: none; }
  .example { margin-bottom: 60px; }
  .example-header {
    display: flex; align-items: baseline; gap: 12px;
    margin-bottom: 12px; border-bottom: 1px solid #000000; padding-bottom: 8px;
  }
  .example-name { font-size: 15px; font-weight: bold; }
  .example-file { font-size: 11px; color: #888888; }
  .example-error {
    font-size: 12px; color: #CC0000; margin-bottom: 12px;
    padding: 8px; border: 1px solid #CC0000; background: #FFF0F0;
  }
  .svg-container { border: 1px solid #000000; display: inline-block; margin-bottom: 8px; }
  .svg-container svg { display: block; }
  .ascii-container {
    border: 1px solid #CCCCCC; padding: 16px;
    font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
    font-size: 13px; white-space: pre; overflow-x: auto; color: #000000;
    max-width: 800px;
  }
  .yaml-link {
    font-size: 10px; color: #888888; margin-top: 4px;
  }
  .yaml-link a { color: #666666; }
`

const indexPage = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Context Window Render</title>
<style>` + commonStyle + `</style></head>
<body>
<h1>Context Window Render</h1>
<div class="subtitle">YAML DSL for LLM context window diagrams</div>
<nav>
  <a href="/svg">SVG Gallery</a>
  <a href="/ascii">ASCII Gallery</a>
</nav>
<p style="font-size:13px;max-width:600px">
  Edit YAML files in the <code>examples/</code> directory.
  Diagrams are re-rendered automatically when files change.
  Refresh the browser to see updates.
</p>
</body></html>`

const svgGalleryTemplate = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Context Window Render — SVG</title>
<style>` + commonStyle + `</style></head>
<body>
<h1>Context Window Render — SVG</h1>
<div class="subtitle">Proportional region diagrams rendered as SVG</div>
<nav>
  <a href="/">Index</a>
  <span class="current">SVG</span>
  <a href="/ascii">ASCII</a>
</nav>

{{range $name := .Names}}
{{with $rd := index $.Diagrams $name}}
<div class="example">
  <div class="example-header">
    <span class="example-name">{{$rd.Name}}</span>
    <span class="example-file">{{$rd.File}}</span>
  </div>
  {{if $rd.Error}}<div class="example-error">Error: {{$rd.Error}}</div>{{end}}
  {{if $rd.SVG}}
  <div class="svg-container">{{$rd.SVG | safeHTML}}</div>
  {{end}}
  <div class="yaml-link"><a href="/yaml/{{$rd.Name}}">view YAML source</a></div>
</div>
{{end}}
{{end}}

</body></html>`

const asciiGalleryTemplate = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Context Window Render — ASCII</title>
<style>` + commonStyle + `</style></head>
<body>
<h1>Context Window Render — ASCII</h1>
<div class="subtitle">Terminal-style context window diagrams</div>
<nav>
  <a href="/">Index</a>
  <a href="/svg">SVG</a>
  <span class="current">ASCII</span>
</nav>

{{range $name := .Names}}
{{with $rd := index $.Diagrams $name}}
<div class="example">
  <div class="example-header">
    <span class="example-name">{{$rd.Name}}</span>
    <span class="example-file">{{$rd.File}}</span>
  </div>
  {{if $rd.Error}}<div class="example-error">Error: {{$rd.Error}}</div>{{end}}
  {{if $rd.ASCII}}
  <div class="ascii-container">{{$rd.ASCII}}</div>
  {{end}}
  <div class="yaml-link"><a href="/yaml/{{$rd.Name}}">view YAML source</a></div>
</div>
{{end}}
{{end}}

</body></html>`

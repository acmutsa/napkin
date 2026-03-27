package static

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// GetStaticDir returns the path to static files if they exist
func GetStaticDir() string {
	// Check for dist folder relative to binary
	candidates := []string{
		"./dist",
		"../napkin-app/dist",
		"./napkin-app/dist",
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(dir)
			return absPath
		}
	}
	return ""
}

// SPAHandler serves static files and falls back to index.html for SPA routing
func SPAHandler(staticDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(staticDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, r.URL.Path)

		// Check if file exists
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if it's a directory with index.html
		indexPath := filepath.Join(path, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA client-side routing
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})
}

// SetupStaticHandler registers the SPA handler if static files exist
func SetupStaticHandler(mux *http.ServeMux) {
	staticDir := GetStaticDir()
	if staticDir != "" {
		log.Printf("Serving static files from: %s", staticDir)
		mux.Handle("/", SPAHandler(staticDir))
	}
}

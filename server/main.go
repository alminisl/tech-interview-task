package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	root := flag.String("root", "..", "directory to serve static files from (project root)")
	flag.Parse()

	// Serve the whole project root so the page can reach both /frontend and the
	// /potree submodule build. http.FileServer answers HTTP Range requests, which
	// is what COPC streaming relies on.
	fs := http.FileServer(http.Dir(*root))

	mux := http.NewServeMux()
	mux.Handle("/", fs)

	// Convenience: open "/" straight into the viewer.
	mux.HandleFunc("/index.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/frontend/", http.StatusFound)
	})

	log.Printf("serving %s on http://localhost%s (viewer at /frontend/)", *root, *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

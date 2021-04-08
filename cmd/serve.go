package cmd

/*
Copyright © 2020 Giuseppe Lavagetto

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/lavagetto/memeoid/api"
	"github.com/spf13/cobra"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// serveCmd-managed variables
var gifDir string
var memeDir string
var port int
var tplPath string
var certPath string

// prometheus metric definition
var httpDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "memeoid_http_duration_seconds",
		Help: "Duration of http requests",
	},
	[]string{"path", "gif"},
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "An http server to generate memes on request.",
	Long:  `At the moment memeoid only works with a local filesystem.`,
	Run: func(cmd *cobra.Command, args []string) {
		r := mux.NewRouter()

		// static routes
		setupStaticRoute(r, "/gifs/", gifDir)
		setupStaticRoute(r, "/meme/", memeDir)

		// meme routes
		setupMemeRoutes(r)

		// Handle the root of the site with the controller
		http.Handle("/", r)

		// Add prometheus metrics
		r.Use(telemetryMiddleware)
		r.Path("/metrics").Handler(promhttp.Handler())
		
		// Setup logging
		customLog := handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux)
		
		// Start Http server
		portStr := fmt.Sprintf(":%d", port)
		if certPath != "" {
			key := path.Join(certPath, "privkey.pem")
			fullchain := path.Join(certPath, "fullchain.pem")
			// Add an HSTS header
			r.Use(hstsMiddleware)
			http.ListenAndServeTLS(portStr, fullchain, key, customLog)
		} else {
			http.ListenAndServe(portStr, customLog)
		}
	},
}

func setupStaticRoute(router *mux.Router, uriPrefix string, docRoot string) {
	dir := http.FileServer(http.Dir(docRoot))
	router.PathPrefix(uriPrefix).Handler(http.StripPrefix(uriPrefix, dir))
}

func setupMemeRoutes(router *mux.Router) {
	handler := &api.MemeHandler{
		ImgPath:    gifDir,
		OutputPath: memeDir,
		FontName:   fontName,
		MemeURL:    "meme",
	}
	handler.LoadTemplates(tplPath)

	// Homepage
	router.Path("/").
		Methods("GET", "HEAD").
		HandlerFunc(handler.ListGifs)

	//TODO add required query parameters and validation

	// Form
	router.Path("/generate").
		Queries("from", "{from}").
		Methods("GET", "HEAD").
		HandlerFunc(handler.Form)

	// I "heart" the action api
	router.Path("/w/api.php").
		Queries("from", "{from}").
		Methods("GET").
		HandlerFunc(handler.MemeFromRequest)

	// Thumbnails
	router.Path("/thumb/{width:[0-9]+}x{height:[0-9]+}/{from}").
		Methods("GET", "HEAD").
		HandlerFunc(handler.Preview)
}


func telemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			route := mux.CurrentRoute(r)
			path, _ := route.GetPathTemplate()
			v := mux.Vars(r)
			gif := "-"
			if g, ok := v["from"]; ok {
				gif = g
			}
			timer := prometheus.NewTimer(httpDuration.WithLabelValues(path, gif))
			next.ServeHTTP(w, r)
			timer.ObserveDuration()
		},
	)
}

func hstsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=864000")
			next.ServeHTTP(w, r)
		},
	)
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// flags and configuration settings.
	serveCmd.Flags().StringVarP(&gifDir, "image-dir", "i", "./fixtures", "The directory where base gifs are stored")
	serveCmd.Flags().StringVarP(&memeDir, "meme-dir", "m", "./memes", "The directory where memes are stored")
	serveCmd.Flags().IntVarP(&port, "port", "p", 3000, "The port to listen on")
	serveCmd.Flags().StringVar(&tplPath, "templates", "./templates", "Path to the teplate directory")
	serveCmd.Flags().StringVar(&certPath, "certpath", "", "Set this to your letsencrypt directory if you want TLS to work")
}

package api

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
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"image/gif"
	"image/jpeg"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lavagetto/memeoid/img"
	"github.com/go-ozzo/ozzo-validation/v4"
)

// MemeHandler is the base structure that
// handles most web operations
type MemeHandler struct {
	// ImgPath is the filesystem path where all images are located
	ImgPath string
	// OutputPath is the path on the filesystem where all memes will be saved
	OutputPath string
	// FontName is the font to use
	FontName string
	// MemeURL is the url at which the file will be served
	MemeURL   string
	templates *template.Template
}

type generateRequest struct {
	From string	`json:"from"`
}

func (r generateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.From, validation.Required),
	)
}

func newGenerateRequest(r *http.Request) (*generateRequest, error) {
	req := generateRequest {
		From: r.URL.Query().Get("from"),
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return &req, nil
}

type memeRequest struct {
	From 	string	`json:"from"`
	Top 	string	`json:"top"`
	Bottom	string	`json:"bottom"`
}

func (r memeRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.From, validation.Required),
		validation.Field(&r.Top, validation.When(r.Bottom == "", validation.Required.Error("Either Top or Bottom is required."))),
		validation.Field(&r.Bottom, validation.When(r.Top == "", validation.Required.Error("Either Top or Bottom is required."))),
	)
}

func newMemeRequest(r *http.Request) (*memeRequest, error) {
	req := memeRequest {
		From: r.URL.Query().Get("from"),
		Top: r.URL.Query().Get("top"),
		Bottom: r.URL.Query().Get("bottom"),
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return &req, nil
}

// LoadTemplates pre-parses the templates.
// Must be called before starting the server.
func (h *MemeHandler) LoadTemplates(basepath string) {
	if h.templates == nil {
		h.templates = template.Must(template.ParseFiles(
			basepath+"/banner.html.gotmpl",
			basepath+"/generate.html.gotmpl",
		))
	}
}

// allGifs returns a list of all gifs
func (h *MemeHandler) allGifs() (*[]string, error) {
	var gifs []string
	files, err := ioutil.ReadDir(h.ImgPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		name := file.Name()
		if filepath.Ext(name) == ".gif" {
			gifs = append(gifs, name)
		}
	}
	return &gifs, err
}

func (h *MemeHandler) jsonBanner(gifs *[]string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	js, err := json.Marshal(gifs)
	if err != nil {
		http.Error(w, `{"error": "bad json encoding"}`, http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func (h *MemeHandler) htmlBanner(gifs *[]string, w http.ResponseWriter) {
	err := h.templates.ExecuteTemplate(w, "banner.html.gotmpl", gifs)
	if err != nil {
		// Yes, this is a reference to the EasyTimeLine MediaWiki extension.
		http.Error(w, "Bad data: maybe ploticus is not installed?", http.StatusInternalServerError)
	}
}

func (h *MemeHandler) getImageFromRequest(w http.ResponseWriter, r *http.Request) string {
	// vars := mux.Vars(r)
	// imageName, ok := vars["from"]
	// if !ok {
	// 	http.Error(w, "missing 'from' parameter", http.StatusBadRequest)
	// 	return ""
	// }
	imageName := r.FormValue("from")
	if imageName == "" {
		http.Error(w, "missing 'from' parameter", http.StatusBadRequest)
		return ""
	}
	// Check that the gif actually exists
	imgFullPath := path.Join(h.ImgPath, imageName)
	if _, err := os.Stat(imgFullPath); os.IsNotExist(err) {
		http.Error(w, "image not found", http.StatusNotFound)
		return ""
	}
	return imageName
}

func (h *MemeHandler) imageExists(imageName string) (string, bool) {
	imgFullPath := path.Join(h.ImgPath, imageName)
	if _, err := os.Stat(imgFullPath); os.IsNotExist(err) {
		return "", false
	}
	return imgFullPath, true
}

// Form returns a form that will generate the meme
func (h *MemeHandler) Form(w http.ResponseWriter, r *http.Request) {
	req, err := newGenerateRequest(r)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}
	if _, exists := h.imageExists(req.From); !exists {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	err = h.templates.ExecuteTemplate(w, "generate.html.gotmpl", req.From)
	if err != nil {
		// Yes, this is a reference to... sigh.
		http.Error(w, "General error: is restbase calling itself?", http.StatusInternalServerError)
	}
}

// ListGifs lists the available GIFs
func (h *MemeHandler) ListGifs(w http.ResponseWriter, r *http.Request) {
	gifs, err := h.allGifs()
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	// If the request is for json data, return it
	if acceptHeaders, ok := r.Header["Accept"]; ok {
		for _, hdr := range acceptHeaders {
			if strings.Contains(hdr, "/json") {
				h.jsonBanner(gifs, w)
				return
			}
		}
	}
	h.htmlBanner(gifs, w)
}

// UID returns the unique ID of the requested gif. This is determined
// by a combination of the image name and the text (top and bottom)
func (h *MemeHandler) UID(r *http.Request) (string, error) {
	// Get a sorted version of the request parameters
	uid := []byte(r.URL.Query().Encode())
	// No need to use anything fancier than sha1
	hasher := sha1.New()
	_, err := hasher.Write(uid)
	if err != nil {
		return "", err
	}
	bs := hasher.Sum(nil)
	return fmt.Sprintf("%x", bs), nil
}

func (h *MemeHandler) saveImage(g *gif.GIF, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return gif.EncodeAll(out, g)
}

// MemeFromRequest generates a meme image from a request, and saves it to disk. Then sends a
// 301 to the user.
func (h *MemeHandler) MemeFromRequest(w http.ResponseWriter, r *http.Request) {
	req, err := newMemeRequest(r)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}
	srcImgPath, srcImgExists := h.imageExists(req.From)
	if !srcImgExists {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	uid, uerr := h.UID(r)
	if uerr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Now check if the file at $outputpath/$uid.gif exists. If it does,
	// just redirect. Else generate the file and redirect
	dstGifPath, gifExists := h.memeExists(uid)
	if !gifExists {
		err := h.generateMeme(srcImgPath, req, dstGifPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	redirURL := fmt.Sprintf("/%s/%s.gif", h.MemeURL, uid)
	http.Redirect(w, r, redirURL, http.StatusPermanentRedirect)
}

func (h *MemeHandler) memeExists(uid string) (string, bool) {
	fullPath := path.Join(h.OutputPath, fmt.Sprintf("%s.gif", uid))
	_, err := os.Stat(fullPath)
	return fullPath, !os.IsNotExist(err)
}

func (h *MemeHandler) generateMeme(srcImagePath string, req *memeRequest, dstImgPath string) error {
	meme, err := img.MemeFromFile(
		srcImagePath,
		req.Top,
		req.Bottom,
		h.FontName,
	)
	if err != nil {
		return err
	}
	err = meme.Generate()
	if err != nil {
		return err
	}
	err = h.saveImage(meme.Gif, dstImgPath)
	if err != nil {
		return err
	}
	return nil
}

// Preview returns a thumbnail, in jpeg format
func (h *MemeHandler) Preview(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("In preview!")
	vars := mux.Vars(r)
	imageName := h.getImageFromRequest(w, r)
	if imageName == "" {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	width, err := strconv.ParseUint(vars["width"], 10, 0)
	if err != nil {
		http.Error(w, "width must be specified", http.StatusBadRequest)
	}
	height, err := strconv.ParseUint(vars["height"], 10, 0)
	if err != nil {
		http.Error(w, "height must be specified", http.StatusBadRequest)
	}
	imgFullPath := path.Join(h.ImgPath, imageName)
	tpl, err := img.SimpleTemplate(imgFullPath, h.FontName, 52.0, 8.0)
	if err != nil {
		http.Error(w, "error generating the thumbnail", http.StatusInternalServerError)
		return
	}
	g, err := tpl.GetGif()
	if err != nil {
		http.Error(w, "error generating the thumbnail", http.StatusInternalServerError)
		return
	}
	m := img.Meme{Gif: g}
	thumb := m.Preview(uint(width), uint(height))
	w.Header().Set("Content-Type", "image/jpeg")
	jpeg.Encode(w, thumb, nil)

}

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
	"errors"
	"fmt"
	"image"
	"image/gif"

    "github.com/lavagetto/memeoid/img"
)


type ImageGateway interface {
	FindImage(string) (string, error)
    FindMeme(string) (string, error)
	ListAllGifs() ([]string, error)
    Save(*gif.GIF, string) error
}

type MemeService struct {
    fontName string
    gateway ImageGateway
}

func NewMemeService(fontName string, gateway ImageGateway) *MemeService {
    return &MemeService{
        fontName: fontName,
        gateway: gateway,
    }
}

func (m *MemeService) ListAllGifs() (*[]string, error) {
    l, err := m.gateway.ListAllGifs()
	return &l, err
}

func (m *MemeService) ImageExists(imgName string) (string, bool) {
    img, err := m.gateway.FindImage(imgName)
    exists := true
    if err != nil {
        exists = false
    }
    return img, exists
}

func (m *MemeService) MemeExists(memeUID string) (string, bool) {
    img, err := m.gateway.FindMeme(memeUID)
    exists := true
    if err != nil {
        exists = false
    }
    return img, exists
}

func (m *MemeService) createUID(queryString string) (string, error) {
	hasher := sha1.New()
	_, err := hasher.Write([]byte(queryString))
	if err != nil {
		return "", err
	}
	bs := hasher.Sum(nil)
    return fmt.Sprintf("%x", bs), nil
}

func (m *MemeService) Save(content *gif.GIF, imgFullPath string) error {
	return m.gateway.Save(content, imgFullPath)
}

func (m *MemeService) GeneratePreview(request *previewRequest) (image.Image, error) {
    srcImgPath, srcImgExists := m.ImageExists(request.From)
	if !srcImgExists {
		// http.Error(w, "Image not found", http.StatusNotFound)
		return nil, errors.New("image not found")
	}
	tpl, err := img.SimpleTemplate(srcImgPath, m.fontName, 52.0, 8.0)
	if err != nil {
		// http.Error(w, "error generating the thumbnail", http.StatusInternalServerError)
		return nil, errors.New("error generating the thumbnail")
	}
	g, err := tpl.GetGif()
	if err != nil {
		// http.Error(w, "error generating the thumbnail", http.StatusInternalServerError)
		return nil, errors.New("error generating the thumbnail")
	}
    imgMeme := img.Meme{Gif: g}
	thumb := imgMeme.Preview(request.toUint(request.Width), request.toUint(request.Height))
	return thumb, nil
}

func (m *MemeService) GenerateMeme(request *memeRequest, encodedUrl string) (string, error) {
    srcImgPath, srcImgExists := m.ImageExists(request.From)
	if !srcImgExists {
		// http.Error(w, "Image not found", http.StatusNotFound)
		return "", errors.New("image not found")
	}
	uid, uerr := m.createUID(encodedUrl)
	if uerr != nil {
		// http.Error(w, "internal error", http.StatusInternalServerError)
		return "", uerr
	}
	// Now check if the file at $outputpath/$uid.gif exists. If it does,
	// just redirect. Else generate the file and redirect
	memeGifName := fmt.Sprintf("%s.gif", uid)
	dstGifPath, gifExists := m.MemeExists(memeGifName)
	if !gifExists {
		err := m.generateMeme(srcImgPath, request, dstGifPath)
		if err != nil {
			// http.Error(w, err.Error(), http.StatusInternalServerError)
			return "", err
		}
	}
    return "", nil
}

func (m *MemeService) generateMeme(srcImagePath string, req *memeRequest, dstImgPath string) error {
	meme, err := img.MemeFromFile(
		srcImagePath,
		req.Top,
		req.Bottom,
		m.fontName,
	)
	if err != nil {
		return err
	}
	err = meme.Generate()
	if err != nil {
		return err
	}
	err = m.Save(meme.Gif, dstImgPath)
	if err != nil {
		return err
	}
	return nil
}

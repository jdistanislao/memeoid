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
	"fmt"
	"image/gif"
)


type ImageGateway interface {
	FindImage(string) (string, error)
    FindMeme(string) (string, error)
	ListAllGifs() ([]string, error)
    Save(*gif.GIF, string) error
}

type Meme struct {
    gateway ImageGateway
}

func NewMeme(gateway ImageGateway) *Meme {
    return &Meme{gateway: gateway}
}

func (m *Meme) ListAllGifs() (*[]string, error) {
    l, err := m.gateway.ListAllGifs()
	return &l, err
}

func (m *Meme) ImageExists(imgName string) (string, bool) {
    img, err := m.gateway.FindImage(imgName)
    exists := true
    if err != nil {
        exists = false
    }
    return img, exists
}

func (m *Meme) MemeExists(memeUID string) (string, bool) {
    img, err := m.gateway.FindMeme(memeUID)
    exists := true
    if err != nil {
        exists = false
    }
    return img, exists
}

func (m *Meme) CreateUID(queryString string) (string, error) {
	hasher := sha1.New()
	_, err := hasher.Write([]byte(queryString))
	if err != nil {
		return "", err
	}
	bs := hasher.Sum(nil)
    return fmt.Sprintf("%x", bs), nil
}

func (m *Meme) Save(content *gif.GIF, imgFullPath string) error {
	return m.gateway.Save(content, imgFullPath)
}

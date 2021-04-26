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

// import (
// 	"bytes"
// 	"crypto/sha1"
// 	"encoding/json"
// 	"fmt"
// 	"html/template"
// 	"image/gif"
// 	"image/jpeg"
// 	"io/ioutil"
// 	"net/http"
// 	"os"
// 	"path"
// 	"path/filepath"
// 	"strconv"
// 	"strings"
// )


type ImageGateway interface {
	ImageExists(string) (string, error)
	ListAllGifs() ([]string, error)
}

type Meme struct {
    gateway ImageGateway
}

func NewMeme(gateway ImageGateway) *Meme {
    return &Meme{gateway: gateway}
}

func (m *Meme)ListAllGifs() (*[]string, error) {
    l, err := m.gateway.ListAllGifs()
	return &l, err
}

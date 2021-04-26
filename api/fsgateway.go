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
	"strings"
	"os"
	"path"
	"io/ioutil"
	"path/filepath"
)


type FsImageGateway struct {
	srcPath string
}

func NewFsImageGateway(srcPath string) ImageGateway {
	return &FsImageGateway{srcPath: srcPath}
}

func (g *FsImageGateway) ImageExists(imageName string) (string, error) {
	imgFullPath := path.Join(g.srcPath, imageName)
	if _, err := os.Stat(imgFullPath); os.IsNotExist(err) {
		return "", err
	}
	return imgFullPath, nil
}

func (g *FsImageGateway) ListAllGifs() ([]string, error) {
	var gifs []string
	files, err := ioutil.ReadDir(g.srcPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		name := file.Name()
		if strings.ToUpper(filepath.Ext(name)) == ".GIF" {
			gifs = append(gifs, name)
		}
	}
	return gifs, err
}


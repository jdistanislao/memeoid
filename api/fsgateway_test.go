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
	"image"
	"image/color"
	"image/gif"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)


type FsGatewayTestSuite struct {
	suite.Suite
	TempSrcDir string
	TempDstDir string
	Sut        ImageGateway
}

func (s *FsGatewayTestSuite) SetupSuite() {
	tempSrcDir, err := ioutil.TempDir("", "memeoid-fs-gateway-src")
	if err != nil {
		panic(err)
	}
	tempDstDir, err := ioutil.TempDir("", "memeoid-fs-gateway-dst")
	if err != nil {
		panic(err)
	}
	s.TempSrcDir = tempSrcDir
	s.TempDstDir = tempDstDir
}

func (s *FsGatewayTestSuite) TearDownSuite() {
	os.RemoveAll(s.TempSrcDir)
}

func (s *FsGatewayTestSuite) SetupTest() {
	s.Sut = NewFsImageGateway(s.TempSrcDir, s.TempDstDir)
}

func (s *FsGatewayTestSuite) TearDownTest() {
	deleteAllFiles(s.TempSrcDir)
	deleteAllFiles(s.TempDstDir)
	s.Sut = nil
}

func deleteAllFiles(dir string) {
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		os.Remove(path.Join(dir, f.Name()))
	}
}

func (s *FsGatewayTestSuite) createSrcFiles(filenames ...string) {
	for _, f := range filenames {
		ioutil.WriteFile(path.Join(s.TempSrcDir, f), make([]byte, 1), fs.ModeTemporary)
	}
}

func (s *FsGatewayTestSuite) createMeme(filenames ...string) {
	for _, f := range filenames {
		ioutil.WriteFile(path.Join(s.TempDstDir, f), make([]byte, 1), fs.ModePerm)
	}
}

func isInList(file string, list []string) bool {
	for _, v := range list {
		if v == file {
			return true
		}
	}
	return false
}

func createInMemoryGif() *gif.GIF {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{1, 1}
	img := image.NewPaletted(image.Rectangle{upLeft, lowRight}, color.Palette{color.Black})

	gif := &gif.GIF{}
	gif.Image = append(gif.Image, img)
	gif.Delay = append(gif.Delay, 1)

	return gif
}

func (s *FsGatewayTestSuite) TestInvalidSrcPath() {
	s.Sut = NewFsImageGateway("whatever", "")

	list, err := s.Sut.ListAllGifs()

	s.NotNil(err)
	s.Nil(list, "Should not return a list: %v", list)
}

func (s *FsGatewayTestSuite) TestNoImagesAreReturnedIfSrcPathIsEmpty() {
	list, err := s.Sut.ListAllGifs()
	s.Nil(err, "Should not return an error: %v", err)
	s.Equal(0, len(list), "List should be empty")
}

func (s *FsGatewayTestSuite) TestOnlyGifImagesAreListed() {
	s.createSrcFiles("a.gif", "b.GIF", "c.jpg")

	list, err := s.Sut.ListAllGifs()

	s.Nil(err, "Should not return an error: %v", err)
	s.Equal(2, len(list), "List should not be empty")
	s.True(isInList("a.gif", list))
	s.True(isInList("b.GIF", list))
}

func (s *FsGatewayTestSuite) TestImageNotFound() {
	s.createSrcFiles("b.GIF")

	fullPath, err := s.Sut.FindImage("a.gif")

	s.NotNil(err)
	s.Equal("", fullPath)
}

func (s *FsGatewayTestSuite) TestFindImage() {
	s.createSrcFiles("a.gif", "b.gif")

	fullPath, err := s.Sut.FindImage("a.gif")
	// bFullPath, berr := s.Sut.FindImage("b.GIF") // TODO: manage case insensitive search for extension?

	s.Nil(err, "Should not return an error: %v", err)
	s.True(strings.Contains(fullPath, "a.gif"))

	// s.Nil(berr, "Should not return an error: %v", berr)
	// s.True(strings.Contains(bFullPath, "b.GIF"))
}

func (s *FsGatewayTestSuite) TestMemeNotFound() {
	s.createMeme("b.GIF")

	fullPath, err := s.Sut.FindMeme("a.gif")

	s.NotNil(err)
	s.Equal("", fullPath)
}

func (s *FsGatewayTestSuite) TestFindMeme() {
	s.createMeme("a.gif", "b.gif")

	fullPath, err := s.Sut.FindMeme("a.gif")
	// bFullPath, berr := s.Sut.FindMeme("b.GIF") // TODO: see TestFindImage()

	s.Nil(err, "Should not return an error: %v", err)
	s.True(strings.Contains(fullPath, "a.gif"))

	// s.Nil(berr, "Should not return an error: %v", berr)
	// s.True(strings.Contains(bFullPath, "b.GIF"))
}

func (s *FsGatewayTestSuite) TestSave() {
	dstPath := path.Join(s.TempDstDir, "tempGif.gif")
	content := createInMemoryGif()

	err := s.Sut.Save(content, dstPath)

	s.Nil(err, "Should not return an error: %v", err)
}

func (s *FsGatewayTestSuite) TestSaveError() {
	dstPath := path.Join("dumb", "tempGif.gif")
	content := createInMemoryGif()

	err := s.Sut.Save(content, dstPath)

	s.NotNil(err)
}

func TestFsGatewayTestSuite(t *testing.T) {
	suite.Run(t, new(FsGatewayTestSuite))
}

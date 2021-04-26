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
	"io/fs"
	"io/ioutil"
	"os"
	"testing"
	"path"
	"strings"

	"github.com/stretchr/testify/suite"
)

// const fsSrcPath string = "../img/fixtures/"

type FsGatewayTestSuite struct {
	suite.Suite
	TempDir string
	Sut     ImageGateway
}

func (s *FsGatewayTestSuite) SetupSuite() {
	tempdir, err := ioutil.TempDir("", "memeoid-fs-gateway")
	if err != nil {
		panic(err)
	}
	s.TempDir = tempdir
}

func (s *FsGatewayTestSuite) TeardownSuite() {
	os.RemoveAll(s.TempDir)
}

func (s *FsGatewayTestSuite) SetupTest() {
	s.Sut = NewFsImageGateway(s.TempDir)
}

func (s *FsGatewayTestSuite) TeardownTest() {
	files, _ := ioutil.ReadDir(s.TempDir)
	for _, f := range files {
		os.Remove(path.Join(s.TempDir, f.Name()))
	}
	s.Sut = nil
}

func (s *FsGatewayTestSuite) createFiles(filenames ...string) {
	for _, f := range filenames {
		ioutil.WriteFile(path.Join(s.TempDir, f), make([]byte, 1), fs.ModeTemporary)
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

func (s *FsGatewayTestSuite) TestInvalidSrcPath() {
	s.Sut = NewFsImageGateway("whatever")

	list, err := s.Sut.ListAllGifs()

	s.NotNil(err)
	s.True(strings.Contains(err.Error(), "open whatever: The system cannot find"))
	s.Nil(list, "Should not return a list: %v", list)
}

func (s *FsGatewayTestSuite) TestNoImagesAreReturnedIfSrcPathIsEmpty() {
	list, err := s.Sut.ListAllGifs()
	s.Nil(err, "Should not return an error: %v", err)
	s.Equal(0, len(list), "List should be empty")
}

func (s *FsGatewayTestSuite) TestOnlyGifImagesAreListed() {
	s.createFiles("a.gif", "b.GIF", "c.jpg")

	list, err := s.Sut.ListAllGifs()

	s.Nil(err, "Should not return an error: %v", err)
	s.Equal(2, len(list), "List should not be empty")
	s.True(isInList("a.gif", list))
	s.True(isInList("b.GIF", list))
}

func (s *FsGatewayTestSuite) TestImageDoesNotExists() {
	s.createFiles("b.GIF")

	fullPath, err := s.Sut.ImageExists("a.gif")

	s.NotNil(err)
	s.True(strings.Contains(err.Error(), "a.gif: The system cannot find the file specified."))
	s.Equal("", fullPath)
}

func (s *FsGatewayTestSuite) TestImageExists() {
	s.createFiles("a.gif", "b.gif")

	aFullPath, aerr := s.Sut.ImageExists("a.gif")
	bFullPath, berr := s.Sut.ImageExists("b.GIF") // just a different case

	s.Nil(aerr, "Should not return an error: %v", aerr)
	s.True(strings.Contains(aFullPath, "a.gif"))

	s.Nil(berr, "Should not return an error: %v", berr)
	s.True(strings.Contains(bFullPath, "b.GIF"))
}



func TestFsGatewayTestSuite(t *testing.T) {
	suite.Run(t, new(FsGatewayTestSuite))
}

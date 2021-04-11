package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

const baseMemeUrl 		string = "url"
const baseImgPath 		string = "../img/fixtures/"
const baseTemplatesPath	string = "../templates"
const fontName 			string = "DejaVuSans"

type MemeGenTestSuite struct {
	suite.Suite
	TempDir string
	Sut     *MemeHandler
}

func (s *MemeGenTestSuite) SetupSuite() {
	tempdir, err := ioutil.TempDir("", "memeoid-api")
	if err != nil {
		panic(err)
	}
	s.TempDir = tempdir
}

func (s *MemeGenTestSuite) TeardownSuite() {
	os.RemoveAll(s.TempDir)
}

func (s *MemeGenTestSuite) SetupTest() {
	s.Sut = &MemeHandler{
		OutputPath: s.TempDir,
		ImgPath:    baseImgPath,
		FontName:   fontName,
		MemeURL:    baseMemeUrl,
	}
	s.Sut.LoadTemplates(baseTemplatesPath)
}

func (s *MemeGenTestSuite) TeardownTest() {
	s.Sut = nil
}

func (s *MemeGenTestSuite) TestUID() {
	// Two requests with the same parameters create the same UID
	reader := strings.NewReader("")
	r := httptest.NewRequest(http.MethodGet, "http://localhost/w/api.php?first=a&last=b", reader)
	r1 := httptest.NewRequest(http.MethodGet, "http://localhost/w/api.php?last=b&first=a", reader)

	uid, err := s.Sut.UID(r)
	s.Nil(err, "could not calculate the UID: %v", err)

	uid1, err := s.Sut.UID(r1)
	s.Nil(err, "could not calculate the UID: %v", err)
	s.Equal(uid, uid1, "expected the UIDs to be equal for the same query parameters")

	// But this is case-sensitive.
	r1 = httptest.NewRequest(http.MethodGet, "http://localhost/w/api.php?last=b&First=a", reader)

	uid1, err = s.Sut.UID(r1)
	s.Nil(err, "could not calculate the UID: %v", err)
	s.NotEqual(uid, uid1, "expected the UIDs to be different for different capitalizations")
}

func (s *MemeGenTestSuite) TestListGifs() {
	var testCases = []struct {
		Path        string
		ContentType string
		Status      string
		Body        string
	}{
		{"", "application/json", "200 OK", `["badfile.gif","earth.gif","gagarin.gif"]`},
		{"/nonexistent", "", "404 Not Found", ""},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("Path: %s - ContentType: %s - Status: %s - Body: %s", tc.Path, tc.ContentType, tc.Status, tc.Body)
		s.Run(testName, func() {
			s.Sut.ImgPath = baseImgPath
			if tc.Path != "" {
				s.Sut.ImgPath = tc.Path
			}
			req := httptest.NewRequest(http.MethodGet, "http://localhost/gifs", strings.NewReader(""))
			req.Header.Set("Accept", "text/json")
			rec := httptest.NewRecorder()

			s.Sut.ListGifs(rec, req)

			response := rec.Result()
			s.Equal(tc.Status, response.Status)
			if tc.ContentType != "" {
				s.Equal([]string{tc.ContentType}, response.Header["Content-Type"])
			}
			if tc.Body != "" {
				defer response.Body.Close()
				body, err := ioutil.ReadAll(response.Body)
				s.Nil(err)
				s.Equal(tc.Body, string(body))
			}
		})
	}
}

func (s *MemeGenTestSuite) TestMemeGenerate() {
	var testCases = []struct {
		Uri           string
		StatusCode    int
		FileGenerated bool
	}{
		{"http://localhost/w/api.php", http.StatusBadRequest, false},
		{"http://localhost/w/api.php?from=lala", http.StatusNotFound, false},
		{"http://localhost/w/api.php?from=earth.gif", http.StatusBadRequest, false},
		// earth.gif is a large, animated gif. We run a single render of it.
		{"http://localhost/w/api.php?from=earth.gif&top=test", http.StatusPermanentRedirect, true},
		{"http://localhost/w/api.php?from=gagarin.gif&bottom=test", http.StatusPermanentRedirect, true},
		{"http://localhost/w/api.php?from=gagarin.gif&bottom=test&top=test", http.StatusPermanentRedirect, true},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("Uri: %s - StatusCode: %d - Genereate: %t", tc.Uri, tc.StatusCode, tc.FileGenerated)
		s.Run(testName, func() {
			req := httptest.NewRequest(http.MethodGet, tc.Uri, strings.NewReader(""))
			rec := httptest.NewRecorder()

			s.Sut.MemeFromRequest(rec, req)

			response := rec.Result()
			s.Equal(tc.StatusCode, response.StatusCode)
			if tc.FileGenerated {
				locationPrefix := fmt.Sprintf("/%s/", baseMemeUrl)
				locationHeader, ok := response.Header["Location"]
				s.True(ok, "response should include a Location header")
				s.NotEmpty(locationHeader, "response should include a Location header")

				// Extract location on disk from the Location Header
				fileName := locationHeader[0][len(locationPrefix):]
				filePath := path.Join(s.TempDir, fileName)
				s.FileExists(filePath)
			}
		})
	}
}

func (s *MemeGenTestSuite) TestMemeForm() {
	var testCases = []struct {
		Uri           string
		StatusCode    int
	}{
		{"http://localhost/w/api.php", http.StatusBadRequest},
		{"http://localhost/w/api.php?from=lala", http.StatusNotFound},
		{"http://localhost/w/api.php?from=earth.gif", http.StatusOK},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("Uri: %s - StatusCode: %d", tc.Uri, tc.StatusCode)
		s.Run(testName, func() {
			req := httptest.NewRequest(http.MethodGet, tc.Uri, strings.NewReader(""))
			rec := httptest.NewRecorder()

			s.Sut.Form(rec, req)

			response := rec.Result()
			s.Equal(tc.StatusCode, response.StatusCode)
		})
	}
}

func TestMemeGenTestSuite(t *testing.T) {
	suite.Run(t, new(MemeGenTestSuite))
}

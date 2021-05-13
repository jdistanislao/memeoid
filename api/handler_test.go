package api

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const baseMemeUrl string = "url"
const baseImgPath string = "../img/fixtures/"
const baseTemplatesPath string = "../templates"
const fontName string = "DejaVuSans"

type HandlerTestSuite struct {
	suite.Suite
	TempDir    string
	Sut        *ApiHandler
	ImgGateway *ImageGatewayMock
}

func (s *HandlerTestSuite) SetupSuite() {
	tempdir, err := ioutil.TempDir("", "memeoid-api")
	if err != nil {
		panic(err)
	}
	s.TempDir = tempdir
}

func (s *HandlerTestSuite) TearDownSuite() {
	os.RemoveAll(s.TempDir)
}

func (s *HandlerTestSuite) SetupTest() {
	imgGateway := new(ImageGatewayMock)
	meme := NewMeme(imgGateway)
	s.Sut = &ApiHandler{
		OutputPath: s.TempDir,
		ImgPath:    baseImgPath,
		FontName:   fontName,
		MemeURL:    baseMemeUrl,
		Meme:       meme,
	}
	s.ImgGateway = imgGateway
	s.Sut.LoadTemplates(baseTemplatesPath)
}

func (s *HandlerTestSuite) TearDownTest() {
	s.Sut = nil
	s.ImgGateway = nil
}

func (s *HandlerTestSuite) createSrcGifFile(fileName string) string {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{1024, 1024}
	img := image.NewPaletted(image.Rectangle{upLeft, lowRight}, color.Palette{color.Black})

	content := &gif.GIF{}
	content.Image = append(content.Image, img)
	content.Delay = append(content.Delay, 1)
	
	fullPath := path.Join(s.TempDir, fileName)
	out, _ := os.Create(fullPath)
	defer out.Close()
	gif.EncodeAll(out, content)
	return fullPath
}

func (s *HandlerTestSuite) TestUID() {
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

func (s *HandlerTestSuite) TestListGifs() {
	var testCases = []struct {
		Path        string
		ContentType string
		Status      string
		Gifs        []string
		customError error
	}{
		{"/nonexistent", "", "404 Not Found", nil, errors.New("Some error")},
		{"", "application/json", "200 OK", []string{"a.gif", "b.gif"}, nil},
		{"", "application/json", "200 OK", make([]string, 0), nil},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("Path: %s - ContentType: %s - Status: %s - Body: %s", tc.Path, tc.ContentType, tc.Status, tc.Gifs)
		s.Run(testName, func() {
			s.Sut.ImgPath = baseImgPath
			if tc.Path != "" {
				s.Sut.ImgPath = tc.Path
			}
			req := httptest.NewRequest(http.MethodGet, "http://localhost/gifs", strings.NewReader(""))
			req.Header.Set("Accept", "text/json")
			rec := httptest.NewRecorder()

			s.ImgGateway.On("ListAllGifs").Return(tc.Gifs, tc.customError).Once()

			s.Sut.ListGifs(rec, req)

			response := rec.Result()
			s.Equal(tc.Status, response.Status)
			if tc.ContentType != "" {
				s.Equal([]string{tc.ContentType}, response.Header["Content-Type"])
			}

			s.ImgGateway.AssertExpectations(s.T())
		})
	}
}

func (s *HandlerTestSuite) TestMemeGenerate() {
	var testCases = []struct {
		Uri           string
		StatusCode    int
		FileName      string
		FileGenerated bool
	}{
		{"http://localhost/w/api.php", http.StatusBadRequest, "", false},
		{"http://localhost/w/api.php?from=lala", http.StatusBadRequest, "", false},
		{"http://localhost/w/api.php?from=earth.gif", http.StatusBadRequest, "", false},
		{"http://localhost/w/api.php?from=666.gif&top=test", http.StatusNotFound, "", false},
		{"http://localhost/w/api.php?from=earth.gif&top=test", http.StatusPermanentRedirect, "earth.gif", true},
		{"http://localhost/w/api.php?from=gagarin.gif&bottom=test", http.StatusPermanentRedirect, "gagarin.gif", true},
		{"http://localhost/w/api.php?from=gagarin.gif&bottom=test&top=test", http.StatusPermanentRedirect, "gagarin.gif", true},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("Uri: %s - StatusCode: %d - Genereate: %t", tc.Uri, tc.StatusCode, tc.FileGenerated)
		s.Run(testName, func() {
			req := httptest.NewRequest(http.MethodGet, tc.Uri, strings.NewReader(""))
			rec := httptest.NewRecorder()

			if tc.StatusCode == http.StatusNotFound {
				s.ImgGateway.On("FindImage", mock.AnythingOfType("string")).Return("", errors.New("whatever")).Once()
			}
			if tc.FileGenerated {
				imgPath := s.createSrcGifFile(tc.FileName)
				s.ImgGateway.On("FindImage", mock.AnythingOfType("string")).Return(imgPath, nil).Once()
				s.ImgGateway.On("FindMeme", mock.AnythingOfType("string")).Return("", errors.New("whatever")).Once()
				s.ImgGateway.On("Save", mock.AnythingOfType("*gif.GIF"), mock.AnythingOfType("string")).Return(nil).Once()
			}

			s.Sut.MemeFromRequest(rec, req)

			response := rec.Result()
			s.Equal(tc.StatusCode, response.StatusCode)
			if tc.FileGenerated {
				locationHeader, ok := response.Header["Location"]
				s.True(ok, "response should include a Location header")
				s.NotEmpty(locationHeader, "response should include a Location header")
			}

			s.ImgGateway.AssertExpectations(s.T())
		})
	}
}

func (s *HandlerTestSuite) TestMemeForm() {
	var testCases = []struct {
		Uri			string
		StatusCode	int
		ImgExists	bool
	}{
		{"http://localhost/w/api.php", http.StatusBadRequest, false},
		{"http://localhost/w/api.php?from=lala", http.StatusNotFound, false},
		{"http://localhost/w/api.php?from=earth.gif", http.StatusOK, true},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("Uri: %s - StatusCode: %d", tc.Uri, tc.StatusCode)
		s.Run(testName, func() {
			req := httptest.NewRequest(http.MethodGet, tc.Uri, strings.NewReader(""))
			rec := httptest.NewRecorder()

			if tc.StatusCode != http.StatusBadRequest {
				var err error = nil
				if !tc.ImgExists {
					err = errors.New("whatever")
				}
				s.ImgGateway.On("FindImage", mock.AnythingOfType("string")).Return("", err).Once()
			}

			s.Sut.Form(rec, req)

			response := rec.Result()
			s.Equal(tc.StatusCode, response.StatusCode)

			s.ImgGateway.AssertExpectations(s.T())
		})
	}
}

func (s *HandlerTestSuite) TestPreview() {
	var testCases = []struct {
		Uri        string
		FileName   string
		StatusCode int
	}{
		{"http://localhost/thumb?from=whatever.gif&width=20&height=a", "", http.StatusBadRequest},
		{"http://localhost/thumb?from=whatever.gif&width=a&height=20", "", http.StatusBadRequest},
		{"http://localhost/thumb?width=20&height=20", "", http.StatusBadRequest},
		{"http://localhost/thumb?from=666.gif&width=20&height=20", "", http.StatusNotFound},
		{"http://localhost/thumb?from=gagarin.gif&width=20&height=20", "gagarin.gif", http.StatusOK},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("Uri: %s - StatusCode: %d", tc.Uri, tc.StatusCode)
		s.Run(testName, func() {
			req := httptest.NewRequest(http.MethodGet, tc.Uri, strings.NewReader(""))
			rec := httptest.NewRecorder()

			if tc.StatusCode == http.StatusNotFound {
				s.ImgGateway.On("FindImage", mock.AnythingOfType("string")).Return("", errors.New("whatever")).Once()
			}
			if tc.StatusCode == http.StatusOK {
				imgPath := s.createSrcGifFile(tc.FileName)
				s.ImgGateway.On("FindImage", mock.AnythingOfType("string")).Return(imgPath, nil).Once()
			}

			s.Sut.Preview(rec, req)

			response := rec.Result()
			s.Equal(tc.StatusCode, response.StatusCode)
			if response.StatusCode == http.StatusOK {
				contentTypeHeader, ok := response.Header["Content-Type"]
				s.True(ok, "response should include a Content-Type header")
				s.NotEmpty(contentTypeHeader, "response should include a Content-Type header")
				s.Equal("image/jpeg", contentTypeHeader[0])

				body := make([]byte, response.ContentLength)
				response.Body.Read(body)
				s.Equal("image/jpeg", http.DetectContentType(body))
			}

			s.ImgGateway.AssertExpectations(s.T())
		})
	}
}

func TestMemeGenTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

// ----- Mocks -------

type ImageGatewayMock struct {
	mock.Mock
}

func (m *ImageGatewayMock) FindImage(imageName string) (string, error) {
	args := m.Called(imageName)
	return args.String(0), args.Error(1)
}

func (m *ImageGatewayMock) FindMeme(memeUID string) (string, error) {
	args := m.Called(memeUID)
	return args.String(0), args.Error(1)
}

func (m *ImageGatewayMock) ListAllGifs() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *ImageGatewayMock) Save(content *gif.GIF, imgFullPath string) error {
	args := m.Called(content, imgFullPath)
	return args.Error(0)
}

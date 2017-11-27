package iffy

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type MultipartBodyPart interface {
	WriteField(key string, writer *multipart.Writer) error
}

type MultipartBodyString struct {
	Data string
}

func (mbs MultipartBodyString) WriteField(key string, writer *multipart.Writer) error {
	return writer.WriteField(key, mbs.Data)
}

type MultipartBodyFile struct {
	Filename string
}

func (mbf MultipartBodyFile) WriteField(key string, writer *multipart.Writer) error {
	_, err := os.Stat(mbf.Filename)
	if !os.IsNotExist(err) {
		part, err := writer.CreateFormFile(key, filepath.Base(mbf.Filename))
		if err != nil {
			return err
		}
		file, err := os.Open(mbf.Filename)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(part, file)
		if err != nil {
			return err
		}
	}
	return nil
}

type MultipartBody struct {
	contentType string
	Data        map[string]MultipartBodyPart
}

func NewMultipartBody() *MultipartBody {
	return &MultipartBody{
		Data: map[string]MultipartBodyPart{},
	}
}

func (mp *MultipartBody) Set(key string, value MultipartBodyPart) *MultipartBody {
	mp.Data[key] = value
	return mp
}

func (mp *MultipartBody) ContentType() string {
	return mp.contentType
}

func (mp *MultipartBody) GetBody(tmpl TemplaterFunc) (io.Reader, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer func() {
		_ = writer.Close()
	}()
	for key, value := range mp.Data {
		err := value.WriteField(key, writer)
		if err != nil {
			return nil, err
		}
	}
	mp.contentType = writer.FormDataContentType()
	return body, nil
}

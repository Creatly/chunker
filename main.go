package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
)

const (
	_chunkSize = 3 << 20
	_url       = "http://localhost:8000/api/v1/admins/upload/video"
	// generate and paste access token here.
	_token = ""
	// paste school domain.
	_referer = "http://localhost:8000/"
	// file you want to upload
	_filename = "test.MOV"
)

func main() {
	file, err := os.Open(_filename)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fileSize := stat.Size()
	log.Println("file size: ", fileSize)

	offset := int64(0)
	reader := bufio.NewReader(file)
	buf := make([]byte, _chunkSize)
	finish := false

	for {
		if finish {
			break
		}

		file.Seek(offset, 0)

		_, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				finish = true
			} else {
				log.Fatal(err)
			}
		}

		if err := upload(_url, _token, file.Name(), buf, offset, fileSize); err != nil {
			log.Fatal(err)
		}

		offset += _chunkSize
		if offset > fileSize {
			offset = fileSize
		}

		proccesed := (float64(offset) / float64(stat.Size())) * 100
		log.Printf("Loaded %.2f%% of file content", proccesed)
	}
}

func upload(url, token, filename string, chunkBytes []byte, offset, filesize int64) error {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return err
	}

	if _, err := part.Write(chunkBytes); err != nil {
		return err
	}

	writer.SetBoundary(writer.Boundary())

	if err = writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return err
	}

	rangeLimit := offset + _chunkSize
	if rangeLimit > filesize {
		rangeLimit = filesize
	}

	contentRange := fmt.Sprintf("bytes %d-%d/%d", offset, rangeLimit, filesize)
	req.Header.Set("Content-Range", contentRange)
	req.Header.Set("Referer", _referer)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	log.Println("Content-Range", contentRange)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return errors.New("status not ok")
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Println("resp", string(respBody))

	return nil
}

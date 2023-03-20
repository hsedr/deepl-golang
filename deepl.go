package deepl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anthdm/tasker"
	"github.com/carlmjohnson/requests"
	"github.com/deepl/types"
	"github.com/fatih/structs"
)

const (
	_host       = "https://api-free.deepl.com/v2"
	host        = "http://localhost:3000/v2"
	contentType = "application/x-www-form-urlencoded"
)

type Translator struct {
	ApiKey string
}

func NewTranslator(key string) *Translator {
	return &Translator{
		ApiKey: key,
	}
}

func (d *Translator) TranslateTextAsync(text string, sourceLang string, targetLang string, options *types.TextTranslateOptions) tasker.TaskFunc[types.Translations] {
	return func(ctx context.Context) (types.Translations, error) {
		var response types.Translations
		err := requests.
			URL(host+"/translate").
			Header("Authorization", "DeepL-Auth-Key "+d.ApiKey).
			Header("Connection", "keep-alive").
			UserAgent("Live-Translator/1.0").
			ContentType(contentType).
			Header("Content-Length", strconv.Itoa(utf8.RuneCountInString(text))).
			Param("text", text).
			Param("source_lang", sourceLang).
			Param("target_lang", targetLang).
			Config(func(rb *requests.Builder) {
				for k, v := range structToMap(options) {
					rb.Param(k, v)
				}
			}).
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

// todo: documentation
func (d *Translator) TranslateDocumentAsync(s string, t string, f io.Reader, options types.DocumentTranslateOptions) tasker.TaskFunc[types.DocumentStatus] {
	return func(ctx context.Context) (types.DocumentStatus, error) {
		var status types.DocumentStatus
		doc, err := tasker.Spawn(d.uploadDocumentAsync(s, t, f, options)).Await()
		if err != nil {
			return status, err
		}
		status, err = tasker.Spawn(d.isDocumentTranslationComplete(&doc)).Await()
		if err != nil {
			return status, err
		}
		_, err = tasker.Spawn(d.downloadDocumentAsync(&doc, options.OutputFile)).Await()
		if err != nil {
			return status, err
		}
		return status, nil
	}
}

func (d *Translator) uploadDocumentAsync(s string, t string, file io.Reader, options types.DocumentTranslateOptions) tasker.TaskFunc[types.DocumentIDAndKey] {
	return func(ctx context.Context) (types.DocumentIDAndKey, error) {
		var doc types.DocumentIDAndKey
		body := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(body)
		bodyWriter.WriteField("source_lang", s)
		bodyWriter.WriteField("target_lang", t)
		fileWriter, err := bodyWriter.CreateFormFile("file", options.FileName)
		if err != nil {
			fmt.Println(err)
			return doc, err
		}
		io.Copy(fileWriter, file)
		bodyWriter.Close()
		err = requests.
			URL(host+"/document").
			Header("Authorization", "DeepL-Auth-Key "+d.ApiKey).
			Header("Connection", "keep-alive").
			UserAgent("Live-Translator/1.0").
			ContentType(fmt.Sprintf("multipart/form-data;boundary=%s", bodyWriter.Boundary())).
			BodyBytes(body.Bytes()).
			ToJSON(&doc).
			Fetch(context.Background())
		if err != nil {
			return doc, err
		}
		return doc, nil
	}
}

func (d *Translator) checkDocumentStatusAsync(doc *types.DocumentIDAndKey) tasker.TaskFunc[types.DocumentStatus] {
	return func(ctx context.Context) (types.DocumentStatus, error) {
		path := fmt.Sprintf("/document/%s", doc.DocumentID)
		var res types.DocumentStatus
		err := requests.
			URL(host+path).
			Header("Authorization", "DeepL-Auth-Key "+d.ApiKey).
			Header("Connection", "keep-alive").
			UserAgent("Live-Translator/1.0").
			ContentType(contentType).
			Header("Content-Length", strconv.Itoa(utf8.RuneCountInString(doc.DocumentKey))).
			Param("document_key", doc.DocumentKey).
			ToJSON(&res).
			Fetch(context.Background())
		if err != nil {
			return res, err
		}
		return res, nil
	}
}

// Fullfills when document translation either finnished or ran into an error.
func (d *Translator) isDocumentTranslationComplete(doc *types.DocumentIDAndKey) tasker.TaskFunc[types.DocumentStatus] {
	return func(ctx context.Context) (types.DocumentStatus, error) {
		status, err := tasker.Spawn(d.checkDocumentStatusAsync(doc)).Await()
		if err != nil {
			return status, err
		}
		for !status.Done() && status.Ok() {
			secs := float64(status.SecondsRemaining/2 + 1)
			time.Sleep(time.Duration(secs) * time.Second)
			status, err = tasker.Spawn(d.checkDocumentStatusAsync(doc)).Await()
			if err != nil {
				return status, err
			}
		}
		if !status.Ok() {
			return status, errors.New("docoument translation failed, status not ok")
		}
		return status, err
	}
}

func (d *Translator) downloadDocumentAsync(doc *types.DocumentIDAndKey, file io.Writer) tasker.TaskFunc[bool] {
	return func(ctx context.Context) (bool, error) {
		path := fmt.Sprintf("/document/%s/result", doc.DocumentID)
		err := requests.
			URL(host+path).
			Header("Authorization", "DeepL-Auth-Key "+d.ApiKey).
			Header("Connection", "keep-alive").
			UserAgent("Live-Translator/1.0").
			ContentType(contentType).
			Header("Content-Length", strconv.Itoa(utf8.RuneCountInString(doc.DocumentKey))).
			Param("document_key", doc.DocumentKey).
			ToWriter(file).
			Fetch(context.Background())
		if err != nil {
			return false, err
		}
		return true, nil
	}
}

func (d *Translator) GetUsage() tasker.TaskFunc[types.Usage] {
	return func(ctx context.Context) (types.Usage, error) {
		var response types.Usage
		err := requests.
			URL(host+"/usage").
			Header("Authorization", "DeepL-Auth-Key "+d.ApiKey).
			UserAgent("Live-Translator/1.0").
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

// Retreives supported languages.
//
// languageType, type of language to retrive, "source" or "target"
func (d *Translator) GetLanguagesAsync(languageType string) tasker.TaskFunc[[]types.SupportedLanguage] {
	return func(ctx context.Context) ([]types.SupportedLanguage, error) {
		var response []types.SupportedLanguage
		err := requests.
			URL(host+"/languages").
			Header("Authorization", "DeepL-Auth-Key "+d.ApiKey).
			UserAgent("Live-Translator/1.0").
			Param("type", languageType).
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

func (d *Translator) GetGlossaryLanguagesAsync() tasker.TaskFunc[types.GlossaryLanguagePairs] {
	return func(ctx context.Context) (types.GlossaryLanguagePairs, error) {
		var response types.GlossaryLanguagePairs
		err := requests.
			URL(host+"/glossary-language-pairs").
			Header("Authorization", "DeepL-Auth-Key "+d.ApiKey).
			UserAgent("Live-Translator/1.0").
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

func IsFreeAccountAuthKey(key string) bool {
	return strings.HasSuffix(key, ":fx")
}

// Returns the struct represented as a map, with the json fields as
// keys.
func structToMap(s interface{}) map[string]string {
	ret := make(map[string]string)
	str := structs.New(s)
	m := str.Map()
	for k, v := range m {
		f := str.Field(k)
		json := f.Tag("json")
		value := fmt.Sprintf("%v", v)
		if json == "" || value == "" {
			continue
		}
		ret[json] = value
	}
	return ret
}

package deepl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/anthdm/tasker"
	"github.com/carlmjohnson/requests"
	"github.com/deepl/types"
	"github.com/fatih/structs"
)

type Translator struct {
	HttpClient *http.Client
}

func NewTranslator(authKey string, options types.TranslatorOptions) (*Translator, error) {
	if authKey == "" {
		return &Translator{}, errors.New("authKey must be a non-empty string")
	}
	retries := 5
	timeout := time.Second * 5
	serverURL := ""
	headers := make(map[string]string)
	if options.ServerURL != "" {
		serverURL = options.ServerURL
	} else if IsFreeAccountAuthKey(authKey) {
		serverURL = "https://api-free.deepl.com"
	} else {
		serverURL = "https://api.deepl.com"
	}
	if options.Retries >= 0 {
		retries = options.Retries
	}
	if options.TimeOut >= 1 {
		timeout = options.TimeOut
	}
	headers["Authorization"] = fmt.Sprint("DeepL-Auth-Key ", authKey)
	headers["User-Agent"] = constructUserAgentString(options.SendPlattformInfo, options.AppInfo)
	return &Translator{
		HttpClient: NewTransport(serverURL, headers, timeout, retries).Client(),
	}, nil
}

func (d *Translator) TranslateTextAsync(text string, sourceLang string, targetLang string, options *types.TextTranslateOptions) tasker.TaskFunc[types.Translations] {
	return func(ctx context.Context) (types.Translations, error) {
		var response types.Translations
		err := requests.
			URL("/translate").
			Client(d.HttpClient).
			ContentType("application/x-www-form-urlencoded").
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

func (d *Translator) uploadDocumentAsync(s string, t string, file io.Reader, options types.DocumentTranslateOptions) tasker.TaskFunc[types.DocumentHandle] {
	return func(ctx context.Context) (types.DocumentHandle, error) {
		var doc types.DocumentHandle
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
			URL("/document").
			Client(d.HttpClient).
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

func (d *Translator) checkDocumentStatusAsync(doc *types.DocumentHandle) tasker.TaskFunc[types.DocumentStatus] {
	return func(ctx context.Context) (types.DocumentStatus, error) {
		path := fmt.Sprintf("/document/%s", doc.DocumentID)
		var res types.DocumentStatus
		err := requests.
			URL(path).
			Client(d.HttpClient).
			ContentType("application/x-www-form-urlencoded").
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
func (d *Translator) isDocumentTranslationComplete(doc *types.DocumentHandle) tasker.TaskFunc[types.DocumentStatus] {
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

func (d *Translator) downloadDocumentAsync(doc *types.DocumentHandle, file io.Writer) tasker.TaskFunc[bool] {
	return func(ctx context.Context) (bool, error) {
		path := fmt.Sprintf("/document/%s/result", doc.DocumentID)
		err := requests.
			URL(path).
			Client(d.HttpClient).
			ContentType("application/x-www-form-urlencoded").
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
			URL("/usage").
			Client(d.HttpClient).
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
			URL("/languages").
			Client(d.HttpClient).
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
			URL("/glossary-language-pairs").
			Client(d.HttpClient).
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

func checkStatusCode() {
	//TODO
}

func constructUserAgentString(sendPlattformInfo bool, appInfo types.AppInfo) string {
	libraryInfo := "deepl-golang/1.0 "
	if sendPlattformInfo {
		system := runtime.GOOS
		goVersion := runtime.Version()
		libraryInfo += system + " " + goVersion
	}
	if appInfo != (types.AppInfo{}) {
		libraryInfo += fmt.Sprint(" "+appInfo.AppName, "/", appInfo.AppVersion)
	}
	return libraryInfo
}

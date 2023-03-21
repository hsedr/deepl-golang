package deepl

import (
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

// TranslateDocumentAsync translates a document and returns a task that can be awaited.
func (d *Translator) TranslateDocumentAsync(s string, t string, f io.Reader, options types.DocumentTranslateOptions) tasker.TaskFunc[types.DocumentStatus] {
	return func(ctx context.Context) (types.DocumentStatus, error) {
		var status types.DocumentStatus
		doc, err := tasker.Spawn(d.uploadDocumentAsync(s, t, f, options)).Await()
		if err != nil {
			return status, err
		}
		status, err = tasker.Spawn(d.isDocumentTranslationCompleteAsync(&doc)).Await()
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

// uploadDocumentAsync uploads a document to the DeepL API and returns a task that can be awaited.
func (d *Translator) uploadDocumentAsync(s string, t string, file io.Reader, options types.DocumentTranslateOptions) tasker.TaskFunc[types.DocumentHandle] {
	return func(ctx context.Context) (types.DocumentHandle, error) {
		var doc types.DocumentHandle
		bodyWriter := &multipart.Writer{}
		var contentType string
		err := requests.
			URL("/document").
			Client(d.HttpClient).
			BodyWriter(func(w io.Writer) error {
				bodyWriter = multipart.NewWriter(w)
				defer bodyWriter.Close()
				bodyWriter.WriteField("source_lang", s)
				bodyWriter.WriteField("target_lang", t)
				fileWriter, err := bodyWriter.CreateFormFile("file", options.FileName)
				if err != nil {
					fmt.Println(err)
					return err
				}
				io.Copy(fileWriter, file)
				contentType = bodyWriter.FormDataContentType()
				return nil
			}).
			ContentType(contentType).
			ToJSON(&doc).
			Fetch(context.Background())
		if err != nil {
			return doc, err
		}
		return doc, nil
	}
}

// checkDocumentStatusAsync checks the status of a document translation and returns a task that can be awaited.
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

// isDocumentTranslationCompleteAsync checks if a document translation is complete and returns a task that can be awaited.
// If the translation is not complete, the task will wait for half the estimated time remaining and check again.
func (d *Translator) isDocumentTranslationCompleteAsync(doc *types.DocumentHandle) tasker.TaskFunc[types.DocumentStatus] {
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

// downloadDocumentAsync downloads a document translation and returns a task that can be awaited.
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

// GetUsageAsync returns the current usage of the DeepL API.
func (d *Translator) GetUsageAsync() tasker.TaskFunc[types.Usage] {
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

// GetLanguagesAsync returns the supported languages of the DeepL API.
// The languageType parameter can be either "source" or "target".
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

// GetGlossaryLanguagesAsync returns the supported languages of the DeepL API.
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

// IsFreeAccountAuthKey returns true if the given auth key is a free account auth key.
func IsFreeAccountAuthKey(key string) bool {
	return strings.HasSuffix(key, ":fx")
}

// structToMap converts a struct to a map[string]string.
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

// constructUserAgentString constructs the user agent string that is sent with each request.
func constructUserAgentString(sendPlattformInfo bool, appInfo types.AppInfo) string {
	libraryInfo := "deepl-golang/1.0 "
	if sendPlattformInfo {
		system := runtime.GOOS
		goVersion := runtime.Version()
		libraryInfo += fmt.Sprintf("%s %s", system, goVersion)
	}
	if appInfo != (types.AppInfo{}) {
		libraryInfo += fmt.Sprintf(" %s/%s", appInfo.AppName, appInfo.AppVersion)
	}
	return libraryInfo
}

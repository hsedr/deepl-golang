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
	"github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/hsedr/deepl-golang/consts"
	"github.com/hsedr/deepl-golang/types"
)

type Translator struct {
	HttpClient *http.Client
}

func NewTranslator(authKey string, opts ...func(*types.TranslatorOptions) error) (*Translator, error) {
	options := types.TranslatorOptions{Headers: map[string]string{}}
	if authKey == "" {
		return &Translator{}, errors.New("authKey must be a non-empty string")
	}
	options.Headers["Authorization"] = fmt.Sprint("DeepL-Auth-Key ", authKey)
	for _, opt := range opts {
		opt(&options)
	}
	if options.ServerURL == "" {
		if IsFreeAccountAuthKey(authKey) {
			options.ServerURL = "https://api-free.deepl.com/v2"
		} else {
			options.ServerURL = "https://api.deepl.com/v2"
		}
	}
	return &Translator{
		HttpClient: NewTransport(options.ServerURL, options.Headers, options.TimeOut, options.Retries).Client(),
	}, nil
}

func WithServerURL(serverURL string) func(*types.TranslatorOptions) error {
	return func(options *types.TranslatorOptions) error {
		options.ServerURL = serverURL
		return nil
	}
}

func WithUserAgent(sendPlattformInfo bool, appInfo types.AppInfo) func(*types.TranslatorOptions) error {
	return func(options *types.TranslatorOptions) error {
		options.Headers["User-Agent"] = constructUserAgentString(sendPlattformInfo, appInfo)
		return nil
	}
}

func WithRetries(retries int) func(*types.TranslatorOptions) error {
	return func(options *types.TranslatorOptions) error {
		options.Retries = retries
		return nil
	}
}

func WithTimeOut(timeout time.Duration) func(*types.TranslatorOptions) error {
	return func(options *types.TranslatorOptions) error {
		options.TimeOut = timeout
		return nil
	}
}

func WithHeaders(headers map[string]string) func(*types.TranslatorOptions) error {
	return func(options *types.TranslatorOptions) error {
		for k, v := range headers {
			options.Headers[k] = v
		}
		return nil
	}
}

func (d *Translator) TranslateTextAsync(
	text []string,
	sourceLang consts.SourceLang,
	targetLang consts.TargetLang,
	opts ...func(*types.TextTranslateOptions) error,
) tasker.TaskFunc[[]types.Translation] {
	return func(ctx context.Context) ([]types.Translation, error) {
		var response types.Translations
		options := types.TextTranslateOptions{}
		for _, opt := range opts {
			opt(&options)
			break
		}
		err := requests.
			URL("/translate").
			Client(d.HttpClient).
			ContentType("application/x-www-form-urlencoded").
			Param("text", text...).
			Param("source_lang", string(sourceLang)).
			Param("target_lang", string(targetLang)).
			Config(func(rb *requests.Builder) {
				for k, v := range structToMap(options) {
					rb.Param(k, v)
				}
			}).
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response.Translations, err
		}
		return response.Translations, nil
	}
}

func WithTextTranslateOptions(options types.TextTranslateOptions) func(*types.TextTranslateOptions) error {
	return func(opts *types.TextTranslateOptions) error {
		opts = &options
		return nil
	}
}

// TranslateDocumentAsync translates a document and returns a task that can be awaited.
func (d *Translator) TranslateDocumentAsync(
	s consts.SourceLang,
	t consts.TargetLang,
	f io.Reader,
	w io.Writer,
	opts ...func(*types.DocumentTranslateOptions) error,
) tasker.TaskFunc[types.DocumentStatus] {
	return func(ctx context.Context) (types.DocumentStatus, error) {
		var status types.DocumentStatus
		options := types.DocumentTranslateOptions{}
		for _, opt := range opts {
			opt(&options)
			break
		}
		if options.FileName == "" {
			options.FileName = uuid.New().String()
		}
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

func WithDocumentTranslateOptions(options types.DocumentTranslateOptions) func(*types.DocumentTranslateOptions) error {
	return func(opts *types.DocumentTranslateOptions) error {
		opts = &options
		return nil
	}
}

// uploadDocumentAsync uploads a document to the DeepL API and returns a task that can be awaited.
func (d *Translator) uploadDocumentAsync(
	s consts.SourceLang,
	t consts.TargetLang,
	file io.Reader,
	options types.DocumentTranslateOptions,
) tasker.TaskFunc[types.DocumentHandle] {
	return func(ctx context.Context) (types.DocumentHandle, error) {
		var doc types.DocumentHandle
		boundary := strings.Replace(uuid.New().String(), "-", "", -1)
		contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
		err := requests.
			URL("/document").
			Client(d.HttpClient).
			BodyWriter(func(w io.Writer) error {
				bodyWriter := multipart.NewWriter(w)
				bodyWriter.SetBoundary(boundary)
				bodyWriter.WriteField("source_lang", string(s))
				bodyWriter.WriteField("target_lang", string(t))
				bodyWriter.WriteField("glossary_id", options.GlossaryID)
				fileWriter, err := bodyWriter.CreateFormFile("file", options.FileName)
				if err != nil {
					return err
				}
				io.Copy(fileWriter, file)
				bodyWriter.Close()
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

// CreateGlossaryAsync creates a glossary
func (d *Translator) CreateGlossaryAsync(
	name string,
	source consts.SourceLang,
	target consts.TargetLang,
	glossary GlossaryEntries,
) tasker.TaskFunc[types.Glossary] {
	return func(ctx context.Context) (types.Glossary, error) {
		var response types.Glossary
		if len(glossary.Entries) == 0 {
			return response, errors.New("no entries provided")
		}
		tsv := glossary.ToTSV()
		response, err := tasker.Spawn(d.internalCreateGlossary(name, source, target, tsv)).Await()
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

func (d *Translator) internalCreateGlossary(
	name string,
	source consts.SourceLang,
	target consts.TargetLang,
	glossary string,
) tasker.TaskFunc[types.Glossary] {
	return func(ctx context.Context) (types.Glossary, error) {
		var response types.Glossary
		err := requests.
			URL("/glossaries").
			Method("POST").
			Client(d.HttpClient).
			Param("name", name).
			Param("source_lang", string(source)).
			Param("target_lang", string(target)).
			Param("entries", glossary).
			Param("entries_format", "tsv").
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

func (d *Translator) GetGlossariesAsync() tasker.TaskFunc[[]types.Glossary] {
	return func(ctx context.Context) ([]types.Glossary, error) {
		var response types.Glossaries
		err := requests.
			URL("/glossaries").
			Client(d.HttpClient).
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response.Glossaries, err
		}
		return response.Glossaries, nil
	}
}

// GetGlossaryAsync returns a task that can be awaited to get a glossary.
func (d *Translator) GetGlossaryDetailsAsync(id string) tasker.TaskFunc[types.Glossary] {
	return func(ctx context.Context) (types.Glossary, error) {
		var response types.Glossary
		err := requests.
			URL(fmt.Sprintf("/glossaries/%s", id)).
			Client(d.HttpClient).
			ToJSON(&response).
			Fetch(context.Background())
		if err != nil {
			return response, err
		}
		return response, nil
	}
}

// GetGlossaryEntriesAsync returns a task that can be awaited to get glossaries.
func (d *Translator) GetGlossaryEntriesAsync(id string) tasker.TaskFunc[GlossaryEntries] {
	return func(ctx context.Context) (GlossaryEntries, error) {
		var response string
		err := requests.
			URL(fmt.Sprintf("/glossaries/%s/entries", id)).
			Client(d.HttpClient).
			ToString(&response).
			Fetch(context.Background())
		if err != nil {
			return GlossaryEntries{}, err
		}
		glossaryEntries, err := NewGlossaryEntries(response)
		if err != nil {
			return GlossaryEntries{}, err
		}
		return *glossaryEntries, nil
	}
}

// DeleteGlossaryAsync returns a task that can be awaited to delete a glossary.
func (d *Translator) DeleteGlossaryAsync(id string) tasker.TaskFunc[bool] {
	return func(ctx context.Context) (bool, error) {
		err := requests.
			URL(fmt.Sprintf("/glossaries/%s", id)).
			Client(d.HttpClient).
			Delete().
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
// JSON field tags ares used as keys.
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

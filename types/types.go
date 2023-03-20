package types

import (
	"io"

	"github.com/deepl/constants"
)

type TextTranslateOptions struct {
	// "0" -> no splitting
	// "1" -> default
	// "nonewline"
	SplitSentences string `json:"split_sentences"`

	// "0" -> default
	// "1" -> formatting takes effect
	PreserveFormatting string `json:"preserve_formatting"`

	Formality constants.Formality `json:"formality"`

	GlossaryID string `json:"glossary_id"`

	// "xml" or "html"
	TagHandling string `json:"tag_handling"`

	// comma-seperated list of xml tags
	NonSplittingTags string `json:"non_splitting_tags"`

	// "0" or "1"
	OutlineDetection string `json:"outline_detection"`

	// comma-seperated list of xml tags
	SplittingTags string `json:"splitting_tags"`

	//comma-seperated list of xml tags
	IgnoreTags string `json:"ignore_tags"`
}

type DocumentTranslateOptions struct {
	FileName   string
	OutputFile io.Writer
	Formality  constants.Formality
	GlossaryID string
}

type DocumentIDAndKey struct {
	DocumentID  string `json:"document_id"`
	DocumentKey string `json:"document_key"`
}

type DocumentStatus struct {
	DocumentID       string `json:"document_id"`
	Status           string `json:"status"`
	SecondsRemaining int    `json:"seconds_remaining"`
	BilledCharacters int    `json:"billed_characters"`
	ErrorMessage     string `json:"error_message"`
}

func (d *DocumentStatus) Ok() bool {
	return d.ErrorMessage == ""
}

func (d *DocumentStatus) Done() bool {
	return d.Status == "done"
}

type Translation struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

type Translations struct {
	Translations []Translation `json:"translations"`
}

type Usage struct {
	CharacterCount    int `json:"character_count"`
	CharacterLimit    int `json:"character_limit"`
	DocumentLimit     int `json:"document_limit"`
	DocumentCount     int `json:"document_count"`
	TeamDocumentLimit int `json:"team_document_limit"`
	TeamDocumentCount int `json:"team_document_count"`
}

type SupportedLanguage struct {
	Language          string `json:"language"`
	Name              string `json:"name"`
	SupportsFormality bool   `json:"supports_formality"`
}

type GlossaryLanguagePair struct {
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

type GlossaryLanguagePairs struct {
	SupportedLanguages []GlossaryLanguagePair `json:"supported_languages"`
}

package deepl

import (
	"fmt"
	"os"
	"testing"

	"github.com/anthdm/tasker"
	"github.com/deepl/constants"
	"github.com/deepl/types"
	"github.com/google/go-cmp/cmp"
)

func TestTranslateTextAsync(t *testing.T) {
	key := "key"
	text := "proton beam"
	translator, err := NewTranslator(key, types.TranslatorOptions{
		ServerURL:         "http://localhost:3000/v2",
		SendPlattformInfo: true,
	})
	if err != nil {
		t.Errorf(err.Error())
	}
	options := &types.TextTranslateOptions{}
	res := tasker.Spawn(translator.TranslateTextAsync(text, string(constants.SourceLangEnglish), string(constants.TargetLangGerman), options))
	translation, err := res.Await()
	if err != nil {
		fmt.Println(err)
	}
	want := types.Translation{
		DetectedSourceLanguage: "EN",
		Text:                   "Protonenstrahl",
	}
	if !cmp.Equal(translation.Translations[0], want) {
		t.Errorf("got %s, want %s", translation.Translations[0], want)
	}
}

func TestTranslateDocumentAsync(t *testing.T) {
	key := "key"

	translator, err := NewTranslator(key, types.TranslatorOptions{ServerURL: "http://localhost:3000/v2"})
	if err != nil {
		t.Errorf(err.Error())
	}
	file, _ := os.Create("result.txt")
	defer file.Close()
	options := types.DocumentTranslateOptions{
		FileName:   "result.txt",
		OutputFile: file,
	}
	input, _ := os.Open("test.txt")
	res := tasker.Spawn(translator.TranslateDocumentAsync(string(constants.SourceLangEnglish), string(constants.TargetLangGerman), input, options))
	_, err = res.Await()
	if err != nil {
		fmt.Println(err)
	}
}

func TestConstructUserAgent(t *testing.T) {
	appInfo := types.AppInfo{
		AppName:    "TestApp",
		AppVersion: "1.0",
	}
	got := constructUserAgentString(true, appInfo)
	want := "deepl-golang/1.0 windows go1.20.1 TestApp/1.0"
	if got != want {
		t.Errorf("want: %s, got %s", want, got)
	}
}

func TestToSCV(t *testing.T) {
	entries, _ := NewGlossaryEntries(map[string]string{
		"proton": "Protonen",
		"beam":   "Strahl",
	})
	got := entries.ToTSV()
	want := "proton\tProtonen\nbeam\tStrahl"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFromTSV(t *testing.T) {
	tsv := "proton\tProtonen\nbeam\tStrahl"
	entries, _ := NewGlossaryEntries(tsv)
	got := entries.Entries
	want := map[string]string{
		"proton": "Protonen",
		"beam":   "Strahl",
	}
	if !cmp.Equal(got, want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestValidateGlossaryTerm(t *testing.T) {
	terms := map[string]bool{
		"proton": true,
		"beam\n": false,
		"beam\t": false,
		"beam\r": false,
	}
	entries, _ := NewGlossaryEntries(map[string]string{})
	for k, v := range terms {
		err := entries.validateGlossaryTerm(k)
		if err == nil && !v {
			t.Errorf("term should be valid: %s", k)
		}
		if err != nil && v {
			t.Errorf("term should be invalid: %s", k)
		}
	}
}

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
	translator := NewTranslator(key)
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
	translator := NewTranslator(key)
	file, _ := os.Create("result.txt")
	defer file.Close()
	options := types.DocumentTranslateOptions{
		FileName:   "result.txt",
		OutputFile: file,
	}
	input, _ := os.Open("test.txt")
	res := tasker.Spawn(translator.TranslateDocumentAsync(string(constants.SourceLangEnglish), string(constants.TargetLangGerman), input, options))
	_, err := res.Await()
	if err != nil {
		fmt.Println(err)
	}
}

package deepl

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/anthdm/tasker"
	"github.com/deepl/constants"
	"github.com/deepl/types"
	"github.com/google/go-cmp/cmp"
)

func MakeTranslator(header map[string]string) (*Translator, error) {
	key := "auth_key"
	translator, err := NewTranslator(key, types.TranslatorOptions{
		ServerURL:         "http://localhost:3000/v2",
		SendPlattformInfo: true,
		Headers:           header,
		TimeOut:           time.Duration(5) * time.Second,
		Retries:           5,
	})
	if err != nil {
		return nil, err
	}
	return translator, nil
}

func TestTranslator_TranslateTextAsync(t *testing.T) {
	text := []string{"proton beam", "proton beam"}
	translator, err := MakeTranslator(map[string]string{
		"mock-server-session":           "TooManyRequests",
		"mock-server-session-429-count": "4",
	})
	options := &types.TextTranslateOptions{}
	res := tasker.Spawn(translator.TranslateTextAsync(text, constants.SourceLangEnglish, constants.TargetLangGerman, options))
	translations, err := res.Await()
	if err != nil {
		fmt.Println(err)
	}
	want := types.Translations{
		Translations: []types.Translation{
			{
				DetectedSourceLanguage: "EN",
				Text:                   "Protonenstrahl",
			},
			{
				DetectedSourceLanguage: "EN",
				Text:                   "Protonenstrahl",
			},
		},
	}
	if !cmp.Equal(translations, want.Translations) {
		t.Errorf("got %s, want %s", translations, want)
	}
}

func TestTranslator_GetUsageAsync(t *testing.T) {
	translator, err := MakeTranslator(map[string]string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	res := tasker.Spawn(translator.GetUsageAsync())
	usage, err := res.Await()
	if err != nil {
		fmt.Println(err)
	}
	want := types.Usage{
		CharacterCount:    0,
		CharacterLimit:    2000000,
		DocumentLimit:     10000,
		DocumentCount:     0,
		TeamDocumentLimit: 0,
		TeamDocumentCount: 0,
	}
	if !cmp.Equal(usage, want) {
		t.Errorf("got %+v, want %+v", usage, want)
	}
}

func TestTranslator_TranslateDocumentAsync(t *testing.T) {
	translator, err := MakeTranslator(map[string]string{})
	file, _ := os.Create("result.txt")
	defer file.Close()
	options := types.DocumentTranslateOptions{
		FileName:   "result.txt",
		OutputFile: file,
	}
	input, _ := os.Open("test.txt")
	res := tasker.Spawn(translator.TranslateDocumentAsync(constants.SourceLangEnglish, constants.TargetLangGerman, input, options))
	_, err = res.Await()
	if err != nil {
		fmt.Println(err)
	}
}

func TestTranslator_Glossary(t *testing.T) {
	translator, _ := MakeTranslator(map[string]string{})
	entriesString := "proton\tProtonen\nbeam\tStrahl"
	entriesMap := map[string]string{
		"proton": "Protonen",
		"beam":   "Strahl",
	}
	entries, _ := NewGlossaryEntries(entriesString)
	// Create Glossary
	want, err := tasker.Spawn(translator.CreateGlossaryAsync("test", constants.SourceLangEnglish, constants.TargetLangGerman, *entries)).Await()
	if err != nil {
		t.Error(err)
	}
	// Get Glossary Details
	res := tasker.Spawn(translator.GetGlossaryDetailsAsync(want.GlossaryID))
	glossary, err := res.Await()
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(glossary, want) {
		t.Errorf("Retrieved Glossary invalid, got %+v, want %+v", glossary, want)
	}
	// Get Glossary Entries
	glossaryEntries, err := tasker.Spawn(translator.GetGlossaryEntriesAsync(want.GlossaryID)).Await()
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(glossaryEntries.Entries, entriesMap) {
		t.Errorf("Glossary Entry invalid, got %s, want %s", glossaryEntries, entriesString)
	}

	glossaries, err := tasker.Spawn(translator.GetGlossariesAsync()).Await()
	if err != nil {
		t.Error(err)
	}
	if len(glossaries) == 1 {
		if !cmp.Equal(glossaries[0], want) {
			t.Errorf("Retrieved Glossaries invalid, got %+v, want %+v", glossaries[0], want)
		}
	} else {
		t.Error("Glossary was not created")
	}
	// Delete Glossary
	_, err = tasker.Spawn(translator.DeleteGlossaryAsync(want.GlossaryID)).Await()
	if err != nil {
		t.Error(err)
	}
	// Get Glossary Details
	glossary, err = tasker.Spawn(translator.GetGlossaryDetailsAsync(want.GlossaryID)).Await()
	if err == nil {
		t.Error(err)
	}
	if glossary != (types.Glossary{}) {
		t.Error("Glossary was not deleted")
	}
}

func TestTranslator_ConstructUserAgent(t *testing.T) {
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

func TestGlossaryEntries_ToTSV(t *testing.T) {
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

func TestGlossaryEntries_FromTSV(t *testing.T) {
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

func TestGlossaryEntries_ValidateGlossaryTerm(t *testing.T) {
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

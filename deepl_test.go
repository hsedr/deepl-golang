package deepl

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/anthdm/tasker"
	"github.com/google/go-cmp/cmp"
	"github.com/hsedr/deepl-golang/consts"
	"github.com/hsedr/deepl-golang/types"
)

func MakeTranslator(header map[string]string) (*Translator, error) {
	key := "auth_key"
	translator, err := NewTranslator(key,
		WithServerURL("http://localhost:3000/v2"),
		WithUserAgent(true, types.AppInfo{}),
		WithHeaders(header),
		WithTimeOut(time.Duration(5)*time.Second),
		WithRetries(5),
	)
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
	res := tasker.Spawn(translator.TranslateTextAsync(text, consts.SourceLangEnglish, consts.TargetLangGerman))
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
	if usage == (types.Usage{}) {
		t.Error("Error occured while retreiving usage")
	}
}

func TestTranslator_TranslateDocumentAsync(t *testing.T) {
	translator, err := MakeTranslator(map[string]string{
		"mock-server-session":                    "TranslateDocumentTranslateTime",
		"mock-server-session-doc-translate-time": "10000",
	})
	file, _ := os.Create("result.txt")
	defer file.Close()
	input, _ := os.Open("test.txt")
	defer input.Close()
	res := tasker.Spawn(translator.TranslateDocumentAsync(consts.SourceLangEnglish, consts.TargetLangGerman, input, file))
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
	want, err := tasker.Spawn(translator.CreateGlossaryAsync("test", consts.SourceLangEnglish, consts.TargetLangGerman, *entries)).Await()
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
	want := "deepl-golang/1.0 windows go1.20.2 TestApp/1.0"
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

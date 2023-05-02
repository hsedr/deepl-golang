package main

import (
	"log"

	"github.com/anthdm/tasker"
	"github.com/hsedr/deepl-golang"
	"github.com/hsedr/deepl-golang/consts"
)

func main() {
	translator, err := deepl.NewTranslator("YOUR_API_KEY")
	if err != nil {
		log.Fatal(err)
	}
	task := tasker.Spawn(translator.TranslateTextAsync([]string{"proto beam"}, consts.SourceLangEnglish, consts.TargetLangGerman))
	result, err := task.Await()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(result)
}

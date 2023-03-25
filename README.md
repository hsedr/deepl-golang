# deepl-golang

## Overview

**Unofficial DeepL API iplementation currently in progress.**

Most of the functionalities of the other DeepL SDKs are implemented but need further testing.

For now, the Methods perform reliably when testing against the [DeepL-Mock-API](https://github.com/DeepLcom/deepl-mock).

All HTTP-related methods are carried out asynchronously, using a [tasker library](https://github.com/anthdm/tasker) which allows to await results.

## How to Use

### Translate Texts
```golang
text := []string{"proton beam"}
key := "auth_key"
translator, _ := NewTranslator(key, types.TranslatorOptions{})
options := &types.TextTranslateOptions{}
task := tasker.Spawn(translator.TranslateTextAsync(text, constants.SourceLangEnglish, constants.TargetLangGerman, options))
translations, err := task.Await()
if err != nil {
  fmt.Println(err)
}
fmt.Println(translations[0].Text) // Protonenstrahl
```

### Get Usage and other general information
```golang
key := "auth_key"
translator, err := NewTranslator(key, types.TranslatorOptions{})
if err != nil {		
  fmt.Println(err)
}
res := tasker.Spawn(translator.GetUsageAsync())
usage, err := res.Await()
if err != nil {
  fmt.Println(err)
}
fmt.Printf("%+v", usage)
```

### Translate Documents
```golang
key := "auth_key"
translator, err := NewTranslator(key, types.TranslatorOptions{})
file, _ := os.Create("result.txt")
defer file.Close()
options := types.DocumentTranslateOptions{
	FileName:   "result.txt",
	OutputFile: file,
}
input, _ := os.Open("test.txt")
defer input.Close()
res := tasker.Spawn(translator.TranslateDocumentAsync(constants.SourceLangEnglish, constants.TargetLangGerman, input, options))
_, err = res.Await()
if err != nil {
  fmt.Println(err)
}
```

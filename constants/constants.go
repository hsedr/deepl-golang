package constants

type Formality string
type SourceLang string
type TargetLang string
type DocumentStatusCode string

const (
	Default    Formality = "default"
	More       Formality = "more"
	Less       Formality = "less"
	PreferMore Formality = "prefer_more"
	PreferLess Formality = "prefer_less"
)

const (
	DocumentStatusQueued      DocumentStatusCode = "queued"
	DocumentStatusTranslating DocumentStatusCode = "translating"
	DocumentStatusError       DocumentStatusCode = "error"
	DocumentStatusDone        DocumentStatusCode = "done"
)

const (
	SourceLangBulgarian  SourceLang = "BG"
	SourceLangCzech      SourceLang = "CS"
	SourceLangDanish     SourceLang = "DA"
	SourceLangGreek      SourceLang = "EL"
	SourceLangGerman     SourceLang = "DE"
	SourceLangEnglish    SourceLang = "EN"
	SourceLangSpanish    SourceLang = "ES"
	SourceLangEstonian   SourceLang = "ET"
	SourceLangFinnish    SourceLang = "FI"
	SourceLangFrench     SourceLang = "FR"
	SourceLangHungarian  SourceLang = "HU"
	SourceLangIndonesian SourceLang = "ID"
	SourceLangItalian    SourceLang = "IT"
	SourceLangJapanese   SourceLang = "JA"
	SourceLangKorean     SourceLang = "KO"
	SourceLangLithuanian SourceLang = "LT"
	SourceLangLatvian    SourceLang = "LV"
	SourceLangNorwegian  SourceLang = "NB"
	SourceLangDutch      SourceLang = "NL"
	SourceLangPolish     SourceLang = "PL"
	SourceLangPortuguese SourceLang = "PT"
	SourceLangRomanian   SourceLang = "RO"
	SourceLangRussian    SourceLang = "RU"
	SourceLangSlovak     SourceLang = "SK"
	SourceLangSlovenian  SourceLang = "SL"
	SourceLangSwedish    SourceLang = "SV"
	SourceLangTurkish    SourceLang = "TR"
	SourceLangUkrainian  SourceLang = "UK"
	SourceLangChinese    SourceLang = "ZH"
)

const (
	TargetLangBulgarian           TargetLang = "BG"
	TargetLangCzech               TargetLang = "CS"
	TargetLangDanish              TargetLang = "DA"
	TargetLangGreek               TargetLang = "EL"
	TargetLangGerman              TargetLang = "DE"
	TargetLangEnglishUS           TargetLang = "EN-US"
	TargetLangEnglishGB           TargetLang = "EN-GB"
	TargetLangSpanish             TargetLang = "ES"
	TargetLangEstonian            TargetLang = "ET"
	TargetLangFinnish             TargetLang = "FI"
	TargetLangFrench              TargetLang = "FR"
	TargetLangHungarian           TargetLang = "HU"
	TargetLangIndonesian          TargetLang = "ID"
	TargetLangItalian             TargetLang = "IT"
	TargetLangJapanese            TargetLang = "JA"
	TargetLangKorean              TargetLang = "KO"
	TargetLangLithuanian          TargetLang = "LT"
	TargetLangLatvian             TargetLang = "LV"
	TargetLangNorwegian           TargetLang = "NB"
	TargetLangDutch               TargetLang = "NL"
	TargetLangPolish              TargetLang = "PL"
	TargetLangPortugueseBrazilian TargetLang = "PT-BR"
	TargetLangPortuguese          TargetLang = "PT-PT"
	TargetLangRomanian            TargetLang = "RO"
	TargetLangRussian             TargetLang = "RU"
	TargetLangSlovak              TargetLang = "SK"
	TargetLangSlovenian           TargetLang = "SL"
	TargetLangSwedish             TargetLang = "SV"
	TargetLangTurkish             TargetLang = "TR"
	TargetLangUkrainian           TargetLang = "UK"
	TargetLangChinese             TargetLang = "ZH"
)

package errorsc

const (
	LangEN = "en"
	LangCN = "cn"
)

var CurrentLang = LangEN

func SetLanguage(lang string) {
	if lang != LangEN && lang != LangCN {
		CurrentLang = LangEN
	}
	CurrentLang = lang
}

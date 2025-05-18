package interfaces

import "time"

type Gender int32

const (
	MALE Gender = iota + 1
	FEMALE
	NEUTRAL
)

type LanguageCode struct {
	Code     string `firestore:"code"`
	Name     string `firestore:"name"`
	Language string `firestore:"language"`
	Country  string `firestore:"country"`
	Platform string `firestore:"platform"`
}

type Voice struct {
	Name                   string `firestore:"name"`
	Language               string `firestore:"language"`
	LanguageCode           string `firestore:"languageCodes"`
	SsmlGender             Gender `firestore:"ssmlGender"`
	NaturalSampleRateHertz int32  `firestore:"naturalSampleRateHertz"`
	Platform               string `firestore:"platform"`
	SampleURL              string `firestore:"sampleUrl,omitempty"`
}

type Language struct {
	Name     string `firestore:"name"`
	Code     string `firestore:"code"`
	Platform string `firestore:"platform"`
}

type Title struct {
	Name         string
	TitleLang    string
	ToVoice      string
	FromVoice    string
	Pause        int
	TitlePhrases []Phrase
	ToPhrases    []Phrase
	Pattern      int
}

type Phrase struct {
	ID   int
	Text string
}

type Status int

const (
	New Status = iota
	Used
)

type Token struct {
	UploadUsed bool
	TimesUsed  int
	Created    time.Time
	Hash       string
}

type FirestoreToken struct {
	UploadUsed bool
	TimesUsed  int
	Created    time.Time
}

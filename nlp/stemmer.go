// Package nlp implémente le traitement linguistique français.
// Stemmer Snowball FR intégré — zéro dépendance externe.
package nlp

import (
	"strings"
	"unicode"
)

// Stemmer applique l'algorithme Snowball pour le français.
type Stemmer struct {
	stopwords map[string]bool
}

// NewStemmer crée un stemmer français.
func NewStemmer() *Stemmer {
	return &Stemmer{
		stopwords: buildStopwords(),
	}
}

// Stem retourne la racine d'un mot français.
func (s *Stemmer) Stem(word string) string {
	w := strings.ToLower(strings.TrimSpace(word))
	if len(w) < 3 {
		return w
	}
	if s.stopwords[w] {
		return w
	}
	return snowballFR(w)
}

// StemAll applique Stem à chaque mot.
func (s *Stemmer) StemAll(words []string) []string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		st := s.Stem(w)
		if st != "" {
			out = append(out, st)
		}
	}
	return out
}

// IsStopword teste si un mot est un mot vide.
func (s *Stemmer) IsStopword(word string) bool {
	return s.stopwords[strings.ToLower(word)]
}

// Tokenize découpe un texte en mots normalisés.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	// Normaliser les accents pour le matching
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '\'' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				t := current.String()
				// Ignorer les mots trop courts (sauf s'ils sont un seul mot)
				if len(t) >= 2 {
					tokens = append(tokens, t)
				}
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		t := current.String()
		if len(t) >= 2 {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// --- Snowball FR simplifié ---

func snowballFR(word string) string {
	// Normaliser les accents
	w := normalizeAccents(word)

	// Trouver les régions R1, R2
	r1, r2 := regions(w)

	// Étape 1 : suffixes standards
	w = step1(w, r1, r2)

	// Étape 2 : suffixes verbaux
	w = step2(w, r1, r2)

	// Étape 3 : nettoyage final
	w = step3(w)

	return w
}

func normalizeAccents(w string) string {
	replacer := strings.NewReplacer(
		"â", "a", "à", "a", "ä", "a",
		"ê", "e", "è", "e", "é", "e", "ë", "e",
		"î", "i", "ï", "i",
		"ô", "o", "ö", "o",
		"û", "u", "ù", "u", "ü", "u",
		"ÿ", "y", "ç", "c",
	)
	return replacer.Replace(w)
}

func regions(w string) (int, int) {
	// R1 = après la première voyelle suivie d'une consonne
	// R2 = R1 appliqué au reste
	r1 := len(w)
	r2 := len(w)

	vowels := "aeiouy"

	for i := 1; i < len(w); i++ {
		if strings.ContainsRune(vowels, rune(w[i-1])) && !strings.ContainsRune(vowels, rune(w[i])) {
			r1 = i + 1
			break
		}
	}

	for i := r1 + 1; i < len(w); i++ {
		if strings.ContainsRune(vowels, rune(w[i-1])) && !strings.ContainsRune(vowels, rune(w[i])) {
			r2 = i + 1
			break
		}
	}

	return r1, r2
}

func step1(w string, r1, r2 int) string {
	// Suffixes nominaux/adjectivaux (les plus longs d'abord)
	suffixes := []struct {
		suffix string
		region int // 1=R1, 2=R2
	}{
		{"issements", 2}, {"issement", 2},
		{"atrices", 2}, {"atrice", 2}, {"ateurs", 2}, {"ateur", 2}, {"ations", 2}, {"ation", 2},
		{"logies", 2}, {"logie", 2},
		{"usions", 2}, {"usion", 2}, {"utions", 2}, {"ution", 2},
		{"ences", 2}, {"ence", 2}, {"ances", 2}, {"ance", 2},
		{"ments", 2}, {"ment", 2},
		{"ites", 2}, {"ite", 2}, {"ites", 2},
		{"ives", 2}, {"ive", 2}, {"ifs", 2}, {"if", 2},
		{"euses", 2}, {"euse", 2}, {"eux", 2},
		{"ables", 2}, {"able", 2}, {"ibles", 2}, {"ible", 2},
		{"istes", 2}, {"iste", 2},
	}

	for _, s := range suffixes {
		if strings.HasSuffix(w, s.suffix) {
			pos := len(w) - len(s.suffix)
			region := r1
			if s.region == 2 {
				region = r2
			}
			if pos >= region {
				return w[:pos]
			}
			break
		}
	}
	return w
}

func step2(w string, r1, r2 int) string {
	// Suffixes verbaux
	verbSuffixes := []string{
		"eraients", "eraient", "assions", "assent", "issions", "issent",
		"erions", "erons", "eront", "irons", "iront",
		"erait", "erai", "eras", "irait", "irai",
		"aient", "antes", "ante", "ants", "ant",
		"ions", "ient",
		"ees", "ee", "er", "ez", "es",
		"ons", "ont", "ait", "ais",
		"ir", "is", "it",
	}

	for _, s := range verbSuffixes {
		if strings.HasSuffix(w, s) {
			pos := len(w) - len(s)
			if pos >= r1 {
				return w[:pos]
			}
		}
	}
	return w
}

func step3(w string) string {
	// Nettoyage : supprimer le e/s/t final
	if len(w) > 2 {
		last := w[len(w)-1]
		if last == 's' {
			w = w[:len(w)-1]
		}
	}
	if len(w) > 2 && w[len(w)-1] == 'e' {
		w = w[:len(w)-1]
	}
	return w
}

func buildStopwords() map[string]bool {
	words := []string{
		"au", "aux", "avec", "ce", "ces", "dans", "de", "des", "du",
		"elle", "en", "et", "eux", "il", "ils", "je", "la", "le", "les",
		"leur", "lui", "ma", "mais", "me", "meme", "mes", "moi", "mon",
		"ne", "nos", "notre", "nous", "on", "ou", "par", "pas", "pour",
		"qu", "que", "qui", "sa", "se", "ses", "son", "sur", "ta", "te",
		"tes", "toi", "ton", "tu", "un", "une", "vos", "votre", "vous",
		"est", "sont", "soit", "etre", "avoir", "fait", "comme", "tout",
		"tous", "plus", "cette", "bien", "peut", "entre", "aussi",
	}
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

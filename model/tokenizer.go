// Package model implémente un mini-transformer « from scratch » — embeddings,
// self-attention, rétropropagation manuelle, SGD — sans framework de machine
// learning. Le domaine d'exemple du dépôt est le contrôle d'accès aux
// documents (package policy), dont les requêtes sont déjà fournies sous forme
// de tokens par Request.Tokens ; Tokenize ci-dessous reste une utilité
// générique pour découper du texte (code ou langage naturel) si vous adaptez
// le modèle à un autre jeu de données.
package model

import (
	"strings"
	"unicode"
)

// multiCharOps liste les opérateurs à plusieurs caractères à reconnaître
// avant de retomber sur un découpage caractère par caractère.
var multiCharOps = []string{
	":=", "=>", "==", "!=", "<=", ">=", "&&", "||", "++", "--", "->", "...",
}

// Tokenize découpe un extrait de code (ou de texte) en une séquence de
// tokens textuels. Les identifiants et mots-clés sont conservés tels quels
// (la casse compte : "func" et "function" doivent rester deux tokens
// différents). Les littéraux de chaînes et les nombres sont normalisés en
// tokens génériques pour ne pas polluer le vocabulaire avec du contenu non
// porteur de signal pour la tâche de classification.
func Tokenize(src string) []string {
	var tokens []string
	runes := []rune(src)
	i := 0
	for i < len(runes) {
		r := runes[i]

		switch {
		case unicode.IsSpace(r):
			i++

		case r == '"' || r == '\'' || r == '`':
			quote := r
			j := i + 1
			for j < len(runes) && runes[j] != quote {
				if runes[j] == '\\' && j+1 < len(runes) {
					j++
				}
				j++
			}
			tokens = append(tokens, "STR")
			i = j + 1

		case unicode.IsDigit(r):
			j := i
			for j < len(runes) && (unicode.IsDigit(runes[j]) || runes[j] == '.') {
				j++
			}
			tokens = append(tokens, "NUM")
			i = j

		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}
			tokens = append(tokens, string(runes[i:j]))
			i = j

		default:
			matched := false
			for _, op := range multiCharOps {
				n := len(op)
				if i+n <= len(runes) && string(runes[i:i+n]) == op {
					tokens = append(tokens, op)
					i += n
					matched = true
					break
				}
			}
			if !matched {
				tokens = append(tokens, string(r))
				i++
			}
		}
	}
	return tokens
}

// unkToken est le token de repli utilisé pour tout mot jamais vu à
// l'entraînement — indispensable pour classifier du code réellement inédit.
const unkToken = "UNK"

// Vocab associe chaque token connu à un indice entier. L'indice de unkToken
// est toujours 0.
type Vocab struct {
	TokenToID map[string]int
}

// NewVocab construit un vocabulaire à partir d'un corpus déjà tokenisé.
func NewVocab(corpus [][]string) *Vocab {
	v := &Vocab{TokenToID: map[string]int{unkToken: 0}}
	for _, tokens := range corpus {
		for _, t := range tokens {
			if _, ok := v.TokenToID[t]; !ok {
				v.TokenToID[t] = len(v.TokenToID)
			}
		}
	}
	return v
}

// Size retourne la taille du vocabulaire (UNK compris).
func (v *Vocab) Size() int {
	return len(v.TokenToID)
}

// Encode convertit des tokens textuels en indices, en retombant sur UNK
// pour tout token absent du vocabulaire d'entraînement.
func (v *Vocab) Encode(tokens []string) []int {
	ids := make([]int, len(tokens))
	for i, t := range tokens {
		if id, ok := v.TokenToID[t]; ok {
			ids[i] = id
		} else {
			ids[i] = 0
		}
	}
	return ids
}

// String liste les tokens connus, pour du debug rapide.
func (v *Vocab) String() string {
	keys := make([]string, 0, len(v.TokenToID))
	for k := range v.TokenToID {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

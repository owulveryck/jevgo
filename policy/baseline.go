package policy

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/owulveryck/jevgo/model"
)

// BagOfEmbeddings est un classifieur volontairement SANS mécanisme
// d'attention : chaque token a un embedding, on en prend la moyenne, et une
// couche linéaire classe le résultat — aucun mélange entre positions.
//
// C'est la référence à laquelle comparer le transformer de jevgo/model sur
// les requêtes-faille : si les deux modèles s'accordent, l'attention n'a
// rien apporté sur cette tâche ; s'ils divergent, c'est que le mélange des
// tokens entre eux (ce que l'attention fait et ce sac de mots ne fait pas)
// a changé la prédiction.
type BagOfEmbeddings struct {
	Vocab     *model.Vocab
	Embedding [][]float64 // [vocabSize][dim]
	WClass    [][]float64 // [numClasses][dim]
	dim       int
}

// numClasses = 0 (deny) / 1 (allow), comme model.NumClasses.
const numClasses = 2

func NewBagOfEmbeddings(vocab *model.Vocab, dim int, seed int64) *BagOfEmbeddings {
	rng := rand.New(rand.NewSource(seed))
	randMat := func(rows, cols int) [][]float64 {
		m := make([][]float64, rows)
		for i := range m {
			m[i] = make([]float64, cols)
			for j := range m[i] {
				m[i][j] = (rng.Float64() - 0.5) * 0.2
			}
		}
		return m
	}
	return &BagOfEmbeddings{
		Vocab:     vocab,
		Embedding: randMat(vocab.Size(), dim),
		WClass:    randMat(numClasses, dim),
		dim:       dim,
	}
}

func softmax2(logits []float64) []float64 {
	maxVal := math.Max(logits[0], logits[1])
	e0 := math.Exp(logits[0] - maxVal)
	e1 := math.Exp(logits[1] - maxVal)
	sum := e0 + e1
	return []float64{e0 / sum, e1 / sum}
}

func (b *BagOfEmbeddings) pool(ids []int) []float64 {
	pooled := make([]float64, b.dim)
	for _, tok := range ids {
		for d := 0; d < b.dim; d++ {
			pooled[d] += b.Embedding[tok][d] / float64(len(ids))
		}
	}
	return pooled
}

func (b *BagOfEmbeddings) logits(pooled []float64) []float64 {
	logits := make([]float64, numClasses)
	for k := 0; k < numClasses; k++ {
		sum := 0.0
		for d := 0; d < b.dim; d++ {
			sum += b.WClass[k][d] * pooled[d]
		}
		logits[k] = sum
	}
	return logits
}

// Forward moyenne les embeddings des tokens (aucun mélange entre positions)
// puis applique la tête de classification linéaire.
func (b *BagOfEmbeddings) Forward(ids []int) (int, []float64) {
	probs := softmax2(b.logits(b.pool(ids)))
	best := 0
	if probs[1] > probs[0] {
		best = 1
	}
	return best, probs
}

// TrainStep : passe avant, perte d'entropie croisée, rétropropagation
// directe (pas d'attention à traverser) et mise à jour SGD.
func (b *BagOfEmbeddings) TrainStep(ids []int, label int, lr float64) float64 {
	n := len(ids)
	pooled := b.pool(ids)
	probs := softmax2(b.logits(pooled))
	loss := -math.Log(probs[label] + 1e-12)

	dLogits := []float64{probs[0], probs[1]}
	dLogits[label] -= 1

	// Gradient vers pooled, calculé AVANT toute mise à jour de WClass.
	dPooled := make([]float64, b.dim)
	for k := 0; k < numClasses; k++ {
		for d := 0; d < b.dim; d++ {
			dPooled[d] += dLogits[k] * b.WClass[k][d]
		}
	}

	// Mise à jour de la tête de classification.
	for k := 0; k < numClasses; k++ {
		for d := 0; d < b.dim; d++ {
			b.WClass[k][d] -= lr * dLogits[k] * pooled[d]
		}
	}

	// Le pooling est une moyenne : le gradient est réparti également,
	// puis chaque embedding concerné est mis à jour.
	for _, tok := range ids {
		for d := 0; d < b.dim; d++ {
			b.Embedding[tok][d] -= lr * dPooled[d] / float64(n)
		}
	}
	return loss
}

type serializedBaseline struct {
	Vocab     map[string]int `json:"vocab"`
	Embedding [][]float64    `json:"embedding"`
	WClass    [][]float64    `json:"wclass"`
	Dim       int            `json:"dim"`
}

func (b *BagOfEmbeddings) Save(path string) error {
	s := serializedBaseline{
		Vocab:     b.Vocab.TokenToID,
		Embedding: b.Embedding,
		WClass:    b.WClass,
		Dim:       b.dim,
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadBagOfEmbeddings(path string) (*BagOfEmbeddings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s serializedBaseline
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &BagOfEmbeddings{
		Vocab:     &model.Vocab{TokenToID: s.Vocab},
		Embedding: s.Embedding,
		WClass:    s.WClass,
		dim:       s.Dim,
	}, nil
}

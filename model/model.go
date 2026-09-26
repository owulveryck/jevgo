package model

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
)

const (
	DModel     = 16 // Dimension latente de chaque token
	NumClasses = 2  // 0 = deny, 1 = allow
)

// Étiquettes du dataset — à utiliser plutôt que des 0/1 en dur.
const (
	LabelDeny  = 0
	LabelAllow = 1
)

// Model est un mini-transformer à une seule tête d'attention :
// Embedding -> self-attention (Q,K,V) -> moyenne (pooling) -> classifieur linéaire.
// Contrairement à une démo qui injecterait des poids à la main, TOUS les
// poids ci-dessous sont appris par descente de gradient (voir TrainStep).
type Model struct {
	Vocab      *Vocab
	Embedding  [][]float64 // [vocabSize][DModel]
	Wq, Wk, Wv [][]float64 // [DModel][DModel]
	WClass     [][]float64 // [NumClasses][DModel]
}

func randMatrix(rows, cols int, rng *rand.Rand) [][]float64 {
	m := make([][]float64, rows)
	for i := range m {
		m[i] = make([]float64, cols)
		for j := range m[i] {
			m[i][j] = (rng.Float64() - 0.5) * 0.2
		}
	}
	return m
}

func zeros(rows, cols int) [][]float64 {
	m := make([][]float64, rows)
	for i := range m {
		m[i] = make([]float64, cols)
	}
	return m
}

func matVecMul(mat [][]float64, vec []float64) []float64 {
	out := make([]float64, len(mat))
	for i := range mat {
		sum := 0.0
		for j := range vec {
			sum += mat[i][j] * vec[j]
		}
		out[i] = sum
	}
	return out
}

func dot(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func softmax(logits []float64) []float64 {
	maxVal := -math.MaxFloat64
	for _, v := range logits {
		if v > maxVal {
			maxVal = v
		}
	}
	expSum := 0.0
	out := make([]float64, len(logits))
	for i, v := range logits {
		out[i] = math.Exp(v - maxVal)
		expSum += out[i]
	}
	for i := range out {
		out[i] /= expSum
	}
	return out
}

// New crée un modèle à poids aléatoires, à partir d'un vocabulaire déjà
// construit sur le corpus d'entraînement (voir NewVocab).
func New(vocab *Vocab, seed int64) *Model {
	rng := rand.New(rand.NewSource(seed))
	return &Model{
		Vocab:     vocab,
		Embedding: randMatrix(vocab.Size(), DModel, rng),
		Wq:        randMatrix(DModel, DModel, rng),
		Wk:        randMatrix(DModel, DModel, rng),
		Wv:        randMatrix(DModel, DModel, rng),
		WClass:    randMatrix(NumClasses, DModel, rng),
	}
}

// cache conserve les valeurs intermédiaires d'une passe avant, nécessaires
// pour calculer la rétropropagation.
type cache struct {
	x       [][]float64 // embedding par position
	Q, K, V [][]float64
	attn    [][]float64 // attn[i][j] = poids d'attention du token i sur j
	context [][]float64 // vecteur de contexte par position
	pooled  []float64
	probs   []float64
}

func (m *Model) forward(tokens []int) *cache {
	n := len(tokens)
	c := &cache{}

	c.x = make([][]float64, n)
	for i, tok := range tokens {
		c.x[i] = m.Embedding[tok]
	}

	c.Q = make([][]float64, n)
	c.K = make([][]float64, n)
	c.V = make([][]float64, n)
	for i := 0; i < n; i++ {
		c.Q[i] = matVecMul(m.Wq, c.x[i])
		c.K[i] = matVecMul(m.Wk, c.x[i])
		c.V[i] = matVecMul(m.Wv, c.x[i])
	}

	scale := math.Sqrt(float64(DModel))
	c.attn = make([][]float64, n)
	c.context = make([][]float64, n)
	for i := 0; i < n; i++ {
		scores := make([]float64, n)
		for j := 0; j < n; j++ {
			scores[j] = dot(c.Q[i], c.K[j]) / scale
		}
		c.attn[i] = softmax(scores)

		ctx := make([]float64, DModel)
		for j := 0; j < n; j++ {
			for d := 0; d < DModel; d++ {
				ctx[d] += c.attn[i][j] * c.V[j][d]
			}
		}
		c.context[i] = ctx
	}

	c.pooled = make([]float64, DModel)
	for i := 0; i < n; i++ {
		for d := 0; d < DModel; d++ {
			c.pooled[d] += c.context[i][d] / float64(n)
		}
	}

	logits := matVecMul(m.WClass, c.pooled)
	c.probs = softmax(logits)
	return c
}

// Forward fait une passe d'inférence : classe prédite + distribution complète.
func (m *Model) Forward(tokens []int) (int, []float64) {
	c := m.forward(tokens)
	best := 0
	if c.probs[1] > c.probs[0] {
		best = 1
	}
	return best, c.probs
}

// TrainStep effectue une passe avant, calcule la perte d'entropie croisée,
// rétropropage analytiquement à travers la tête de classification, le
// pooling, l'auto-attention (softmax compris) et les projections Q/K/V
// jusqu'aux embeddings, puis met à jour tous les poids par SGD.
func (m *Model) TrainStep(tokens []int, label int, lr float64) float64 {
	n := len(tokens)
	c := m.forward(tokens)

	loss := -math.Log(c.probs[label] + 1e-12)

	// --- Tête de classification ---
	dLogits := make([]float64, NumClasses)
	copy(dLogits, c.probs)
	dLogits[label] -= 1 // gradient softmax + cross-entropy

	dWClass := zeros(NumClasses, DModel)
	dPooled := make([]float64, DModel)
	for k := 0; k < NumClasses; k++ {
		for d := 0; d < DModel; d++ {
			dWClass[k][d] = dLogits[k] * c.pooled[d]
			dPooled[d] += dLogits[k] * m.WClass[k][d]
		}
	}

	// --- Pooling (moyenne) : le gradient est réparti également entre positions ---
	dContext := make([][]float64, n)
	for i := 0; i < n; i++ {
		dContext[i] = make([]float64, DModel)
		for d := 0; d < DModel; d++ {
			dContext[i][d] = dPooled[d] / float64(n)
		}
	}

	scale := math.Sqrt(float64(DModel))
	dQ := zeros(n, DModel)
	dK := zeros(n, DModel)
	dV := zeros(n, DModel)

	// --- Auto-attention : rétropropagation à travers le softmax ---
	for i := 0; i < n; i++ {
		dAttnRow := make([]float64, n)
		for j := 0; j < n; j++ {
			dAttnRow[j] = dot(dContext[i], c.V[j])
			for d := 0; d < DModel; d++ {
				dV[j][d] += c.attn[i][j] * dContext[i][d]
			}
		}

		// Jacobienne du softmax en ligne i :
		// dscore_j = a_j * (dAttn_j - somme_k a_k * dAttn_k)
		var weighted float64
		for j := 0; j < n; j++ {
			weighted += c.attn[i][j] * dAttnRow[j]
		}
		dScores := make([]float64, n)
		for j := 0; j < n; j++ {
			dScores[j] = c.attn[i][j] * (dAttnRow[j] - weighted)
		}

		for j := 0; j < n; j++ {
			for d := 0; d < DModel; d++ {
				dQ[i][d] += dScores[j] * c.K[j][d] / scale
				dK[j][d] += dScores[j] * c.Q[i][d] / scale
			}
		}
	}

	// --- Projections Q, K, V -> gradient vers Wq, Wk, Wv et vers les embeddings ---
	dWq := zeros(DModel, DModel)
	dWk := zeros(DModel, DModel)
	dWv := zeros(DModel, DModel)
	dX := zeros(n, DModel)

	for i := 0; i < n; i++ {
		for a := 0; a < DModel; a++ {
			for b := 0; b < DModel; b++ {
				dWq[a][b] += dQ[i][a] * c.x[i][b]
				dWk[a][b] += dK[i][a] * c.x[i][b]
				dWv[a][b] += dV[i][a] * c.x[i][b]
				dX[i][b] += m.Wq[a][b]*dQ[i][a] + m.Wk[a][b]*dK[i][a] + m.Wv[a][b]*dV[i][a]
			}
		}
	}

	// --- Descente de gradient (SGD) ---
	for k := 0; k < NumClasses; k++ {
		for d := 0; d < DModel; d++ {
			m.WClass[k][d] -= lr * dWClass[k][d]
		}
	}
	for a := 0; a < DModel; a++ {
		for b := 0; b < DModel; b++ {
			m.Wq[a][b] -= lr * dWq[a][b]
			m.Wk[a][b] -= lr * dWk[a][b]
			m.Wv[a][b] -= lr * dWv[a][b]
		}
	}
	for i, tok := range tokens {
		for d := 0; d < DModel; d++ {
			m.Embedding[tok][d] -= lr * dX[i][d]
		}
	}

	return loss
}

// serializedModel est la forme JSON persistée sur disque : les poids ET le
// vocabulaire, pour que l'inférence retombe toujours sur les mêmes indices.
type serializedModel struct {
	Vocab     map[string]int `json:"vocab"`
	Embedding [][]float64    `json:"embedding"`
	Wq        [][]float64    `json:"wq"`
	Wk        [][]float64    `json:"wk"`
	Wv        [][]float64    `json:"wv"`
	WClass    [][]float64    `json:"wclass"`
}

// Save sauvegarde les poids appris et le vocabulaire associé.
func (m *Model) Save(path string) error {
	s := serializedModel{
		Vocab:     m.Vocab.TokenToID,
		Embedding: m.Embedding,
		Wq:        m.Wq,
		Wk:        m.Wk,
		Wv:        m.Wv,
		WClass:    m.WClass,
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

// Load recharge un modèle entraîné (poids + vocabulaire) depuis le disque.
func Load(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s serializedModel
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &Model{
		Vocab:     &Vocab{TokenToID: s.Vocab},
		Embedding: s.Embedding,
		Wq:        s.Wq,
		Wk:        s.Wk,
		Wv:        s.Wv,
		WClass:    s.WClass,
	}, nil
}

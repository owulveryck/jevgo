# Jevgo — mini-transformer « from scratch » en Go : comprendre en construisant

Jevgo est un classifieur de code source (Go vs TypeScript) implémenté
« from scratch » en Go, sans framework de machine learning. L'objectif
n'est pas la performance, mais de rendre visible, ligne par ligne, ce qui
se passe dans un transformer minimal — de l'embedding jusqu'à la
rétropropagation.

Le dépôt est découpé en trois programmes qui partagent le même package :

```
model/       le transformer + le tokenizer (package model)
training/    entraîne le modèle, écrit weights/router.json
inference/   recharge les poids et classe du code
weights/     les poids sérialisés (JSON), produit de l'entraînement
```

> Le modèle n'a rien d'utile en production : c'est un objet d'étude.
> L'objectif est de comprendre ce que fait *vraiment* un transformer,
> en le construisant soi-même, ligne par ligne.

Ce README suit le découpage de la [documentation Diátaxis](https://diataxis.fr/) :
un **tutoriel** pour apprendre en faisant, des **how-to** pour résoudre des
problèmes précis, une **référence** pour consulter, et une **explication**
pour comprendre.

---

## Tutoriel : jouer avec le transformer

*Orienté apprentissage : suivez les étapes, vous ferez marcher le modèle
de bout en bout et vous verrez un transformer apprendre sous vos yeux.*

Prérequis : [Go](https://go.dev) ≥ 1.21 installé. C'est tout — aucune
dépendance externe.

**1. Entraînez le modèle.** Depuis la racine du dépôt :

```bash
go run ./training
```

Vous verrez le vocabulaire se construire, puis la perte diminuer époque
après époque :

```
Vocabulaire : 117 tokens
Époque   1 — perte moyenne : 0.6933
Époque  50 — perte moyenne : 0.0623
Époque 100 — perte moyenne : 0.0023
...
Poids et vocabulaire sauvegardés dans weights/router.json
```

La perte initiale 0.6933, c'est `-ln(2)` : le modèle n'y voit que du feu
et devine au hasard entre deux classes. À la fin, elle est proche de zéro :
le modèle s'est trompé sur quasiment aucun exemple.

**2. Classez du code.** Toujours depuis la racine :

```bash
go run ./inference
```

Le programme charge les poids fraîchement appris et classe deux extraits
de démonstration :

```
« func Divide(a, b float64) (float64, error) { » ...
-> Go (confiance 98.5%, distribution [Go: 0.985, TS: 0.015])

« export function divide(a: number, b: number): numb... » ...
-> TypeScript (confiance 100.0%, distribution [Go: 0.000, TS: 1.000])
```

**3. Testez avec votre propre code.** Lancez le mode interactif :

```bash
go run ./inference -stdin
```

Collez n'importe quel extrait (Go ou TypeScript), terminez par une ligne
contenant `###`, et regardez la décision :

```
Collez un extrait de code, terminez par une ligne '###' :
const x: number = 42;
###
« const x: number = 42; »
-> TypeScript (confiance 99.8%, distribution [Go: 0.002, TS: 0.998])
```

Amusez-y : des commentaires, du pseudocode, un mélange des deux langues…
Le modèle se trompe parfois, et c'est justement intéressant d'essayer de
comprendre *pourquoi* (un indice : l'[explication](#explication-comment-ça-marche)
plus bas).

**4. Cassez tout et recommencez.** Effacez `weights/router.json`,
relancez `go run ./training` : le modèle repart d'une initialisation
aléatoire (toujours identique grâce au seed 1337) et réapprend tout.
C'est la boucle complète de vie d'un modèle : entraîner, sérialiser,
charger, inférer.

---

## How-to : recettes

*Orienté tâche : vous savez déjà utiliser le dépôt, voici comment faire
quelque chose de précis.*

### Changer le dataset sans toucher au modèle

Le cas le plus simple : garder deux classes, mais changer la tâche (par
exemple « code testé » vs « code non testé », ou « Go » vs « Rust »).

1. Remplacez les exemples de `dataset` dans `training/main.go` par vos
   propres extraits.
2. Adaptez les noms `LabelGo` / `LabelTypeScript` dans `model/model.go`
   si les classes ne correspondent plus au domaine (ce ne sont que des
   constantes `0` et `1`).
3. Relancez `go run ./training` : le vocabulaire est reconstruit
   automatiquement à partir du nouveau corpus, et le modèle réapprend
   depuis zéro.

Rien d'autre à changer : le tokenizer est générique (il découpe du
texte structuré, pas spécifiquement du Go ou du TypeScript).

### Ajouter une troisième classe (ou plus)

Pour passer par exemple à trois langages (Go / TypeScript / Python) :

1. Dans `model/model.go`, passez `NumClasses = 3` et ajoutez une
   constante `LabelPython = 2`.
2. Dans `training/main.go`, ajoutez des exemples Python étiquetés avec
   `model.LabelPython`.
3. Rendez l'argmax de `Forward` générique : il est actuellement écrit
   pour deux classes (`if c.probs[1] > c.probs[0]`). Remplacez-le par
   un parcours de `probs` qui retourne l'indice du maximum.
4. Dans `inference/main.go`, la fonction `classify` affiche
   `[Go: %.3f, TS: %.3f]` en dur : affichez plutôt la distribution
   complète en bouclant sur `probs`.

Le reste de `model.go` est déjà générique : `WClass`, la boucle de
calcul des logits, le softmax et la rétropropagation de la tête de
classification (`for k := 0; k < NumClasses; k++`) s'adaptent
automatiquement au nombre de classes.

### Ajouter une deuxième tête d'attention (multi-head)

Le modèle actuel n'a qu'une seule tête d'attention. Pour en ajouter une
deuxième : dupliquez `Wq`, `Wk`, `Wv` par tête, calculez un contexte par
tête, puis concaténez les contextes avant le pooling. Cela demande
d'adapter le forward *et* le backward, puisque le gradient devra être
réparti entre les têtes. C'est une bonne étape suivante une fois le
fonctionnement à une tête bien compris — mais plus intrusive que les
deux recettes précédentes.

### Le modèle n'apprend pas : où regarder

- Vérifiez que la perte moyenne décroît dans les logs d'entraînement
  (`training/main.go`). Si elle stagne, essayez d'augmenter `lr` ou
  `epochs`.
- Assurez-vous que le dataset contient des exemples suffisamment
  discriminants pour chaque classe (des mots-clés qui n'apparaissent
  presque que dans l'une des deux).
- Rappelez-vous que ce modèle est volontairement minuscule (une seule
  tête, pas d'encodage positionnel) : il apprend des corrélations
  lexicales, pas une véritable compréhension syntaxique du code.

### Utiliser le package model dans votre propre programme

```go
m, err := model.Load("weights/router.json")
if err != nil { ... }

tokens := model.Tokenize("func main() {}")
class, probs := m.Forward(m.Vocab.Encode(tokens))
// class: 0 (Go) ou 1 (TypeScript), probs: distribution de probabilités
```

---

## Référence : architecture et API

*Orienté information : description sèche et exacte de ce qui existe.*

### Fichiers

| Chemin | Rôle |
|---|---|
| `model/model.go` | Le transformer : initialisation, passe avant, rétropropagation, sérialisation |
| `model/tokenizer.go` | Tokenization lexicale du code source + vocabulaire |
| `training/main.go` | Corpus d'entraînement et boucle SGD |
| `inference/main.go` | Chargement des poids et classification (CLI) |
| `weights/router.json` | Poids + vocabulaire (produit par `training`, consommé par `inference`) |

### Tokenization (`model/tokenizer.go`)

- `Tokenize(src string) []string` — découpe du code en tokens lexicaux :
  identifiants et mots-clés conservés intacts, chaînes → `STR`,
  nombres → `NUM`, opérateurs multi-caractères (`:=`, `=>`, `==`, `!=`,
  `<=`, `>=`, `&&`, `||`, `++`, `--`, `->`, `...`) reconnus d'un bloc.
- `Vocab` — dictionnaire token → indice, avec `UNK` (indice 0) comme
  repli pour tout token inconnu.
  - `NewVocab(corpus [][]string) *Vocab`
  - `(*Vocab).Size() int`
  - `(*Vocab).Encode(tokens []string) []int`

### Modèle (`model/model.go`)

Architecture : `Embedding → self-attention (Q, K, V) → pooling moyen →
classifieur linéaire → softmax`, avec `DModel = 16` et `NumClasses = 2`.

```go
func New(vocab *Vocab, seed int64) *Model
func (m *Model) Forward(tokens []int) (class int, probs []float64)
func (m *Model) TrainStep(tokens []int, label int, lr float64) float64
func (m *Model) Save(path string) error
func Load(path string) (*Model, error)
```

- `Forward` retourne la classe prédite (argmax) et la distribution complète.
- `TrainStep` fait un cycle complet forward → entropie croisée →
  rétropropagation → mise à jour SGD, et retourne la perte.
- Labels : `model.LabelGo = 0`, `model.LabelTypeScript = 1`.

### Format de `weights/router.json`

```json
{
  "vocab":     { "<token>": <indice>, ... },
  "embedding": [[float × 16] × vocabSize],
  "wq":        [[float × 16] × 16],
  "wk":        [[float × 16] × 16],
  "wv":        [[float × 16] × 16],
  "wclass":    [[float × 16] × 2]
}
```

Le vocabulaire fait partie du fichier : à l'inférence, les tokens sont
encodés avec *exactement* les indices vus à l'entraînement.

---

## Explication : comment ça marche ?

*Orienté compréhension : le pourquoi du comment, pour qui veut comprendre
ce que le code met en œuvre.*

### Comment fonctionne l'embedding

Un ordinateur ne manipule pas des mots-clés comme `func` ou `interface`,
il manipule des nombres. La première étape consiste donc à faire
correspondre chaque token à un indice entier (c'est le rôle de `Vocab`,
dans `tokenizer.go`), puis à faire correspondre chaque indice à un
vecteur de nombres réels — c'est la table `Embedding` :

```go
Embedding [][]float64 // [vocabSize][DModel]
```

`Embedding[tok]` est le vecteur (de dimension `DModel = 16`) associé au
token d'indice `tok`. Au départ, ces vecteurs sont complètement
aléatoires (voir `randMatrix` dans `New`) : le mot `func` n'a a priori
aucun rapport avec le mot `package`. C'est l'entraînement qui va, pas à
pas, déplacer ces vecteurs dans l'espace à 16 dimensions jusqu'à ce que
des tokens qui jouent un rôle similaire pour la tâche se retrouvent dans
des zones du même espace utiles à la classification finale.

Deux points importants :

- **Rien n'est appris à l'avance.** Contrairement à des embeddings
  pré-entraînés (Word2Vec, GloVe, ou les embeddings d'un LLM), ici les
  16 nombres qui représentent `func` n'ont de sens que pour cette tâche
  précise. Ce sont des paramètres du modèle comme les autres.
- **Le token inconnu (`UNK`, indice 0) partage un seul vecteur** pour
  tous les mots jamais vus à l'entraînement. C'est une limite assumée :
  un identifiant inédit (`myCustomFunc42`) sera traité comme n'importe
  quel autre mot inconnu ; seul le contexte syntaxique autour (les vrais
  mots-clés, eux, sont dans le vocabulaire) permet encore de classifier
  correctement.

### Comment fonctionne l'entraînement

L'entraînement (`training/main.go` + `Model.TrainStep`) répète, pour
chaque exemple du dataset et sur plusieurs époques, le cycle classique
d'un réseau de neurones :

1. **Passe avant (forward).** Le code est tokenisé, encodé en indices,
   puis traverse le modèle (embedding → attention → pooling →
   classification) pour produire une distribution de probabilités
   `[P(Go), P(TypeScript)]`.
2. **Calcul de la perte.** On compare cette distribution à la vraie
   étiquette avec l'entropie croisée : `loss = -log(P(bonne classe))`.
   Plus le modèle était confiant en se trompant, plus la perte est élevée.
3. **Rétropropagation (backward).** C'est la partie la plus dense du
   code. On calcule, poids par poids, dans quelle direction et de combien
   il faudrait le bouger pour réduire la perte. La chaîne remonte dans le
   sens inverse du forward : classification → pooling → attention
   (softmax inclus) → Q/K/V → embeddings. C'est la règle de dérivation
   en chaîne (*chain rule*), écrite ici à la main plutôt que déléguée à
   un moteur d'autodiff (PyTorch, JAX…).
4. **Mise à jour (SGD).** Chaque poids est ajusté dans la direction
   opposée à son gradient, proportionnellement au taux d'apprentissage
   `lr` : `poids -= lr * gradient`.

Ce cycle est répété `epochs = 300` fois sur l'intégralité du dataset. Le
log affiché (`Époque N — perte moyenne`) doit décroître régulièrement :
c'est le signe que le modèle apprend effectivement à séparer les deux
langages plutôt que de mémoriser au hasard.

**Pourquoi exemple par exemple, sans batch ni padding ?** Parce que
chaque extrait de code a une longueur différente, et que le mécanisme
d'attention n'exige pas de longueur fixe (contrairement à une couche
dense classique). Traiter un exemple à la fois évite d'avoir à gérer du
padding et des masques — c'est plus lent qu'un vrai entraînement par
batch, mais beaucoup plus simple à lire.

### Comment fonctionne l'inférence

L'inférence (`inference/main.go` + `Model.Forward`) réutilise exactement
le même chemin que la passe avant de l'entraînement, mais sans calcul de
perte ni de gradient :

1. `model.Tokenize(code)` découpe le texte en tokens.
2. `m.Vocab.Encode(tokens)` convertit ces tokens en indices, en retombant
   sur `UNK` pour tout mot absent du vocabulaire appris.
3. `m.Forward(ids)` fait traverser ces indices au modèle avec les poids
   figés (ceux sauvegardés dans `weights/router.json` par
   l'entraînement) et renvoie la classe la plus probable.

C'est pour cela que `Save`/`Load` sérialisent à la fois les poids et le
vocabulaire : sans le même vocabulaire, un même mot pourrait se voir
attribuer un indice différent entre l'entraînement et l'inférence, et le
modèle deviendrait incohérent.

### Ce que fait l'attention, concrètement

Pour chaque token, le modèle calcule trois vecteurs : une *query* (ce
que je cherche), une *key* (ce que je propose) et une *value* (ce que je
transmets). Le produit scalaire query·key mesure la pertinence de chaque
token pour chaque autre ; un softmax transforme ces scores en poids
d'attention ; chaque token reçoit alors une moyenne pondérée des values.
Concrètement, un token `:` situé près d'un `number` capte un signal
« TypeScript », un token `:=` capte un signal « Go » : l'attention
laisse les tokens se transmettre leurs indices syntaxiques respectifs.
Le pooling moyen condense ensuite l'ensemble en un seul vecteur, que la
tête de classification projette sur deux logits, transformés en
probabilités par un softmax.

### Limites connues (et instructives)

- **Invariance par permutation.** Faute d'encodage positionnel, permuter
  les tokens d'une même séquence ne change *rien* à la décision : le
  pooling moyen et l'attention sans position ne voient que des sacs de
  tokens. Sur cette tâche, c'est suffisant ; sur de la compréhension de
  code, ce serait rédhibitoire.
- **Un seul bloc, une seule tête.** Les vrais transformers empilent des
  dizaines de blocs à plusieurs têtes. Ici, une seule passe d'attention
  suffit parce que la tâche ne demande qu'un niveau de raisonnement.
- **Sur-apprentissage garanti.** Le corpus fait quelques dizaines
  d'exemples ; le modèle finit par les mémoriser parfaitement. La perte
  d'entraînement à 0.0003 ne dit rien de sa généralisation réelle —
  une bonne leçon sur le sens (et les limites) de cette métrique.

Autrement dit : ce dépôt n'enseigne pas comment construire un modèle
utile, il enseigne comment fonctionne la machine. Le reste — échelle,
données, régularisation — est une histoire d'ingénierie qui n'a de sens
qu'une fois la machine comprise.

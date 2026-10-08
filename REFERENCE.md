# Référence — paramètres, sorties et fichiers de Jev

Document de référence (au sens Diátaxis) : description neutre et complète
des paramètres, des sorties et des formats de fichiers. Pour apprendre en
faisant, voir [`TUTORIAL.md`](TUTORIAL.md) ; pour la vue d'ensemble, voir
[`README.md`](README.md).

Toutes les valeurs de cette page sont celles du code, aux emplacements
indiqués. Les chemins sont relatifs à la racine du dépôt.

---

## 1. Paramètres d'entraînement

Déclarés en constantes dans `training/main.go`.

| Constante | Valeur | Type | Rôle |
|---|---|---|---|
| `lr` | `0.02` | `float64` | Taux d'apprentissage (*learning rate*) de la descente de gradient stochastique. |
| `epochs` | `400` | `int` | Nombre de passes complètes sur le dataset d'entraînement. |
| `attnWeightsPath` | `weights/policy-attention.json` | `string` | Fichier de sortie du transformer. |
| `baseWeightsPath` | `weights/policy-baseline.json` | `string` | Fichier de sortie du sac d'embeddings (sans attention). |
| *(seed, en dur)* | `1337` | `int64` | Graine d'initialisation des poids des **deux** modèles. |

### 1.1 `lr` — taux d'apprentissage

`poids -= lr * gradient` à chaque exemple. `lr` contrôle l'amplitude du
pas de descente.

- **Trop grand** : la mise à jour dépasse le minimum, la perte oscille ou
  diverge. Constaté dans ce dépôt : avec `lr = 0.1` ou `0.2`, le modèle
  converge vers une solution qui prédit DENY sur presque toute lecture
  `internal` et **oublie la règle 5** (les 8 cas `contractor` + `read` +
  `internal` de même département, pourtant ALLOW, sont mal classés).
- **Trop petit** : l'entraînement est lent et peut ne pas avoir convergé
  dans le nombre d'époques imparti.
- **Valeur par défaut** `0.02` : 100 % d'exactitude sur le dataset
  d'entraînement à 400 époques, tout en généralisant sur les cas-faille.

### 1.2 `epochs` — nombre d'époques

Une époque = un parcours de tout le dataset, exemple par exemple (pas de
batch). La perte est donc calculée `epochs × len(train)` fois.

- Avec la configuration par défaut (`lr = 0.02`), les points de
  convergence observés sont :

  | Époque | Exactitude train | Règle 5 (ALLOW) | Faille généralisée (DENY) |
  |---|---|---|---|
  | 100 | 96,5 % | 8/8 | 0/24 |
  | 300 | 97,8 % | 8/8 | 0/24 |
  | 400 | 100 % | 8/8 | 11/24 |
  | 800 | 100 % | 8/8 | 11/24 |

  Autrement dit : 400 époques suffisent, au-delà la généralisation ne
  progresse plus. Augmenter `epochs` sans baisser `lr` ne garantit pas
  une meilleure généralisation.

### 1.3 `seed` — graine aléatoire

Passée à `model.New(vocab, 1337)` et `policy.NewBagOfEmbeddings(vocab,
model.DModel, 1337)`. Elle initialise `rand.NewSource(seed)` et sert
uniquement à tirer les poids de départ (`randMatrix`, loi uniforme dans
`[-0.1, 0.1]`).

- **Reproductibilité** : même `seed` + même dataset + même ordre
  d'itération ⇒ mêmes poids finaux. L'ordre d'itération est déterministe
  (celui de `policy.Split`).
- **Effet** : une graine différente change le point de départ, donc la
  solution atteinte. Mesuré sur les graines 1 à 40, à hyperparamètres
  identiques : de 0 à 21 cas-faille refusés sur 24, médiane 9 ; `1337` en
  donne 11. La graine 6 refuse 24/24 mais rate la règle 5 (2/8), et 7
  graines sur 40 n'atteignent pas 100 % d'exactitude sur l'entraînement.
  Avec `1337`, les P(DENY) sur les 24 cas-faille sont tranchées : 10 au-dessus
  de 95 %, 11 sous 2 %.

### 1.4 Ordre d'entraînement

L'entraînement parcourt les exemples dans l'ordre de `train`
(`policy.Split`), pour chaque époque. Il n'y a **ni mélange** (shuffle)
**ni batch** : chaque `TrainStep` met à jour les poids immédiatement
(descente de gradient stochastique pure).

---

## 2. Hyperparamètres du modèle

Déclarés dans `model/model.go`.

| Constante | Valeur | Rôle |
|---|---|---|
| `DModel` | `16` | Dimension latente de chaque embedding et des vecteurs Q/K/V/contexte. |
| `NumClasses` | `2` | Nombre de classes de sortie. |
| `LabelDeny` | `0` | Indice de classe « refusé ». |
| `LabelAllow` | `1` | Indice de classe « autorisé ». |

Le facteur d'échelle de l'attention vaut `math.Sqrt(DModel)` ≈ `4`, il
n'est pas un paramètre réglable.

### 2.1 Formes des tenseurs

| Poids | Dimensions |
|---|---|
| `Embedding` | `[vocabSize][DModel]` |
| `Wq`, `Wk`, `Wv` | `[DModel][DModel]` |
| `WClass` | `[NumClasses][DModel]` |

`NumClasses` et `WClass` sont génériques : augmenter `NumClasses` suffit
pour ajouter des classes (voir `TUTORIAL.md` §5).

### 2.2 Vocabulaire

Construit par `model.NewVocab` sur les tokens du dataset d'entraînement.

| Élément | Valeur |
|---|---|
| Indice de `UNK` | `0` (toujours) |
| Taille observée | `21` tokens (`UNK` + 20 tokens `champ=valeur`) |
| Repli | `Vocab.Encode` associe `UNK` (0) à tout token inconnu |

Les 20 tokens proviennent des domaines de `policy/request.go` : 4 rôles +
3 actions + 3 classifications + 4 `resource_dept` + 4 `requester_dept` +
2 valeurs `mfa`, chaque token étant préfixé par son champ.

### 2.3 Sac d'embeddings (référence sans attention)

`policy/baseline.go` : mêmes `dim = DModel = 16` et `numClasses = 2`, mais
un seul jeu d'embeddings et une couche linéaire, sans attention. Il sert
de point de comparaison, pas de modèle de production.

---

## 3. Sortie de l'entraînement

Format d'une ligne (affichée aux époques 1, 5, 10, … — tous les 5 à partir
de l'époque 5) :

```
Époque  N — perte attention : X.XXXX — perte sans attention : Y.YYYY
```

| Champ | Signification |
|---|---|
| `N` | Numéro d'époque (1-indexé). |
| `perte attention` | Entropie croisée moyenne du transformer sur le dataset d'entraînement. |
| `perte sans attention` | Même mesure pour le sac d'embeddings. |

La perte est `-log(P(bonne classe))` moyennée sur les exemples. Repères :

| Valeur | Interprétation |
|---|---|
| `≈ 0.693` = `ln 2` | modèle équivalent à une pièce équilibrée (aucun apprentissage). |
| `≈ 0.1` | modèle presque toujours correct mais peu confiant. |
| `≈ 0.000` | modèle correct et très confiant sur le dataset d'entraînement. |

Une perte qui **ne décroît pas** signale un problème d'hyperparamètres
(`lr` trop grand ou trop petit) ou un dataset non discriminant.

Lignes finales :

```
Poids sauvegardés dans weights/policy-attention.json et weights/policy-baseline.json
Interrogez-les avec : go run ./ask -demo
```

---

## 4. Paramètres d'inférence — commande `ask`

Déclarés par `flag` dans `ask/main.go`. Tous optionnels, mais une requête
complète exige les cinq champs de texte (sauf `-demo`).

| Drapeau | Type | Défaut | Effet |
|---|---|---|---|
| `-demo` | `bool` | `false` | Joue 5 requêtes pré-choisies et affiche un tableau comparatif. Ignore les autres drapeaux. |
| `-role` | `string` | `""` | Rôle du demandeur : `admin`, `employee`, `contractor`, `guest`. |
| `-action` | `string` | `""` | Action : `read`, `write`, `delete`. |
| `-classification` | `string` | `""` | Classification du document : `public`, `internal`, `confidential`. |
| `-resource-dept` | `string` | `""` | Département du document : `engineering`, `hr`, `finance`, `marketing`. |
| `-requester-dept` | `string` | `""` | Département du demandeur (mêmes valeurs). |
| `-mfa` | `bool` | `false` | MFA validée. Accepte `-mfa` (vrai) ou `-mfa=false`. |

Règles de fonctionnement :

- Si `-demo` est absent et qu'un des cinq champs texte est vide :
  message d'usage et code de sortie `2`.
- Chaque champ texte est **comparé** aux listes `policy.*`. Une valeur
  hors liste n'arrête pas le programme : elle est signalée comme « hors
  vocabulaire » et le modèle l'encode en `UNK`.
- Si un fichier de poids est absent ou illisible : sortie en erreur avec
  le rappel `(lancez d'abord 'go run ./training')`.

---

## 5. Sortie de l'inférence

### 5.1 Requête unique

```
Requête : role action classification resource_dept/requester_dept mfa=<bool>

Résultat
  Rego (règle littérale)        : ALLOW|DENY
  Jev (avec attention)          : ALLOW|DENY (<confiance> %)
  Sac de mots (sans attention)  : ALLOW|DENY (<confiance> %)
```

| Champ | Signification |
|---|---|
| `Rego (règle littérale)` | Décision du moteur OPA (`policy.Evaluator.Eval`) sur `policy.rego`. Aucune probabilité. |
| `Jev (avec attention)` | `model.Forward` : classe prédite et confiance associée. |
| `Sac de mots (sans attention)` | `BagOfEmbeddings.Forward`, même présentation. |
| `confiance` | Probabilité de la classe prédite, en pourcentage (`probs[classe] × 100`). |

La classe prédite est l'`argmax` des probabilités. En binaire, cela
équivaut au seuil `0.5` : `ALLOW` si `P(ALLOW) > P(DENY)`, sinon `DENY`.

Si `Jev` et `Rego` diffèrent, une ligne `⚠ Le modèle avec attention
diverge de Rego.` est affichée, suivie d'une section `Analyse` choisie
selon le cas : requête-faille, valeur hors vocabulaire, divergence
ordinaire, ou accord.

### 5.2 Mode `-demo`

Tableau `tabwriter` à cinq colonnes :

| Colonne | Contenu |
|---|---|
| `REQUETE` | `role action classification resource_dept/requester_dept mfa=<bool>`. |
| `REGO` | Décision littérale. |
| `JEV (attention)` | Prédiction du transformer + confiance. |
| `SAC DE MOTS` | Prédiction du sac d'embeddings + confiance. |
| `NOTE` | `faille regle 5`, `hors vocabulaire: <champ>=<valeur>`, ou `-`. |

Suivent un compteur de divergences du transformer par rapport à Rego, et
le rappel du nombre de requêtes hors vocabulaire.

---

## 6. Datasets et fichiers

### 6.1 Tailles

| Ensemble | Construit par | Taille |
|---|---|---|
| Dataset complet | `policy.All()` | `1152` requêtes (`4 × 3 × 3 × 4 × 4 × 2`). |
| Entraînement | `policy.Split()` (1ʳᵉ valeur) | `1128` requêtes. |
| Cas-faille | `policy.Split()` (2ᵉ valeur) | `24` requêtes. |

Les 24 cas-faille sont les requêtes `contractor` + `read` + `internal`
avec `resource_dept ≠ requester_dept` ; elles déclenchent la règle 5 mais
sont retirées de l'entraînement.

### 6.2 Décision de l'oracle

La politique est exécutée par le moteur OPA (SDK Go
`github.com/open-policy-agent/opa/rego`), via `policy.Evaluator` :

| Élément | Détail |
|---|---|
| Fichier politique | `policy/policy.rego`, embarqué dans le binaire (`go:embed`). |
| Syntaxe | Rego **v1** (`allow if { … }`, `default allow := false`). Le SDK est configuré avec `rego.SetRegoVersion(ast.RegoV1)`. |
| Requête | `data.access.allow`. |
| Compilation | `policy.NewEvaluator(ctx)` appelle `PrepareForEval` une fois. |
| Évaluation | `Evaluator.Eval(ctx, req)` construit l'input JSON puis appelle `Eval` ; renvoie un `bool` (règles 1 à 5, défaut DENY). |
| Champs d'entrée | `role`, `action`, `resource_classification`, `resource_department`, `requester_department`, `mfa`. |

Aucune logique de politique n'est dupliquée en Go : `policy.rego` est
l'unique source de vérité.

### 6.3 Fichiers de poids (JSON)

Transformer — `weights/policy-attention.json`, écrit par `Model.Save` :

| Clé | Contenu |
|---|---|
| `vocab` | `map[string]int` : token → indice. |
| `embedding` | `[][]float64` `[vocabSize][DModel]`. |
| `wq`, `wk`, `wv` | `[][]float64` `[DModel][DModel]`. |
| `wclass` | `[][]float64` `[NumClasses][DModel]`. |

Sac d'embeddings — `weights/policy-baseline.json`, écrit par
`BagOfEmbeddings.Save` :

| Clé | Contenu |
|---|---|
| `vocab` | `map[string]int`. |
| `embedding` | `[][]float64` `[vocabSize][dim]`. |
| `wclass` | `[][]float64` `[numClasses][dim]`. |
| `dim` | `int` (= `DModel`). |

Le vocabulaire est sauvegardé avec les poids : sans lui, un même token
pourrait recevoir un indice différent à l'inférence. Les fichiers sont
régénérés par `go run ./training` et ignorés par Git
(`.gitignore` : `weights/*.json`).

---

## 7. Codes de sortie

| Code | Commande | Cause |
|---|---|---|
| `0` | `training`, `ask` | Succès. |
| `1` | `training`, `ask` | Échec de sauvegarde/chargement des poids (`log.Fatalf`). |
| `2` | `ask` | Champs de requête manquants (hors `-demo`). |
| `2` | `ask` | Drapeau invalide (par ex. `-mfa` sans valeur booléenne), géré par `flag`. |
| `1` | `training`, `ask` | Échec de compilation ou d'évaluation de `policy.rego` par OPA (`log.Fatalf`). |

---

## 8. Dépendances

| Dépendance | Rôle | Portée |
|---|---|---|
| `github.com/open-policy-agent/opa` | Moteur Rego : compilation et évaluation de `policy.rego` comme oracle. | `policy` (import `opa/rego`, `opa/ast`), utilisée par `training` et `ask`. |

Le reste du dépôt (`model`, `policy` hors OPA, `training`, `ask`) n'utilise
que la bibliothèque standard. Le module est déclaré dans `go.mod` et figé
par `go.sum`.

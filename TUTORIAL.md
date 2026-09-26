# Tutoriel — comprendre et étendre le mini-transformer Jev

Ce document accompagne le code de Jev : un classifieur qui apprend à
imiter une politique de contrôle d'accès écrite en Rego, implémenté
« from scratch » en Go, sans framework de machine learning. L'objectif
n'est pas la performance, mais de rendre visible, ligne par ligne, ce qui
se passe dans un transformer minimal — de l'embedding jusqu'à la
rétropropagation.

Le dépôt se lit dans cet ordre :

```
model/       le transformer (model.go) + le vocabulaire (tokenizer.go)
policy/      le domaine : Request, l'oracle Decide, le sac de mots baseline
training/    l'entraînement des deux modèles
ask/         le CLI d'inférence et de comparaison à Rego
```

---

## 1. Comment fonctionne l'embedding

Un ordinateur ne manipule pas des chaînes comme `role=contractor` ou
`action=read`, il manipule des nombres. La première étape consiste donc à
faire correspondre chaque token à un indice entier (c'est le rôle de
`Vocab`, dans `model/tokenizer.go`), puis à faire correspondre chaque
indice à un vecteur de nombres réels — c'est la table `Embedding` :

```go
Embedding [][]float64 // [vocabSize][DModel]
```

`Embedding[tok]` est le vecteur (de dimension `DModel = 16`) associé au
token d'indice `tok`. Au départ, ces vecteurs sont complètement
aléatoires (voir `randMatrix` dans `model.New`) : `role=contractor` n'a a
priori aucun rapport avec `action=read`. C'est l'entraînement qui va, pas
à pas, déplacer ces vecteurs dans l'espace à 16 dimensions jusqu'à ce que
des tokens qui jouent un rôle similaire pour la tâche se retrouvent dans
des zones du même espace utiles à la classification finale.

Dans le cas de la politique d'accès, les tokens ne sont pas des mots d'un
texte : `policy.Request.Tokens()` sérialise chaque requête en six tokens
« champ=valeur » :

```
role=contractor  action=read  classification=internal
resource_dept=hr  requester_dept=marketing  mfa=false
```

Chaque token est une **unité opaque** : `resource_dept=hr` et
`requester_dept=hr` sont deux tokens distincts, avec deux embeddings
indépendants. C'est délibéré : le modèle ne peut pas détecter une
correspondance de département en comptant simplement des occurrences
partagées entre champs, il doit apprendre une vraie relation entre les
deux positions (via l'attention, voir §2).

Deux points importants :

- **Rien n'est appris à l'avance** : contrairement à des embeddings
  pré-entraînés (Word2Vec, GloVe, ou les embeddings d'un LLM), ici les 16
  nombres qui représentent `role=admin` n'ont de sens que pour cette
  tâche précise. Ce sont des paramètres du modèle comme les autres.
- **Le token inconnu (`UNK`, indice 0) partage un seul vecteur** pour
  toutes les valeurs jamais vues à l'entraînement. C'est une limite
  assumée : un rôle inédit comme `intern` (absent des domaines de
  `policy/request.go`) sera traité comme UNK. Le CLI `ask` le signale,
  voir §4.

---

## 2. Comment fonctionne l'entraînement

L'entraînement (`training/main.go` + `Model.TrainStep`) répète, pour
chaque requête du dataset et sur plusieurs époques, le cycle classique
d'un réseau de neurones :

1. **Passe avant (forward)** : la requête est sérialisée en tokens
   (`Request.Tokens()`), encodée en indices, puis traverse le modèle
   (embedding → self-attention → pooling → classification) pour produire
   une distribution de probabilités `[P(DENY), P(ALLOW)]`.
2. **Calcul de la perte** : on compare cette distribution à la vraie
   étiquette avec l'entropie croisée — `loss = -log(P(bonne classe))`.
   Plus le modèle était confiant et se trompait, plus la perte est
   élevée.
3. **Rétropropagation (backward)** : on calcule, poids par poids, dans
   quelle direction et de combien il faudrait le bouger pour réduire la
   perte. La chaîne va dans le sens inverse du forward : classification →
   pooling → attention (softmax inclus) → Q/K/V → embeddings. C'est la
   règle de dérivation en chaîne (*chain rule*), écrite ici à la main
   plutôt que déléguée à un moteur d'autodiff (PyTorch, JAX...).
4. **Mise à jour (SGD)** : chaque poids est ajusté dans la direction
   opposée à son gradient, proportionnellement au taux d'apprentissage
   `lr` : `poids -= lr * gradient`.

Ce cycle est répété `epochs = 400` fois sur l'intégralité du dataset, avec
`lr = 0.02`. Le log affiché (Époque N — perte moyenne) doit décroître
régulièrement : c'est le signe que le modèle apprend effectivement à
séparer ALLOW de DENY plutôt que de mémoriser au hasard.

**Les étiquettes viennent de Rego.** Pour chaque requête, l'étiquette est
`req.Decide()` — la traduction fidèle de `policy.rego` en Go. Personne
n'annote à la main : c'est l'oracle qui fabrique le dataset supervisé
(voir §4).

**Deux modèles, un même dataset.** `training/main.go` entraîne en
parallèle le transformer (`model.New`) et un sac d'embeddings sans
attention (`policy.NewBagOfEmbeddings`), afin de comparer ce que
l'attention apporte. Le sac de mots moyenne les embeddings des six tokens
puis applique une couche linéaire : aucun mélange entre positions.

**Pourquoi exemple par exemple, sans batch ?** Le jeu de données est ici
de taille fixe (six tokens par requête), mais on traite volontairement un
exemple à la fois pour garder le code lisible et rétropropager
exactement comme dans un vrai entraînement. C'est plus lent qu'un
entraînement par batch, mais beaucoup plus simple à suivre.

---

## 3. Comment fonctionne l'inférence

L'inférence (`ask/main.go` + `Model.Forward`) réutilise exactement le
même chemin que la passe avant de l'entraînement, mais sans calcul de
perte ni de gradient :

1. `req.Tokens()` sérialise la requête en tokens.
2. `m.Vocab.Encode(tokens)` convertit ces tokens en indices, en retombant
   sur UNK pour toute valeur absente du vocabulaire appris.
3. `m.Forward(ids)` fait traverser ces indices au modèle avec les poids
   figés (ceux sauvegardés dans `weights/policy-attention.json` par
   l'entraînement) et renvoie la classe la plus probable.

C'est pour cela que `Save`/`Load` sérialisent à la fois les poids **et le
vocabulaire** : sans le même vocabulaire, un même token pourrait se voir
attribuer un indice différent entre l'entraînement et l'inférence, et le
modèle deviendrait incohérent.

---

## 4. Le cas d'usage « politique d'accès » : quelle question, quel rôle pour Rego

Cette section répond à une confusion fréquente : on ne classe pas des
employés, et Rego n'est pas appelé par le modèle.

### Quelle est la question posée ?

Chaque exemple n'est pas une personne mais une **requête d'accès** : un
sextuplet `(rôle, action, classification du document, département du
document, département du demandeur, MFA)`. La question posée au modèle
est binaire, sur cette requête : serait-elle autorisée (ALLOW) ou refusée
(DENY) par la politique ?

Le modèle apprend donc une fonction `f(requête) → {ALLOW, DENY}` à partir
d'exemples de requêtes déjà tranchées — pas un jugement sur des
personnes, mais l'imitation d'une fonction de décision.

### Le rôle de Rego pendant l'entraînement : l'oracle qui étiquette

En apprentissage supervisé, il faut des paires (entrée, étiquette).
Personne n'a annoté 1128 requêtes à la main : c'est le moteur de règles —
`policy.rego`, reproduit par `Request.Decide()` en Go — qui joue ce rôle.
Pour chaque requête générée par `policy.All()`, on lui demande
« ALLOW ou DENY ? », et cette réponse devient l'étiquette d'entraînement
utilisée par `TrainStep`.

Rego n'est donc **jamais appelé par le modèle**, ni pendant
l'entraînement ni pendant l'inférence : il a servi une seule fois, en
amont, à fabriquer le dataset étiqueté. C'est la même relation qu'entre
un correcteur qui note des copies et un élève qui essaie ensuite de
deviner la note sans le correcteur — sauf qu'ici le correcteur est
déterministe et peut noter l'intégralité des copies possibles (1152
combinaisons), pas juste un échantillon.

### Le rôle de Rego à l'inférence : la référence à laquelle on compare

Les 24 requêtes-faille (`policy.Split`, deuxième valeur de retour) n'ont
jamais été montrées au modèle à l'entraînement. Pour savoir ce qu'aurait
dû répondre le modèle, le CLI redemande à Rego (`Decide()`) — sur ce
sous-ensemble précis, la réponse est toujours ALLOW, par construction
(elles ont été sélectionnées justement parce qu'elles déclenchent la
règle 5).

Une précision importante : ceci **n'est pas** une mesure d'exactitude
classique sur un échantillon aléatoire représentatif — c'est une sonde
qualitative ciblée sur un cas limite choisi à l'avance. Pour mesurer
l'exactitude générale du modèle (a-t-il bien appris la règle sur des cas
ordinaires, pas seulement sur la faille), il faudrait mettre de côté un
second échantillon, aléatoire cette fois, pris dans le reste du dataset —
ce que le code actuel ne fait pas encore. C'est une amélioration simple à
ajouter si vous voulez un vrai chiffre d'exactitude en plus de
l'observation qualitative sur la faille.

### Le CLI : visualiser la généralisation là où Rego a des limites

`go run ./ask -demo` affiche un tableau qui confronte, sur cinq requêtes
choisies, la réponse littérale de Rego et les prédictions des deux
modèles. C'est le point pédagogique du dépôt : **Rego exécute, le modèle
généralise**. Voici la sortie typique (après `go run ./training`) :

```
REQUETE                                                 REGO   JEV (attention)  SAC DE MOTS    NOTE
employee read public engineering/engineering mfa=false  ALLOW  ALLOW (100.0%)   ALLOW (98.4%)  -
guest delete confidential finance/marketing mfa=false   DENY   DENY (100.0%)    DENY (100.0%)  -
contractor read internal hr/hr mfa=false                ALLOW  ALLOW (100.0%)   DENY (86.1%)   -
contractor read internal hr/marketing mfa=false         ALLOW  DENY (95.2%)     DENY (85.8%)   faille regle 5
intern read confidential hr/engineering mfa=false       DENY   DENY (100.0%)    ALLOW (85.6%)  hors vocabulaire: role=intern
```

Ce que chaque ligne raconte :

- **Lignes 1-2** — des cas ordinaires : Rego et le transformer
  s'accordent. C'est le socle : le modèle a bien appris la politique.
- **Ligne 3** — `contractor` lit un document interne de **son**
  département. La règle 5 est présente à l'entraînement (seules les
  requêtes à département *différent* ont été mises de côté), et le
  transformer l'apprend (ALLOW). Le sac de mots, qui ne mélange pas les
  positions, ne capte pas cette combinaison et se trompe (DENY) — c'est
  là que l'attention se justifie.
- **Ligne 4** — la même requête, mais lue dans un **autre** département.
  Ces 24 cas ont été retirés du dataset. Rego applique mécaniquement sa
  règle 5 et répond ALLOW ; le transformer, lui, a vu que les règles 2
  et 3 exigent la correspondance de département, et généralise : DENY.
  **C'est la réponse que Rego ne sait pas donner** : une politique
  écrite en règles ne peut pas signaler que sa propre règle 5 est un cas
  limite suspect. Le modèle statistique, lui, « doute » et rend ce doute
  visible sous forme de probabilité.
- **Ligne 5** — un rôle inconnu (`intern`) : Rego retombe sur son cas par
  défaut (`default allow = false`), le modèle encode le token en UNK et
  continue de raisonner sur les autres champs. Les deux ont leurs
  limites, différentes.

Pour le détail et les probabilités d'une requête précise :

```bash
go run ./ask -role=contractor -action=read -classification=internal \
             -resource-dept=hr -requester-dept=marketing
```

```
Requête : contractor read internal hr/marketing mfa=false

Résultat
  Rego (règle littérale)        : ALLOW
  Jev (avec attention)          : DENY (95.2%)
  Sac de mots (sans attention)  : DENY (85.8%)

  ⚠ Le modèle avec attention diverge de Rego.

Analyse
  Requête-faille : la règle 5 autorise ce contractor à lire un document
  interne d'un autre département, sans vérifier la correspondance — c'est
  le cas volontairement retiré de l'entraînement.
  Rego applique la règle telle qu'écrite et répond ALLOW. Le transformer,
  lui, a appris la règle 5 sur les cas même-département restés dans le
  dataset, et a vu par ailleurs que les règles 2 et 3 exigent la
  correspondance de département : il généralise et répond DENY. C'est
  exactement ce que Rego ne sait pas faire — douter de sa propre règle.
```

Le CLI affiche aussi un avertissement lorsqu'un champ est hors vocabulaire
(il ne peut pas savoir s'il s'agit d'une valeur métier légitime ou d'une
faute de frappe) : dans les deux cas, le token devient UNK côté modèle.

---

## 5. Howto — ajouter ou changer un classificateur

### Changer le dataset sans toucher au modèle

Le cas le plus simple : garder deux classes, mais changer la politique
ou le domaine. Il suffit de modifier le package `policy` :

1. Ajuster les listes de valeurs dans `policy/request.go` (`Roles`,
   `Actions`, `Classifications`, `Departments`).
2. Mettre à jour `Request.Decide()` **et** `policy/policy.rego` en
   parallèle (les deux doivent rester synchronisés).
3. Ajuster si besoin `IsLoophole()` et `Request.Tokens()`.
4. Relancer `go run ./training`, qui reconstruira le vocabulaire et
   réentraînera depuis zéro.

Rien d'autre à changer : `policy.All()` génère le dataset par énumération
des listes, et le vocabulaire est reconstruit automatiquement à partir du
nouveau corpus.

### Ajouter une troisième classe (ou plus)

Pour passer par exemple à trois issues (ALLOW / DENY / REVIEW) :

1. Dans `model/model.go`, changer `NumClasses = 2` en `NumClasses = 3` et
   ajouter une constante d'étiquette (`LabelReview = 2`) à côté de
   `LabelDeny` / `LabelAllow`.
2. Dans `training/main.go`, produire l'étiquette correspondante pour
   chaque requête (l'oracle `Decide()` devient une fonction
   `→ {deny, allow, review}` plutôt qu'un booléen).
3. Aucune autre modification n'est nécessaire dans `model.go` : `WClass`,
   la boucle de calcul des logits, le softmax et la boucle de
   rétropropagation de la tête de classification
   (`for k := 0; k < NumClasses; k++`) sont tous génériques et s'adaptent
   automatiquement au nombre de classes.
4. Dans `ask/main.go`, la fonction `label` gère actuellement deux cas en
   dur — il faudra l'étendre pour afficher les trois issues.

### Ajouter une deuxième tête d'attention (multi-head)

Le modèle actuel n'a qu'une seule tête d'attention. Pour en ajouter une
deuxième, l'idée est de dupliquer `Wq`, `Wk`, `Wv` par tête, calculer un
contexte par tête, puis concaténer les contextes avant le pooling — ce
qui demande d'adapter le forward et le backward en conséquence, puisque
le gradient devra être réparti entre les têtes. C'est une bonne étape
suivante une fois le fonctionnement à une tête bien compris, mais plus
intrusive que les deux cas précédents.

### Où regarder si le modèle n'apprend pas

- Vérifier que la perte moyenne décroît dans les logs d'entraînement
  (`training/main.go`). Si elle stagne, essayer d'augmenter `lr` ou
  `epochs`.
- Attention à l'**équilibre des classes** : la règle 5 ne concerne qu'une
  poignée de requêtes. Avec `lr = 0.1` ou `0.2`, l'entraînement peut
  converger vers un modèle qui prédit DENY partout pour les lectures
  `internal` et **oublier la règle 5**. Avec `lr = 0.02` et 400 époques,
  la configuration par défaut du dépôt atteint 100 % sur le dataset
  d'entraînement tout en généralisant sur les cas-faille.
- S'assurer que chaque classe a des exemples suffisamment discriminants.
- Se rappeler que ce modèle est volontairement minuscule (une seule tête,
  pas d'encodage positionnel) : il apprend des corrélations lexicales,
  pas une véritable compréhension de la politique. Le CLI `ask` sert
  précisément à voir, sans se raconter d'histoires, ce qu'il généralise
  et ce qu'il rate.

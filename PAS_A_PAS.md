# Pas à pas — comment le dataset est fabriqué et comment le modèle apprend

Ce document explique, sans prérequis, **d'où viennent les exemples
d'entraînement** et **ce qui se passe pendant l'apprentissage**. L'objectif
est que chacun comprenne ce que la démo raconte et pourquoi un modèle du
type Jev est intéressant pour de la classification.

Pour la version technique détaillée, voir [`TUTORIAL.md`](TUTORIAL.md) ;
pour les paramètres exacts, [`REFERENCE.md`](REFERENCE.md).

---

## En 30 secondes

On veut un modèle qui répond **ALLOW** (autoriser) ou **DENY** (refuser) à
une demande d'accès — exactement comme la politique Rego. On va lui montrer
des milliers de demandes déjà tranchées par Rego, puis lui apprendre à
deviner la réponse tout seul. Ensuite, on lui pose des demandes que Rego
n'a **jamais** vues à l'entraînement, et on regarde s'il a bien « compris »
ou s'il se contente de réciter.

---

## L'analogie

Imaginez un professeur qui possède un **règlement très précis** pour
autoriser ou refuser des accès. Il peut corriger un nombre illimité de
copies en un temps record. Un élève (le modèle) veut apprendre à noter
**sans le règlement**, uniquement en observant le professeur corriger.

- Le **règlement** = `policy/policy.rego`.
- Le **professeur** = le moteur OPA, qui applique le règlement.
- L'**élève** = le mini-transformer de `model/`.
- Les **copies** = des demandes d'accès.
- La **note du professeur** = l'étiquette ALLOW/DENY.

---

## Étape 1 — La politique, écrite en Rego

`policy/policy.rego` contient 5 règles (`policy/policy.rego`) :

1. `admin` a toujours accès.
2. Un `employee` peut **lire** un document de **son** département.
3. Un `employee` peut **écrire** dans son département, s'il a le MFA.
4. N'importe qui peut **lire** un document `public`.
5. *(la faille volontaire)* Un `contractor` peut **lire** tout document
   `internal`, **sans vérifier le département**.

Une demande d'accès, c'est six informations :

```
rôle, action, classification, département du document,
département du demandeur, MFA
```

---

## Étape 2 — Fabriquer toutes les questions possibles

Au lieu d'inventer quelques exemples à la main, le programme **énumère
toutes les combinaisons** (`policy.All()`, `policy/dataset.go`) :

```
4 rôles × 3 actions × 3 classifications × 4 départements(doc)
        × 4 départements(demandeur) × 2 (MFA)  =  1152 demandes
```

Pourquoi tout énumérer ? Parce que le professeur (Rego) est déterministe
et gratuit : on peut donc lui demander de noter **l'intégralité des cas
possibles**, sans annotation humaine, sans échantillon et sans biais. Le
modèle verra ainsi tous les cas « ordinaires » au moins une fois.

---

## Étape 3 — Demander la réponse à Rego : voilà le dataset

Pour chacune des 1152 demandes, le programme appelle le **moteur OPA** sur
`policy.rego` (`dataset.access.allow`). La réponse devient l'étiquette.

| rôle | action | classif. | dép. doc | dép. dem. | MFA | Rego | étiquette |
|---|---|---|---|---|---|---|---|
| `admin` | `delete` | `confidential` | `hr` | `engineering` | non | ALLOW (règle 1) | 1 |
| `employee` | `read` | `internal` | `hr` | `hr` | non | ALLOW (règle 2) | 1 |
| `employee` | `read` | `internal` | `hr` | `finance` | non | DENY | 0 |
| `guest` | `read` | `public` | `hr` | `engineering` | non | ALLOW (règle 4) | 1 |
| `contractor` | `read` | `internal` | `hr` | `hr` | non | ALLOW (règle 5) | 1 |

> « Rego sert de professeur, mais le modèle ne l'appelle jamais. » Rego a
> fabriqué les étiquettes **une fois, en amont** ; le modèle apprend
> ensuite tout seul, et n'a plus besoin de Rego pour fonctionner.

---

## Étape 4 — Mettre de côté 24 cas, pour tester la compréhension

Parmi les 1152 demandes, 24 sont particulières : `contractor` + `read` +
`internal` avec un département **différent** du sien. C'est la faille de la
règle 5.

`policy.Split()` les retire du dataset d'entraînement :

- **1128 demandes** servent à apprendre ;
- **24 demandes-faille** sont gardées secrètes, pour vérifier à la fin si
  le modèle a « compris l'esprit » de la politique ou s'il a seulement
  mémorisé.

C'est le cœur de la démo : Rego dit ALLOW sur ces 24 cas (sa règle est
trop permissive), mais **aucun de ces cas n'a été montré au modèle**.

---

## Étape 5 — Transformer une demande en nombres

Un ordinateur ne lit pas `role=contractor`. Chaque demande devient **6
tokens** texte (`Request.Tokens()`) :

```
role=contractor  action=read  classification=internal
resource_dept=hr  requester_dept=marketing  mfa=false
```

Puis chaque token reçoit un **indice** entier, via le vocabulaire.

---

## Étape 6 — Construire le vocabulaire

Le programme collecte tous les tokens du dataset d'entraînement et leur
associe un numéro (`model.NewVocab`). Ici : **21 tokens** (les 20 valeurs
possibles + le token spécial `UNK` d'indice 0, pour tout mot inconnu).

C'est ce qui explique la dernière ligne de la démo : `intern` n'est pas
dans le vocabulaire, donc le modèle le voit comme `UNK`.

---

## Étape 7 — Initialiser le modèle au hasard

Avant d'apprendre, le modèle ne sait rien : ses poids sont tirés au hasard
(`model.New`), avec une **graine** (`seed = 1337`) pour que le résultat
soit reproductible. À ce stade, il répond au hasard.

---

## Étape 8 — La boucle d'apprentissage

C'est répété **400 fois** (`epochs = 400`) sur les 1128 demandes. Pour
chaque demande, quatre micro-étapes :

1. **Deviner (forward)** — le modèle traverse embedding → attention →
   pooling → classification et sort deux probabilités :
   `[P(DENY), P(ALLOW)]`.
2. **Mesurer l'erreur (loss)** — on compare avec l'étiquette de Rego. La
   perte vaut `-log(P(bonne réponse))` : plus il était confiant et faux,
   plus elle est grande.
3. **Comprendre son erreur (backward)** — on calcule, pour chaque poids,
   dans quel sens le corriger (rétropropagation).
4. **Se corriger (SGD)** — on décale chaque poids d'un petit pas :
   `poids = poids - lr × correction` (`lr = 0.02`).

La perte affichée doit **baisser** : c'est le signe que l'élève progresse.

| Époque | Perte (attention) | Ce que ça veut dire |
|---|---|---|
| 1 | ≈ 0,69 | il répond presque au hasard |
| 5 | ≈ 0,14 | il imite déjà très bien Rego |
| 400 | ≈ 0,0002 | il récite parfaitement le dataset d'entraînement |

---

## Étape 9 — Sauvegarder ce qui a été appris

À la fin, on écrit sur le disque les poids **et le vocabulaire**
(`weights/policy-attention.json`). Le vocabulaire est indispensable : sans
lui, un même mot changerait de numéro et le modèle deviendrait incohérent.

---

## Étape 10 — Poser une question au modèle

```bash
go run ./ask -demo
go run ./ask -role=contractor -action=read -classification=internal \
             -resource-dept=hr -requester-dept=marketing
```

Le CLI affiche trois réponses : celle de **Rego** (le professeur), celle du
**transformer**, et celle d'un **sac de mots** plus simple (sans attention),
pour comparer.

---

## Ce que la démo démontre

| Demande | Rego | Transformer | Ce que ça montre |
|---|---|---|---|
| `employee read public engineering/engineering` | ALLOW | ALLOW | cas ordinaire : tout le monde est d'accord |
| `guest delete confidential finance/marketing` | DENY | DENY | cas ordinaire : tout le monde est d'accord |
| `contractor read internal hr/hr` | ALLOW | ALLOW | il a **appris** la règle 5 vue à l'entraînement |
| `contractor read internal hr/marketing` | ALLOW | **DENY** | il **généralise** sur un cas jamais vu |
| `intern read ...` | DENY | DENY | rôle inconnu → token `UNK` |

La 4ᵉ ligne est l'essentiel. Rego applique sa règle 5 sans broncher et dit
ALLOW. Le modèle, lui, a retenu des autres règles que « même département »
compte, et refuse. **Il signale la faille que la règle ne sait pas
signaler.** C'est exactement ce qu'un système purement basé sur des règles
ne peut pas faire.

---

## Pourquoi un modèle « type Jev » pour de la classification

Le modèle Jev de TypeSafe AI ne rédige pas de texte : il **classe** une
entrée dans un ensemble d'options définies à l'avance et rend une
**probabilité** et une **confiance** (`README.md`, section « Filiation »).
Ce mode de fonctionnement a plusieurs intérêts pour la classification :

- **Sortie exploitable par un logiciel** — `ALLOW`/`DENY` + probabilité se
  consomme directement, pas besoin de lire ni d'interpréter une phrase.
- **Pas de réponse hors sujet** — les classes sont connues d'avance, donc
  pas de valeur inventée (« hallucination »).
- **La confiance guide la décision** — un cas à 51 % peut être envoyé en
  revue humaine, alors qu'une règle binaire n'exprime aucun doute.
- **Il généralise** — entraîné sur beaucoup d'exemples, il peut traiter des
  combinaisons jamais vues, là où une règle doit être écrite - ou corrigée.
- **Il est léger et rapide** — une fois entraîné, répondre coûte très peu,
  ce qui permet d'en poser beaucoup, à grande échelle.

---

## Et Rego, alors ? Complémentaires, pas concurrents

- **Rego** est **exact, lisible et auditable** : c'est le bon outil pour
  *définir* la politique et pour fabriquer les étiquettes.
- **Le modèle** est **approximatif mais généralisant** : il apprend de
  Rego, répond vite, donne un doute, et peut détecter des cas limites —
  à condition qu'un humain reprenne la main sur les cas douteux.

La démo ne dit pas « le modèle est meilleur que Rego ». Elle montre
comment un modèle statistique **interpole** ce qu'il a observé, comment on
peut le **mesurer**, et pourquoi cette capacité est utile dans un produit
de classification — tout en gardant Rego comme source de vérité.

---

## Récapitulatif en une image

```
policy.rego
    │  (OPA exécute la politique)
    ▼
1152 demandes  ──►  étiquettes ALLOW/DENY
    │
    ├─ 1128 servent à apprendre ──► Transformer ──► weights/policy-attention.json
    │                                   ▲
    │              tokens + vocabulaire │  (400 époques : deviner, mesurer, corriger)
    │
    └─ 24 cas-faille gardés secrets ──► go run ./ask -demo
                                            │
                                            ▼
                              Rego : ALLOW  |  Modèle : DENY ← la généralisation
```

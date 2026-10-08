## Ouverture

```slide
layout: hero
titre: Jev, en jouet
sous: |
  Un transformer écrit à la main
  apprend une politique d’accès
corps:
  - Go pur, sans framework de machine learning
  - Embeddings, self-attention, rétropropagation, SGD
accent: Il conteste, parfois, ce que la règle laisse passer.
credit: false
```

> **À dire** « On va construire, ligne par ligne, un mini-transformer qui apprend
> à imiter une politique de contrôle d'accès écrite en Rego. Et on va le surprendre
> à faire quelque chose que la règle ne sait pas faire. »

Le but n'est pas la performance : c'est de rendre visible ce que fait vraiment un
transformer, de l'embedding jusqu'à la rétropropagation.

## Le mode Jev

```slide
layout: cards
repere: Le principe
titre: Le modèle rend une décision typée, pas du texte
chapeau: >
  Le dépôt reprend le mode d’emploi de Jev (TypeSafe AI, 2026) : un état en entrée,
  une question fermée, une réponse typée et sa probabilité. Projet indépendant,
  sans affiliation.
tone: bleu
cards:
  - tone: violet
    pill: ENTRÉE
    items:
      - h: Une requête d’accès
      - p: Six tokens champ=valeur
      - b:
          - rôle, action, classification
          - deux départements, MFA
  - tone: bleu
    pill: QUESTION
    items:
      - h: Un choix fermé
      - p: ALLOW ou DENY
      - b:
          - aucune réponse hors schéma
          - aucun texte généré
  - tone: teal
    pill: SORTIE
    items:
      - h: Une classe et sa probabilité
      - p: "[P(DENY), P(ALLOW)]"
      - b:
          - consommable par un logiciel
          - une probabilité, pas une garantie
salle:
  cards:
    - tone: violet
      pill: ENTRÉE
      items:
        - h: Une requête d’accès
        - p: six tokens
    - tone: bleu
      pill: QUESTION
      items:
        - h: Un choix fermé
        - p: ALLOW ou DENY
    - tone: teal
      pill: SORTIE
      items:
        - h: Une classe
        - p: et sa probabilité
```

> **À dire** Jev, le modèle de TypeSafe AI, n'est pas un LLM : il reçoit un état et
> une question typée, et rend une valeur typée avec une probabilité. On reproduit ce
> mode d'emploi à l'échelle d'un jouet : une requête d'accès en six tokens, une
> question binaire ALLOW/DENY, une probabilité en sortie.

Différences à garder en tête : Jev est un modèle de production entraîné par RLCD ;
ici `DModel = 16`, une seule tête d'attention, apprentissage supervisé classique.

## La politique

```slide
layout: table
repere: La règle
titre: Cinq règles Rego, dont une faille volontaire
chapeau: >
  policy/policy.rego décide qui peut lire, écrire ou supprimer un document. C’est
  l’unique source de vérité, exécutée par le vrai moteur OPA.
tone: violet
cols: [[N°, 160], [QUI, 420], [PEUT, 1212]]
rows:
  - ["1", admin, tout faire]
  - ["2", employee, lire dans son département]
  - ["3", employee, "écrire dans son département, avec MFA"]
  - ["4", tout le monde, lire un document public]
  - ["5 ⚠", contractor, "lire tout document internal, sans vérifier le département"]
salle:
  rows: [1, 2, 4]
  note: ""
note: La règle 5 laisse un prestataire du marketing lire un document RH interne.
```

> **À dire** Cinq règles. Les règles 2 et 3 exigent que le demandeur soit du même
> département que le document. La règle 5, elle, oublie cette vérification : un
> contractor peut lire n'importe quel document interne. Strictement autorisé par la
> règle, mais un relecteur humain le jugerait suspect.

En salle, on ne montre que les règles 2, 3 et 5 : celles qui comptent pour la démo.
Les règles 1 et 4 se disent.

## Le dataset

```starlark
PALIERS = 3

def boite(c, x, y, w, h, tone, titre, sous):
    s = c.card(x, y, w, h, tone)
    s += c.text(x + w // 2, y + 110, titre, size = c.fs(64), bold = True, col = tone, anchor = "middle")
    s += c.text(x + w // 2, y + 180, sous, size = c.body, col = "text", anchor = "middle")
    return s

def render(step, c):
    s, y = c.head(titre = "Rego étiquette tout, le modèle ne voit pas tout", tone = "violet", repere = "Le dataset")
    top = y + 60
    h = 260
    w = 480
    gap = (c.right - c.left - 3 * w) // 2
    xa = c.left
    xb = c.left + w + gap
    xc = xb + w + gap

    s += boite(c, xa, top, w, h, "violet", "OPA", "exécute policy.rego")

    if step >= 2:
        s += c.arrow(xa + w + 10, top + h // 2, xb - 14, top + h // 2, col = "violet", width = 4)
        s += boite(c, xb, top, w, h, "bleu", "1 152", "requêtes énumérées")
    else:
        s += c.card(xb, top, w, h, "bleu", on = False)

    if step >= 3:
        hh = (h - 30) // 2
        s += c.arrow(xb + w + 10, top + h // 2, xc - 14, top + hh // 2, col = "teal", width = 4)
        s += c.arrow(xb + w + 10, top + h // 2, xc - 14, top + h - hh // 2, col = "rouge", width = 4)
        s += c.rect(xc, top, w, hh, fill = "teal.box", stroke = "teal", width = 3, rx = 16)
        s += c.text(xc + 30, top + hh // 2 + 14, "1 128 apprises", size = c.body, bold = True, col = "teal")
        s += c.rect(xc, top + h - hh, w, hh, fill = "rouge.box", stroke = "rouge", width = 3, rx = 16)
        s += c.text(xc + 30, top + h - hh // 2 + 14, "24 cachées", size = c.body, bold = True, col = "rouge")
    else:
        s += c.card(xc, top, w, h, "teal", on = False)

    msg = [
        "Le moteur de règles est l’oracle : gratuit et déterministe.",
        "On énumère tous les cas possibles, sans annotation humaine.",
        "Les 24 cas-faille ne sont jamais montrés au modèle.",
    ]
    tones = ["violet", "bleu", "rouge"]
    # en salle, la phrase du palier ; en lecture, les trois à la suite
    idx = range(3) if c.format == "lecture" else [step - 1]
    for k, i in enumerate(idx):
        yy = top + h + 100 + k * 70
        s += c.line(c.left, yy, c.left, yy + 50, col = tones[i], width = 8, cap = "round")
        s += c.text(c.left + 36, yy + 38, msg[i], size = c.body, col = "text")
    return s + c.footer(["policy/dataset.go, policy.All() et policy.Split()"], prov = "interne")
```

> **À dire** Palier 1 : le professeur, c'est le moteur OPA qui exécute `policy.rego`.
> Il est déterministe et gratuit : on peut lui faire noter autant de copies qu'on veut.

<!-- palier -->

Palier 2 : on énumère toutes les combinaisons, 4 rôles × 3 actions × 3 classifications
× 4 départements du document × 4 départements du demandeur × 2 MFA = 1 152 requêtes.

<!-- palier -->

Palier 3 : on met de côté les 24 requêtes contractor + read + internal avec un
département différent : la faille. Le modèle apprend sur les 1 128 autres, étiquetées
par OPA. Il ne verra jamais les 24 cas-faille à l'entraînement.

## L'architecture

```starlark
PALIERS = 3

TOKENS = [
    "role=contractor",
    "action=read",
    "classification=internal",
    "resource_dept=hr",
    "requester_dept=marketing",
    "mfa=false",
]

def render(step, c):
    s, y = c.head(titre = "L’attention croise les champs, la moyenne les ignore", tone = "bleu", repere = "Le modèle")
    x0 = c.left + 120
    lh = 92
    y0 = y + 50
    tw = 560
    for i, t in enumerate(TOKENS):
        yy = y0 + i * lh
        hot = step >= 2 and i in (3, 4)
        tone = "magenta" if hot else "violet"
        s += c.rect(x0, yy, tw, lh - 20, fill = tone + ".box", stroke = tone, width = 2, rx = 12)
        s += c.text(x0 + 20, yy + 50, t, size = c.fs(34), col = "text", bold = hot)

    if step >= 2:
        # l'attention : les deux départements se regardent
        ya = y0 + 3 * lh + (lh - 20) // 2
        yb = y0 + 4 * lh + (lh - 20) // 2
        s += c.path("M %d %d C %d %d %d %d %d %d" % (x0, ya, x0 - 110, ya, x0 - 110, yb, x0, yb),
                    stroke = "magenta", width = 6, cap = "round")
        for i in (0, 1, 2, 5):
            yi = y0 + i * lh + (lh - 20) // 2
            s += c.path("M %d %d C %d %d %d %d %d %d" % (x0, yi, x0 - 60, yi, x0 - 60, ya, x0, ya),
                        stroke = "magenta", width = 2, opacity = 0.3)

    xm = x0 + tw + 90
    xr = xm + 440
    ym = y0 + 2 * lh + 20
    if step == 1:
        s += c.arrow(x0 + tw + 20, ym + 30, xm - 10, ym + 30, col = "bleu", width = 4)
        s += c.text(xm, ym, ["21 embeddings", "de 16 nombres"], size = c.body, col = "bleu", bold = True, lh = 52)
    if step == 2:
        s += c.arrow(x0 + tw + 20, ym + 30, xm - 10, ym + 30, col = "magenta", width = 4)
        s += c.text(xm, ym, ["chaque token", "pèse les autres"], size = c.body, col = "magenta", bold = True, lh = 52)
    if step == 3:
        s += c.arrow(x0 + tw + 20, ym + 30, xm - 10, ym + 30, col = "bleu", width = 4)
        s += c.text(xm, ym + 10, ["moyenne", "puis softmax"], size = c.body, col = "text", lh = 52)
        bw = c.right - xr
        s += c.arrow(xr - 120, ym + 30, xr - 20, ym + 30, col = "bleu", width = 4)
        s += c.rect(xr, ym - 60, bw, 70, fill = "bleu.box", stroke = "bleu", rx = 10)
        s += c.rect(xr, ym - 60, int(bw * 0.952), 70, fill = "bleu.pill0", rx = 10)
        s += c.text(xr + 20, ym - 12, "DENY 95,2 %", size = c.body, bold = True, col = "white")
        s += c.text(xr, ym + 80, "ALLOW 4,8 %", size = c.body, col = "text")
    if c.format == "lecture":
        s += c.text(xm, y0 + 4 * lh + 20, [
            "1. Chaque token pointe vers un embedding de 16 nombres appris.",
            "2. La self-attention mélange les tokens : les arcs sont illustratifs.",
            "3. Moyenne, couche linéaire, softmax : deux probabilités.",
            "Un sac de mots saute l’étape 2 et moyenne des tokens isolés.",
        ], size = c.body, col = "text", lh = 40)
    return s
```

> **À dire** Palier 1 : la requête devient six tokens. Chacun est un indice dans un
> vocabulaire de 21 entrées (20 valeurs + UNK), et chaque indice pointe vers un vecteur
> de 16 nombres appris : l'embedding.

<!-- palier -->

Palier 2 : la self-attention. Chaque token calcule, avec Wq, Wk et Wv, combien il doit
regarder chacun des autres. C'est ce qui permet de représenter une combinaison comme
« même département », que la politique utilise partout. Les arcs sont illustratifs :
ce ne sont pas les poids réels du modèle.

<!-- palier -->

Palier 3 : on fait la moyenne des six vecteurs contextualisés, une couche linéaire,
un softmax, et on obtient deux probabilités. Sur cette requête précise, le modèle
répond DENY à 95,2 % (sortie de `go run ./ask`). Le sac de mots de comparaison saute
l'étape d'attention : il fait la moyenne d'embeddings indépendants.

## L'apprentissage

```starlark
PALIERS = 3

# Points documentés : (époque, perte du modèle avec attention)
PTS = [(1, 0.69), (5, 0.14), (400, 0.0002)]

def render(step, c):
    s, y = c.head(titre = "La perte s’effondre en cinq époques", tone = "teal", repere = "L’apprentissage")
    gx0 = c.left + 160
    gx1 = c.right - 40
    gy1 = c.bottom - 150
    gy0 = y + 110
    lmax = math.log(400.0) / math.log(10.0)

    def px(e):
        return gx0 + (gx1 - gx0) * (math.log(float(e)) / math.log(10.0)) / lmax

    def py(l):
        return gy1 - (gy1 - gy0) * l / 0.69

    s += c.line(gx0, gy1, gx1, gy1, col = "axis", width = 3)
    s += c.line(gx0, gy0 - 40, gx0, gy1, col = "axis", width = 3)
    s += c.text(gx0 - 24, gy0 + 12, "0,69", size = c.fs(36), anchor = "end", col = "muted")
    s += c.text(gx0 - 24, gy1 + 12, "0", size = c.fs(36), anchor = "end", col = "muted")
    s += c.text(gx1, gy1 + 60, "époques, échelle log →", size = c.fs(36), col = "muted", anchor = "end")

    # (nom, sous-titre, ton, dx, dy, ancre) : étiquette posée hors de la courbe
    labels = [
        ("ÉP. 1 · 0,69", "au hasard", "magenta", 120, 10, "start"),
        ("ÉP. 5 · 0,14", "imite Rego", "teal", 40, -90, "start"),
        ("ÉP. 400 · 0,0002", "récite tout", "bleu", 0, -150, "end"),
    ]
    for i in range(step):
        e, l = PTS[i]
        x, yy = px(e), py(l)
        if i > 0:
            pe, pl = PTS[i - 1]
            s += c.line(px(pe), py(pl), x, yy, col = "teal", width = 5, cap = "round")
        nom, sub, tone, dx, dy, anchor = labels[i]
        s += c.circle(x, yy, 14, fill = "bg", stroke = tone, width = 5)
        s += c.text(x + dx, yy + dy, [nom, sub], size = c.body, col = tone, bold = True, anchor = anchor, lh = 48)
    return s + c.footer(["PAS_A_PAS.md, étape 8 : lr = 0,02, seed 1337"], prov = "interne")
```

> **À dire** Palier 1 : au départ, les poids sont tirés au hasard, la perte vaut 0,69,
> soit −log(0,5) : le modèle répond au hasard.

<!-- palier -->

Palier 2 : après cinq passes sur les 1 128 exemples, la perte tombe à 0,14. Il imite
déjà très bien Rego. Chaque pas : deviner (forward), mesurer l'erreur (entropie
croisée), calculer la correction (rétropropagation à la main), corriger les poids (SGD).

<!-- palier -->

Palier 3 : à 400 époques, la perte vaut 0,0002 et l'exactitude sur l'entraînement est
de 100 %. Il récite parfaitement ce qu'il a vu. La vraie question : que fait-il sur ce
qu'il n'a pas vu ? Seuls trois points sont mesurés ; les segments entre eux ne sont pas
des mesures.

## La faille

```slide
layout: section
num: "?"
tone: rouge
kicker: LE CAS JAMAIS VU
titre: Un prestataire marketing demande à lire un document RH interne
sous: Rego applique sa règle 5. Que répond le modèle ?
```

> **À dire** C'est le cœur de la démo. contractor, read, internal, document RH,
> demandeur du marketing. Exactement le cas que la règle 5 laisse passer, et qu'on n'a
> jamais montré au modèle.

## Le verdict

```slide
layout: cards
repere: Le verdict
titre: Rego dit ALLOW, le modèle dit DENY à 95 %
chapeau: >
  Sortie de go run ./ask -role=contractor -action=read -classification=internal
  -resource-dept=hr -requester-dept=marketing.
tone: bleu
cards:
  - tone: violet
    pill: RÈGLE LITTÉRALE
    items:
      - h: ALLOW
      - p: La règle 5 s’applique, mécaniquement.
      - b:
          - aucun doute exprimé
          - aucune alerte possible
  - tone: bleu
    pill: TRANSFORMER
    items:
      - h: DENY · 95,2 %
      - p: Il a vu les règles 2 et 3 exiger le même département.
      - p: "Même requête avec MFA oui : ALLOW, P(DENY) 1,3 %."
      - e:
          label: SANS ATTENTION
          lines: ["sac de mots : DENY · 85,8 %", "mais règle 5 ratée : 0 / 8"]
salle:
  cards:
    - tone: violet
      pill: RÈGLE LITTÉRALE
      items:
        - h: ALLOW
        - p: la règle 5 s’applique
    - tone: bleu
      pill: TRANSFORMER
      items:
        - h: DENY · 95,2 %
        - p: le département compte
```

> **À dire** Rego répond ALLOW : il applique sa règle, il ne peut pas signaler qu'elle
> est suspecte. Le transformer a appris la règle 5 sur les cas même-département, et a
> vu par ailleurs que les règles 2 et 3 exigent la correspondance de département. Il
> généralise et répond DENY, à 95,2 %.

Attention : c'est une seule requête. La même avec MFA oui donne ALLOW (P(DENY) 1,3 %),
alors qu'aucune règle de lecture n'utilise le MFA. Sur les 24 cas-faille, 11 DENY et
13 ALLOW, presque tous à plus de 95 % de confiance dans un sens ou dans l'autre.

Le sac de mots, sans attention, dit aussi DENY sur ce cas (85,8 %), mais ce n'est pas une généralisation : il n'a pas appris la règle 5 (0/8 sur les cas contractor même département, DENY à 86,1 % sur contractor hr/hr) et refuse les 24 cas-faille, comme tout contractor en lecture interne. Mesuré : go run ./ask, et un réentraînement à seed 1337. C'est la vraie comparaison : le transformer apprend la règle 5 ET la conteste hors de son domaine, le sac de mots ne fait que refuser en bloc.

## L'honnêteté

```slide
layout: quad
repere: La mesure
titre: La généralisation est réelle, mais partielle
chapeau: Mesuré en réentraînant le modèle (lr = 0,02, 400 époques), seed 1337 puis 40 autres graines.
tone: teal
cards:
  - tone: teal
    lead: 11 / 24
    label: cas-faille refusés
    body: "Réponses tranchées dans les deux sens : 10 cas au-dessus de 95 % de DENY, 11 sous 2 %."
    prov: interne
    source: go run, seed 1337
  - tone: bleu
    lead: 8 / 8
    label: règle 5 apprise
    body: Les cas contractor même département restent ALLOW, comme Rego.
    prov: interne
    source: go run, seed 1337
  - tone: magenta
    lead: 0 à 21
    label: selon la graine
    body: "Sur 40 graines, médiane 9 : même code, mêmes données, autre point de départ, autre généralisation."
    prov: interne
    source: go run, graines 1 à 40
```

> **À dire** Soyons honnêtes : sur les 24 cas-faille, le modèle en refuse 11, pas 24.
> Il garde la règle 5 sur les 8 cas légitimes. Et selon la graine aléatoire, le
> nombre de cas refusés varie de 0 à 21.

Sur 40 graines (1 à 40), le nombre de refus va de 0 à 21, médiane 9 ; la graine 1337 en donne 11, un peu au-dessus de la médiane. REFERENCE.md annonçait « 0 à 19 » : non reproduit tel quel. La graine 6 refuse 24/24 mais rate la règle 5 (2/8), et 7 graines sur 40 n'atteignent pas 100 % sur l'entraînement.

Une divergence n'est ni un bug, ni la preuve que le modèle « comprend » l'intention :
c'est le signe attendu qu'un modèle statistique interpole au lieu d'exécuter la règle.

## La leçon

```slide
layout: cards
repere: La leçon
titre: Rego définit la politique, le modèle donne un indice à vérifier
tone: teal
cards:
  - tone: violet
    pill: REGO
    items:
      - h: Exact et auditable
      - b:
          - définit la politique
          - fabrique les étiquettes
          - reste la source de vérité
  - tone: bleu
    pill: LE MODÈLE
    items:
      - h: Approximatif et instable
      - b:
          - conteste certains cas jamais vus
          - sa confiance n’est pas fiable hors entraînement
          - un désaccord avec Rego mérite une revue humaine
salle:
  cards:
    - tone: violet
      pill: REGO
      items:
        - h: Exact et auditable
        - p: la source de vérité
    - tone: bleu
      pill: LE MODÈLE
      items:
        - h: Un indice instable
        - p: à faire revoir
```

> **À dire** Complémentaires, pas concurrents. Rego est le bon outil pour définir la
> politique et produire les étiquettes. Le modèle apprend de Rego, répond vite, et rend
> une probabilité. Mais sur les cas jamais vus, cette probabilité n'est pas un doute
> calibré : sur les 24 cas-faille, il tranche à plus de 95 % dans un sens ou dans
> l'autre, et la même requête change de camp quand on ajoute un MFA qu'aucune règle de
> lecture n'utilise. Ce qui reste utile : un désaccord avec la règle mérite qu'un
> humain regarde. Ce n'est pas un détecteur de failles.

## Clôture

```slide
layout: hero
titre: À vous de lire
sous: |
  go run ./training
  go run ./ask -demo
corps:
  - 1 200 lignes de Go, sans framework
  - "Pistes : multi-têtes, encodage positionnel, ALLOW / DENY / REVIEW"
credit: false
```

> **À dire** Le code tient en environ 1 200 lignes de Go. Deux
> commandes : entraîner, puis interroger. PAS_A_PAS.md pour comprendre sans prérequis,
> TUTORIAL.md pour le détail technique, REFERENCE.md pour les paramètres.

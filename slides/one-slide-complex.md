## Jev, la carte des accès

```starlark
# One-slide « complexe », pensé pour le format lecture (plancher 22 px).
# À gauche, la carte des 1 152 requêtes, recalculée ici depuis les cinq règles de
# policy.rego ; au centre, ce que calcule le modèle ; à droite, le verdict sur le
# coin rouge de la carte (les 24 cas-faille jamais montrés au modèle).

ROLES = ["admin", "employee", "contractor", "guest"]
ACTIONS = ["read", "write", "delete"]
CLASSES = ["public", "internal", "confidential"]
DEPTS = ["engineering", "hr", "finance", "marketing"]

# Les cinq règles de policy/policy.rego, transcrites pour dessiner la carte.
def allow(role, action, cls, rd, qd, mfa):
    if role == "admin":
        return True
    if role == "employee" and action == "read" and rd == qd:
        return True
    if role == "employee" and action == "write" and rd == qd and mfa:
        return True
    if action == "read" and cls == "public":
        return True
    if role == "contractor" and action == "read" and cls == "internal":
        return True
    return False

def faille(role, action, cls, rd, qd):
    return role == "contractor" and action == "read" and cls == "internal" and rd != qd

TOKENS = [
    "role=contractor",
    "action=read",
    "classification=internal",
    "resource_dept=hr",
    "requester_dept=marketing",
    "mfa=false",
]

PTS = [(1, 0.69), (5, 0.14), (400, 0.0002)]

# P(DENY) du transformer sur les 24 cas-faille, mesurée avec weights/policy-attention.json
# (seed 1337). Ordre des couples (doc ← demandeur) : eng←hr, eng←fin, eng←mkt, hr←eng,
# hr←fin, hr←mkt, fin←eng, fin←hr, fin←mkt, mkt←eng, mkt←hr, mkt←fin.
P24_SANS_MFA = [0.0, 0.996, 0.0, 0.003, 1.0, 0.952, 1.0, 1.0, 0.118, 0.317, 0.002, 1.0]
P24_AVEC_MFA = [0.0, 0.548, 0.0, 0.0, 1.0, 0.013, 0.952, 0.993, 0.004, 0.001, 0.0, 1.0]

# géométrie de la carte
CW = 20      # pas des colonnes
CH = 12      # pas des lignes
GX = 236     # gauche de la grille
GY = 280     # haut de la grille
GA = 4       # écart entre actions
GR = 14      # écart entre rôles
GD = 3       # écart entre départements du document
GC = 16      # écart entre classifications

def col_x(r, a, m):
    return GX + r * (3 * (2 * CW + GA) - GA + GR) + a * (2 * CW + GA) + m * CW

def row_y(ci, rd, qd):
    return GY + ci * (16 * CH + 3 * GD + GC) + rd * (4 * CH + GD) + qd * CH

def titre_bloc(c, x, y, num, texte, tone):
    s = c.circle(x + 18, y - 8, 18, fill = tone + ".pill0")
    s += c.text(x + 18, y, num, size = c.fs(22), bold = True, col = "white", anchor = "middle")
    s += c.text(x + 46, y, texte, size = c.fs(22), bold = True, spacing = 1.4, col = tone + ".lead")
    return s

def render(step, c):
    s, y = c.head(titre = "Rego trace la carte, le transformer en conteste un coin",
                  repere = "Jev · un mini-transformer face à une politique Rego", tone = "bleu")

    # ---------- ① la carte ----------
    g = titre_bloc(c, c.left, y + 14, "1", "1 152 REQUÊTES ÉTIQUETÉES PAR REGO", "violet")
    for r, role in enumerate(ROLES):
        x0 = col_x(r, 0, 0)
        x1 = col_x(r, 2, 1) + CW
        g += c.text((x0 + x1) // 2, GY - 36, role, size = c.fs(22), bold = True, col = "text", anchor = "middle")
        for a, lettre in enumerate(["L", "É", "S"]):
            g += c.text(col_x(r, a, 0) + CW, GY - 8, lettre, size = c.fs(22), col = "muted", anchor = "middle")
    for ci, cls in enumerate(CLASSES):
        ym = row_y(ci, 0, 0) + (row_y(ci, 3, 3) + CH - row_y(ci, 0, 0)) // 2
        g += c.text(GX - 16, ym + 8, cls, size = c.fs(22), bold = True, col = "rouge.lead" if cls == "internal" else "text", anchor = "end")
    for r, role in enumerate(ROLES):
        for a, action in enumerate(ACTIONS):
            for m, mfa in enumerate([True, False]):
                x = col_x(r, a, m)
                for ci, cls in enumerate(CLASSES):
                    for i, rd in enumerate(DEPTS):
                        for j, qd in enumerate(DEPTS):
                            yy = row_y(ci, i, j)
                            if faille(role, action, cls, rd, qd):
                                fill = "rouge"
                            elif allow(role, action, cls, rd, qd, mfa):
                                fill = "teal"
                            else:
                                fill = "empty"
                            g += c.rect(x, yy, CW - 3, CH - 3, fill = fill, rx = 2)
    # le coin rouge
    fx0 = col_x(2, 0, 0) - 5
    fx1 = col_x(2, 0, 1) + CW + 2
    fy0 = row_y(1, 0, 0) - 5
    fy1 = row_y(1, 3, 3) + CH + 2
    g += c.rect(fx0, fy0, fx1 - fx0, fy1 - fy0, stroke = "rouge", width = 3, rx = 6)
    # légende
    ly = row_y(2, 3, 3) + CH + 44
    lx = c.left
    for fill, txt in [("teal", "ALLOW"), ("empty", "DENY"), ("rouge", "faille, hors entraînement")]:
        g += c.rect(lx, ly - 17, 18, 18, fill = fill, rx = 3)
        g += c.text(lx + 28, ly, txt, size = c.fs(22), col = "text")
        lx += 28 + c.measure(txt, c.fs(22)) + 30
    g += c.text(c.left, ly + 32, "L · É · S : lire, écrire, supprimer, chacun MFA oui | non", size = c.fs(22), col = "muted")
    s += c.group(g, box = [c.left, y - 20, 820 - c.left, c.bottom - y + 20])

    # ---------- ② le modèle ----------
    # Deux temps : APPRENDRE sur les cases vertes et grises, puis INTERROGER sur une
    # case rouge. La SORTIE (deux probabilités) alimente la jauge de la colonne ③.
    mx = 868
    mw = 380
    g = titre_bloc(c, mx, y + 14, "2", "IL APPREND À IMITER REGO", "bleu")
    g += c.text(mx, y + 52, "Objectif : prédire ALLOW / DENY sans la règle", size = c.fs(22), col = "text")

    # a · apprendre
    ly0 = y + 92
    g += c.text(mx, ly0, "APPRENDRE", size = c.fs(22), bold = True, spacing = 1.4, col = "teal.lead")
    g += c.text(mx + c.measure("APPRENDRE", c.fs(22), bold = True, spacing = 1.4) + 12, ly0,
                "sur les 1 128 autres cases", size = c.fs(22), col = "text")
    gx0 = mx + 60
    gx1 = mx + mw + 70
    gy0 = ly0 + 40
    gy1 = ly0 + 170
    lmax = math.log(400.0)

    def px(e):
        return gx0 + (gx1 - gx0) * math.log(float(e)) / lmax

    def py(l):
        return gy1 - (gy1 - gy0) * l / 0.69

    g += c.line(gx0, gy1, gx1, gy1, col = "axis", width = 2)
    g += c.line(gx0, gy0, gx0, gy1, col = "axis", width = 2)
    d = ""
    for i, (e, l) in enumerate(PTS):
        d += ("L " if i else "M ") + "%s %s " % (c.num(px(e), 1), c.num(py(l), 1))
    g += c.path(d, stroke = "teal", width = 4, cap = "round", join = "round")
    for (e, l), t, a, dx, dy in [(PTS[0], "0,69", "end", -16, 8), (PTS[1], "0,14 · ép. 5", "start", 16, -12), (PTS[2], "0,0002", "end", 0, -18)]:
        g += c.circle(px(e), py(l), 8, fill = "bg", stroke = "teal", width = 3)
        g += c.text(px(e) + dx, py(l) + dy, t, size = c.fs(22), bold = True, col = "teal.lead", anchor = a)
    g += c.text(gx1, gy1 + 28, "perte sur 400 époques", size = c.fs(22), col = "muted", anchor = "end")

    # b · interroger
    iy0 = gy1 + 80
    g += c.text(mx, iy0, "INTERROGER", size = c.fs(22), bold = True, spacing = 1.4, col = "rouge.lead")
    g += c.text(mx + c.measure("INTERROGER", c.fs(22), bold = True, spacing = 1.4) + 12, iy0,
                "une case rouge, jamais vue", size = c.fs(22), col = "text")
    ty0 = iy0 + 18
    pitch = 52
    for i, t in enumerate(TOKENS):
        hot = i in (3, 4)
        tone = "magenta" if hot else "bleu"
        g += c.rect(mx, ty0 + i * pitch, mw, 40, fill = tone + ".box", stroke = tone, width = 2 if hot else 1.2, rx = 10)
        g += c.text(mx + 16, ty0 + i * pitch + 28, t, size = c.fs(22), bold = hot, col = "text")
    # attention : chaque token pèse les autres (schéma, pas les poids réels)
    ax = mx + mw
    for i in range(6):
        for j in range(i + 1, 6):
            yi = ty0 + i * pitch + 20
            yj = ty0 + j * pitch + 20
            hot = (i, j) == (3, 4)
            bulge = 26 + (j - i) * 18
            g += c.path("M %d %d C %d %d %d %d %d %d" % (ax, yi, ax + bulge, yi, ax + bulge, yj, ax, yj),
                        stroke = "magenta" if hot else "bleu", width = 5 if hot else 1.5, opacity = 1 if hot else 0.35)
    py0 = ty0 + 6 * pitch
    g += c.arrow(mx + mw // 2, py0, mx + mw // 2, py0 + 28, col = "bleu", width = 3, head = 10)
    g += c.text(ax + 70, py0 + 22, "attention (schéma)", size = c.fs(22), italic = True, col = "magenta.lead", anchor = "end")
    oy = py0 + 34
    g += c.rect(mx, oy, mw, 50, fill = "bleu.pill0", rx = 10)
    g += c.text(mx + mw // 2, oy + 33, "SORTIE : P(DENY), P(ALLOW)", size = c.fs(22), bold = True, col = "white", anchor = "middle")
    g += c.text(mx, oy + 84, "moyenne · linéaire · softmax", size = c.fs(22), col = "muted")
    s += c.group(g, box = [mx - 10, y - 20, 1370 - mx, c.bottom - y + 20])

    # les deux flux depuis la carte
    s += c.arrow(col_x(3, 2, 1) + CW + 8, ly0 + 100, mx - 8, ly0 + 100, col = "teal", width = 4, head = 12)
    s += c.path("M %d %d C %d %d %d %d %d %d" % (fx1, fy0 + 60, 800, fy0 + 40, 780, ty0 + 20, mx - 20, ty0 + 20),
                stroke = "rouge", width = 3, dash = "8 6", cap = "round")
    s += c.arrow(mx - 30, ty0 + 20, mx - 6, ty0 + 20, col = "rouge", width = 3, head = 10)
    # la sortie nourrit la jauge
    s += c.arrow(mx + mw + 8, oy + 25, 1418, oy + 25, col = "bleu", width = 3, head = 12)

    # ---------- ③ le résultat ----------
    vx = 1400
    vw = c.right - vx
    g = titre_bloc(c, vx, y + 14, "3", "RÉSULTAT SUR LA FAILLE", "rouge")

    # le résultat en une phrase
    phrase = "Il conteste 11 des 24 accès suspects, mais de façon instable : un indice, pas un détecteur."
    g += c.text(vx, y + 52, c.wrap(phrase, c.fs(22), vw - 30), size = c.fs(22), bold = True, col = "text", lh = 30)

    # les 24 cases rouges : P(DENY) mesurée pour chacune (poids seed 1337).
    # Colonnes : les 12 couples (doc ← demandeur) dans l'ordre de DEPTS ; lignes : MFA non, MFA oui.
    qy = y + 166
    g += c.text(vx, qy, "LES 24 CASES ROUGES · P(DENY)", size = c.fs(22), bold = True, spacing = 1.4, col = "magenta.lead")
    lw = 112
    sp = (vw - lw) // 12
    sq = sp - 5
    demo = 5     # hr ← marketing
    for row, (nom, probas) in enumerate([("MFA non", P24_SANS_MFA), ("MFA oui", P24_AVEC_MFA)]):
        yy = qy + 18 + row * (sq + 10)
        g += c.text(vx, yy + sq - 3, nom, size = c.fs(22), col = "text")
        for k, pd in enumerate(probas):
            xx = vx + lw + k * sp
            if pd >= 0.5:
                g += c.rect(xx, yy, sq, sq, fill = "bleu.pill0", opacity = 0.35 + 0.65 * pd, rx = 4)
            else:
                g += c.rect(xx, yy, sq, sq, fill = "bleu.pill0", opacity = 0.6 * pd, rx = 4)
                g += c.rect(xx, yy, sq, sq, stroke = "rouge", width = 2, rx = 4)
    hx = vx + lw + demo * sp - 4
    g += c.rect(hx, qy + 12, sq + 8, 2 * sq + 22, stroke = "magenta", width = 3, rx = 6)
    g += c.text(vx, qy + 2 * sq + 74, ["▲ la case posée : 95,2 %", "même requête, MFA oui : 1,3 %"],
                size = c.fs(22), bold = True, col = "magenta.lead", lh = 30)
    g += c.text(vx, qy + 2 * sq + 144, ["■ 11 DENY · □ 13 ALLOW, comme Rego", "Sur 40 graines : 0 à 21 (médiane 9)."],
                size = c.fs(22), col = "muted", lh = 30)

    # sur la case posée
    by = oy + 30 - 150 - 160   # juste au-dessus de la jauge
    g += c.text(vx, by, "SUR LA CASE POSÉE", size = c.fs(22), bold = True, spacing = 1.4, col = "magenta.lead")
    g += c.rect(vx, by + 20, vw, 46, fill = "violet.pill0", rx = 10)
    g += c.text(vx + 16, by + 51, "Rego : ALLOW, règle 5", size = c.fs(22), bold = True, col = "white")
    g += c.rect(vx, by + 78, vw, 46, fill = "box", stroke = "teal", width = 1.5, rx = 10)
    g += c.rect(vx, by + 78, vw, 46, fill = "teal.pill0", rx = 10)
    g += c.text(vx + 16, by + 109, "Sac de mots : DENY, règle 5 ratée", size = c.fs(22), bold = True, col = "white")

    # la jauge, à hauteur de la SORTIE du modèle
    cx = vx + vw // 2
    cy = oy + 30
    R = 150
    p = 0.952
    th = math.pi * (1 - p)
    g += c.path("M %d %d A %d %d 0 0 1 %d %d" % (cx - R, cy, R, R, cx + R, cy), stroke = "empty", width = 28, cap = "round")
    g += c.path("M %d %d A %d %d 0 0 1 %s %s" % (cx - R, cy, R, R, c.num(cx + R * math.cos(th), 1), c.num(cy - R * math.sin(th), 1)),
                stroke = "bleu", width = 28, cap = "round")
    g += c.text(cx, cy - 44, "95,2 %", size = c.fs(56), bold = True, col = "bleu.lead", anchor = "middle")
    g += c.text(cx, cy - 6, "Jev : DENY", size = c.fs(26), bold = True, col = "text", anchor = "middle")
    g += c.text(cx - R, cy + 40, "0", size = c.fs(22), col = "muted", anchor = "middle")
    g += c.text(cx + R, cy + 40, "1", size = c.fs(22), col = "muted", anchor = "middle")
    g += c.text(cx, cy + 40, "cette case", size = c.fs(22), col = "muted", anchor = "middle")
    s += c.group(g, box = [vx - 10, y - 20, vw + 10, c.bottom - y + 20])

    return s + c.footer(["carte : les 5 règles de policy.rego · chiffres : go run ./training, ./ask, 40 graines, P(DENY) seed 1337"], prov = "interne")
```

> **À dire** À gauche, la carte complète : les 1 152 requêtes possibles, une case par
> requête. Colonnes : rôle, action, MFA ; lignes : classification, département du
> document, département du demandeur. En vert ce que Rego autorise. On lit la
> politique dans les motifs : la colonne admin pleine, la bande « public » en lecture,
> la diagonale « même département » chez employee. En rouge, le coin que la règle 5
> ouvre : contractor, lecture, interne, autre département. Ces 24 cases sont retirées
> de l'entraînement.

Au centre, l'objectif du modèle : prédire la couleur d'une case, ALLOW ou DENY, sans
connaître la règle. Deux temps. APPRENDRE : il voit les 1 128 cases vertes et grises,
étiquetées par Rego ; la perte passe de 0,69 à 0,14 en cinq époques, 0,0002 à 400
(trois points mesurés). INTERROGER : on lui pose une case rouge qu'il n'a jamais vue,
découpée en six tokens ; l'attention croise les champs (les arcs sont un schéma, pas
les poids appris). La SORTIE est un couple de probabilités, P(DENY) et P(ALLOW), qui
alimente la jauge de droite.

À droite, le résultat. Sur la case posée : Rego dit ALLOW, Jev dit DENY à 95,2 %. Le sac de mots, sans attention, dit aussi DENY sur ce cas (85,8 %), mais ce n'est pas une généralisation : il n'a pas appris la règle 5 (0/8 sur les cas contractor même département, DENY à 86,1 % sur contractor hr/hr) et refuse les 24 cas-faille, comme tout contractor en lecture interne. Mesuré : go run ./ask, et un réentraînement à seed 1337. En une phrase : le modèle conteste 11 des 24 accès suspects ; c'est un signal à faire revoir, pas une meilleure règle. Sur les 24 cas-faille, 11 sont refusés
(les carrés pleins illustrent le compte, pas des cas précis), la règle 5 légitime est
gardée sur 8/8. Sur 40 graines (1 à 40), le nombre de refus va de 0 à 21, médiane 9 ; la graine 1337 en donne 11, un peu au-dessus de la médiane. REFERENCE.md annonçait « 0 à 19 » : non reproduit tel quel. La graine 6 refuse 24/24 mais rate la règle 5 (2/8), et 7 graines sur 40 n'atteignent pas 100 % sur l'entraînement.

Est-ce mieux que Rego ? Non. Rego est la spécification : il applique sa règle sans
erreur, c'est un relecteur humain qui juge la règle 5 suspecte. Le modèle ne corrige
pas Rego, il est en désaccord avec son professeur sur des cas jamais vus. 11 sur 24,
c'est moins de la moitié, et le chiffre varie de 0 à 21 selon la graine. Les fausses
alertes (désaccord sur des cas légitimes jamais vus) ne sont pas mesurées, et les 24
cas ont été choisis en sachant où était la faille. Ce que ça vaut : là où le modèle et
la règle divergent, un humain devrait regarder.

Les 24 cases de la colonne de droite portent la P(DENY) réellement mesurée pour chaque
cas-faille (plus foncé = plus sûr de refuser). Le modèle ne doute pas : il tranche dans
les deux sens, 10 cas au-dessus de 95 %, 11 sous 2 %. La case posée (95,2 %) est
typique du groupe DENY, pas du coin entier. Et la même requête avec MFA, champ que
aucune règle de lecture n'utilise, tombe à 1,3 % : ALLOW. Sa « correspondance de
département » est fragile, et sa confiance n'est pas fiable sur des cas jamais vus.

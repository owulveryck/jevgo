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
CH = 10      # pas des lignes
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

PROCHAINE = "remonter les incohérences à un relecteur, apprendre de ses décisions, puis tester Jev : rappel, calibration, invariance"

def gy1_vignette(ly0):
    return ly0 + 170

def titre_bloc(c, x, y, num, texte, tone):
    s = c.circle(x + 18, y - 8, 18, fill = tone + ".pill0")
    s += c.text(x + 18, y, num, size = c.fs(22), bold = True, col = "white", anchor = "middle")
    s += c.text(x + 46, y, texte, size = c.fs(22), bold = True, spacing = 1.4, col = tone + ".lead")
    return s

def render(step, c):
    s, y = c.head(titre = "Le modèle n’exécute pas la règle : il interpole à partir d’exemples",
                  repere = "Jev · POC : un mini-transformer imitant le mode Jev, pas Jev lui-même", tone = "bleu")

    # ---------- ① la carte ----------
    g = titre_bloc(c, c.left, y + 14, "1", "REGO FOURNIT 1 152 EXEMPLES", "violet")
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
    # là où le MFA change la réponse de Rego : 12 paires sur 576 (employee, écriture, même département)
    for ci in range(3):
        for i in range(4):
            g += c.rect(col_x(1, 1, 0) - 3, row_y(ci, i, i) - 3, 2 * CW + 2, CH + 2, stroke = "magenta", width = 2.5, rx = 3)
    # légende
    ly = row_y(2, 3, 3) + CH + 34
    lx = c.left
    for fill, txt in [("teal", "ALLOW"), ("empty", "DENY"), ("rouge", "faille, hors entraînement")]:
        g += c.rect(lx, ly - 17, 18, 18, fill = fill, rx = 3)
        g += c.text(lx + 28, ly, txt, size = c.fs(22), col = "text")
        lx += 28 + c.measure(txt, c.fs(22)) + 30
    g += c.text(c.left, ly + 32, "L · É · S : lire, écrire, supprimer, en 2 colonnes : MFA oui | non",
                size = c.fs(22), col = "muted")
    g += c.rect(c.left, ly + 47, 30, 16, stroke = "magenta", width = 2.5, rx = 3)
    g += c.text(c.left + 40, ly + 62, ["le MFA (authentification multifacteur) change la", "réponse dans 12 cas sur 576 : employee écrit chez lui"],
                size = c.fs(22), col = "magenta.lead", lh = 30)
    s += c.group(g, box = [c.left, y - 20, 820 - c.left, c.bottom - y + 20])

    # ---------- ② le modèle ----------
    # Deux temps : APPRENDRE sur les cases vertes et grises, puis INTERROGER sur une
    # case rouge. La SORTIE (deux probabilités) alimente la jauge de la colonne ③.
    mx = 868
    mw = 380
    g = titre_bloc(c, mx, y + 14, "2", "IL APPREND PAR L’EXEMPLE", "bleu")
    g += c.text(mx, y + 52, "Objectif : prédire ALLOW / DENY sans la règle", size = c.fs(22), col = "text")

    # a · apprendre
    ly0 = y + 92
    g += c.text(mx, ly0, "APPRENDRE", size = c.fs(22), bold = True, spacing = 1.4, col = "teal.lead")
    g += c.text(mx + c.measure("APPRENDRE", c.fs(22), bold = True, spacing = 1.4) + 12, ly0,
                "sur les 1 128 autres cases", size = c.fs(22), col = "text")
    # ce que voit le modèle : la carte en vignette, le coin rouge masqué (estompé)
    vcw, vch = 6, 3
    vx0, vy0 = mx, ly0 + 18
    for r, role in enumerate(ROLES):
        for a, action in enumerate(ACTIONS):
            for m, mfa in enumerate([True, False]):
                xx = vx0 + r * (6 * vcw + 3) + a * (2 * vcw + 1) + m * vcw
                for ci, cls in enumerate(CLASSES):
                    for i, rd in enumerate(DEPTS):
                        for j, qd in enumerate(DEPTS):
                            yy = vy0 + ci * (16 * vch + 4) + (i * 4 + j) * vch
                            if faille(role, action, cls, rd, qd):
                                g += c.rect(xx, yy, vcw - 1, vch - 1, fill = "rouge", opacity = 0.12)
                            elif allow(role, action, cls, rd, qd, mfa):
                                g += c.rect(xx, yy, vcw - 1, vch - 1, fill = "teal")
                            else:
                                g += c.rect(xx, yy, vcw - 1, vch - 1, fill = "empty")
    mfx = vx0 + 2 * (6 * vcw + 3) - 2
    mfy = vy0 + (16 * vch + 4) - 2
    g += c.rect(mfx, mfy, 2 * vcw + 3, 16 * vch + 3, stroke = "rouge", width = 2.5, dash = "4 3", rx = 3)
    g += c.text(vx0, gy1_vignette(ly0) + 28, "24 cases masquées", size = c.fs(22), italic = True, col = "rouge.lead")

    gx0 = mx + 230
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
    for (e, l), t, a, dx, dy in [(PTS[0], "0,69", "end", -16, 8), (PTS[1], "0,14 · ép. 5", "start", 16, -40), (PTS[2], "0,0002", "end", 0, -18)]:
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
    # le coin de la carte en bande verticale : 16 couples (doc, demandeur) × MFA oui | non.
    # Vert (même département) : vu à l'entraînement. Rouge : masqué, sauf la case posée.
    scw, sch = 15, 300 // 16
    for i, rd in enumerate(DEPTS):
        for j, qd in enumerate(DEPTS):
            for m in range(2):
                xx = mx + m * scw
                yy = ty0 + (i * 4 + j) * sch
                if rd == qd:
                    g += c.rect(xx, yy, scw - 3, sch - 3, fill = "teal", rx = 2)
                elif (rd, qd, m) == ("hr", "marketing", 1):
                    g += c.rect(xx, yy, scw - 3, sch - 3, fill = "rouge", rx = 2)
                else:
                    g += c.rect(xx, yy, scw - 3, sch - 3, fill = "rouge", opacity = 0.12, rx = 2)
    qx = mx + scw
    qy0 = ty0 + 7 * sch
    g += c.rect(qx - 3, qy0 - 3, scw + 3, sch + 3, stroke = "magenta", width = 2.5, rx = 3)
    tx = mx + 2 * scw + 30
    tw = mx + mw - tx
    g += c.line(qx + scw, qy0, tx - 4, ty0, col = "magenta", width = 1.5, opacity = 0.6)
    g += c.line(qx + scw, qy0 + sch - 3, tx - 4, ty0 + 5 * pitch + 40, col = "magenta", width = 1.5, opacity = 0.6)
    for i, t in enumerate(TOKENS):
        hot = i in (3, 4)
        tone = "magenta" if hot else "bleu"
        g += c.rect(tx, ty0 + i * pitch, tw, 40, fill = tone + ".box", stroke = tone, width = 2 if hot else 1.2, rx = 10)
        g += c.text(tx + 14, ty0 + i * pitch + 28, t, size = c.fs(22), bold = hot, col = "text")
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
    s += c.group(g, box = [mx - 10, y - 20, 1370 - mx, c.bottom - y + 20])

    # les deux flux depuis la carte
    s += c.arrow(col_x(3, 2, 1) + CW + 8, ly0 + 100, mx - 8, ly0 + 100, col = "teal", width = 4, head = 12)
    s += c.path("M %d %d C %d %d %d %d %d %d" % (fx1, fy0 + 60, 800, fy0 + 40, 780, ty0 + 135, mx - 20, ty0 + 135),
                stroke = "rouge", width = 3, dash = "8 6", cap = "round")
    s += c.arrow(mx - 30, ty0 + 135, mx - 6, ty0 + 135, col = "rouge", width = 3, head = 10)
    # la sortie du modèle mène au bloc SORTIE de la colonne ③
    s += c.path("M %d %d L 1378 %d L 1378 %d" % (mx + mw + 8, oy + 25, oy + 25, y + 52), stroke = "bleu", width = 3, join = "round")
    s += c.arrow(1376, y + 52, 1394, y + 52, col = "bleu", width = 3, head = 10)

    # ---------- ③ sur les cas cachés : SORTIE puis CONCLUSION ----------
    vx = 1400
    vw = c.right - vx
    g = titre_bloc(c, vx, y + 14, "3", "SUR LES CAS CACHÉS", "rouge")

    # SORTIE : ce que le modèle produit, une P(DENY) par case cachée (poids seed 1337).
    # Colonnes : les 12 couples (doc ← demandeur) dans l'ordre de DEPTS ; lignes : MFA non, MFA oui.
    sy = y + 60
    g += c.text(vx, sy, "SORTIE", size = c.fs(22), bold = True, spacing = 1.4, col = "bleu.lead")
    g += c.text(vx + c.measure("SORTIE", c.fs(22), bold = True, spacing = 1.4) + 12, sy,
                "une P(DENY) par case cachée", size = c.fs(22), col = "text")
    lw = 112
    sp = (vw - lw) // 12
    sq = sp - 5
    demo = 5     # hr ← marketing
    for row, (nom, probas) in enumerate([("MFA non", P24_SANS_MFA), ("MFA oui", P24_AVEC_MFA)]):
        yy = sy + 22 + row * (sq + 10)
        g += c.text(vx, yy + sq - 3, nom, size = c.fs(22), col = "text")
        for k, pd in enumerate(probas):
            xx = vx + lw + k * sp
            if pd >= 0.5:
                g += c.rect(xx, yy, sq, sq, fill = "bleu.pill0", opacity = 0.35 + 0.65 * pd, rx = 4)
            else:
                g += c.rect(xx, yy, sq, sq, fill = "bleu.pill0", opacity = 0.6 * pd, rx = 4)
                g += c.rect(xx, yy, sq, sq, stroke = "rouge", width = 2, rx = 4)
    hx = vx + lw + demo * sp - 4
    g += c.rect(hx, sy + 16, sq + 8, 2 * sq + 22, stroke = "magenta", width = 3, rx = 6)
    ly3 = sy + 2 * sq + 66
    g += c.rect(vx, ly3 - 17, 18, 18, fill = "bleu.pill0", rx = 3)
    g += c.text(vx + 26, ly3, "DENY (foncé = sûr)", size = c.fs(22), col = "muted")
    lx3 = vx + 26 + c.measure("DENY (foncé = sûr)", c.fs(22)) + 24
    g += c.rect(lx3, ly3 - 17, 18, 18, stroke = "rouge", width = 2, rx = 3)
    g += c.text(lx3 + 26, ly3, "ALLOW", size = c.fs(22), col = "muted")
    g += c.text(vx, ly3 + 40, ["▲ case posée : 95,2 % → DENY", "même requête + MFA : 1,3 % → ALLOW"],
                size = c.fs(22), bold = True, col = "magenta.lead", lh = 30)
    ry = ly3 + 92
    g += c.rect(vx, ry, vw, 44, fill = "violet.pill0", rx = 10)
    g += c.text(vx + 16, ry + 30, "Rego : ALLOW sur les 24 (règle 5)", size = c.fs(22), bold = True, col = "white")

    # CE QUE LE POC VALIDE / NE VALIDE PAS
    def bilan(g, y0, tone, titre, signe, items):
        lignes = []
        for it in items:
            lignes.append(c.wrap(it, c.fs(22), vw - 66))
        h = 56
        for l in lignes:
            h += len(l) * 28 + 12
        g += c.rect(vx, y0, vw, h, fill = tone + ".box", stroke = tone, width = 2, rx = 14)
        g += c.text(vx + 18, y0 + 34, titre, size = c.fs(22), bold = True, spacing = 1.4, col = tone + ".lead")
        yy = y0 + 70
        for l in lignes:
            g += c.text(vx + 18, yy, signe, size = c.fs(22), bold = True, col = tone + ".lead")
            g += c.text(vx + 48, yy, l, size = c.fs(22), col = "text", lh = 28)
            yy += len(l) * 28 + 12
        return g, y0 + h

    g, yb = bilan(g, ry + 56, "teal", "LE POC VALIDE", "✓", [
        "Apprendre une politique par l’exemple : 1 128 / 1 128",
        "Sortie typée : classe + probabilité",
        "Repérer une incohérence entre règles voisines (2 et 5)",
    ])
    g, yb = bilan(g, yb + 12, "rouge", "IL NE VALIDE PAS", "✗", [
        "Dire laquelle est la faille (règle 2 cachée : 24/24)",
        "Une confiance fiable (MFA, graine)",
        "Un usage en gateway",
    ])
    if yb > c.bottom:
        c.warn("bilan : déborde de %d px" % (yb - c.bottom))
    s += c.group(g, box = [vx - 10, y - 20, vw + 10, c.bottom - y + 20])

    # prochaine étape : un bandeau sur toute la largeur
    py = c.bottom - 52
    s += c.rect(c.left, py, c.right - c.left, 44, fill = "teal.box", stroke = "teal", width = 2, rx = 10)
    lab = "PROCHAINE ÉTAPE →"
    s += c.text(c.left + 18, py + 30, lab, size = c.fs(22), bold = True, spacing = 1.2, col = "teal.lead")
    s += c.text(c.left + 30 + c.measure(lab, c.fs(22), bold = True, spacing = 1.2), py + 30,
                PROCHAINE, size = c.fs(22), col = "text")
    if c.measure(lab, c.fs(22), bold = True, spacing = 1.2) + c.measure(PROCHAINE, c.fs(22)) + 50 > c.right - c.left:
        c.warn("bandeau : texte trop long")
    if max(yb, oy + 50) > py - 10:
        c.warn("bandeau : le contenu descend jusqu'à %d, bandeau à %d" % (max(yb, oy + 50), py))

    return s + c.footer(["carte : les 5 règles de policy.rego · chiffres : go run ./training, ./ask, 40 graines, 24 blocs cachés, seed 1337"], prov = "interne")
```

> **À dire** Le message : le modèle n'exécute pas la règle, il interpole à partir des
> exemples que la règle a étiquetés. Sur les cas qu'on lui a cachés, il ne recopie pas
> Rego : il le contredit parfois, mais sans fiabilité. Les trois colonnes déroulent cet
> argument : Rego fournit 1 152 exemples étiquetés, le modèle apprend par l'exemple, et sur les
> cas cachés on voit ce que donne l'interpolation.
>
> Le MFA (authentification multifacteur) est un des six champs de chaque
> requête : le modèle le voit dans tous les exemples, à l'entraînement comme à
> l'inférence. Dans la politique, seule la règle 3 l'utilise (un employee écrit dans son
> département avec MFA) : sur la carte, c'est la colonne É d'employee, verte seulement du
> côté MFA oui. Les 12 paires entourées en magenta sont les seuls cas, sur 576, où le
> MFA change la réponse de Rego. Pour une lecture, il ne change rien.
>
> À gauche, la carte complète : les 1 152 requêtes possibles, une case par
> requête. Colonnes : rôle, action, MFA ; lignes : classification, département du
> document, département du demandeur. En vert ce que Rego autorise. On lit la
> politique dans les motifs : la colonne admin pleine, la bande « public » en lecture,
> la diagonale « même département » chez employee. En rouge, le coin que la règle 5
> ouvre : contractor, lecture, interne, autre département. Ces 24 cases sont retirées
> de l'entraînement.

Au centre, l'objectif du modèle : prédire la couleur d'une case, ALLOW ou DENY, sans
connaître la règle. Deux temps. APPRENDRE : il voit les 1 128 cases vertes et grises,
étiquetées par Rego ; la perte passe de 0,69 à 0,14 en cinq époques, 0,0002 à 400
(trois points mesurés). INTERROGER : on lui pose une case rouge qu'il n'a jamais vue (la bande reprend le
coin de la carte : les 24 cases rouges estompées, sauf celle qu'on pose),
découpée en six tokens ; l'attention croise les champs (les arcs sont un schéma, pas
les poids appris). La SORTIE est un couple de probabilités, P(DENY) et P(ALLOW), qui
alimente la jauge de droite.

À droite, sur les cas cachés, deux temps. SORTIE : ce que le modèle produit, une P(DENY) pour chacune des 24 cases cachées ; la case posée sort à 95,2 % (DENY), sa jumelle avec MFA à 1,3 % (ALLOW), et Rego dit ALLOW sur les 24. Puis le bilan du POC. Il valide trois choses : un transformer écrit à la main apprend une politique à partir de ses seuls exemples (1 128 / 1 128) ; il rend une sortie typée, classe et probabilité, comme Jev ; et il repère une incohérence entre règles voisines. Mesure : on a caché, un par un, les 24 blocs de la même forme que la faille (rôle, action, classification, autre département) ; le modèle en complète 22 sans erreur, et ne conteste que deux blocs, exactement les règles 2 et 5, qui se contredisent (employee : DENY, contractor : ALLOW). Il ne valide pas trois autres : dire laquelle est la faille (cacher la règle 2, pourtant correcte, donne 24 désaccords sur 24, plus que la faille) ; une confiance fiable (sûr de lui à plus de 95 %, mais le MFA ou la graine renversent la réponse) ; un usage en gateway (il faudrait l'entraîner sur des décisions de relecteurs, pas sur Rego).
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

En bas, la prochaine étape. Ce POC ne teste pas Jev : il imite son mode d'emploi avec un
mini-transformer fait main. Puisque le modèle repère des incohérences sans savoir
trancher, l'étape suivante est de les remonter à un relecteur, d'apprendre de ses
décisions (plutôt que d'imiter Rego), puis comparer Jev à une baseline simple sur le rappel des cas suspects, la
calibration de sa confiance et l'invariance aux champs sans rôle, comme le MFA.

## Jev en une page

```starlark
# One-slider : pensé pour le format lecture (plancher 22 px).
# Rangée du haut : l'histoire en quatre temps. Rangée du bas : la preuve et la leçon.

PTS = [(1, 0.69), (5, 0.14), (400, 0.0002)]

def panneau(c, x, y, w, h, tone, num, titre):
    s = c.rect(x, y, w, h, fill = tone + ".box", stroke = tone, width = 2, rx = 18)
    s += c.circle(x + 40, y + 42, 22, fill = tone + ".pill0")
    s += c.text(x + 40, y + 51, num, size = c.label, bold = True, col = "white", anchor = "middle")
    s += c.text(x + 76, y + 52, titre, size = c.label, bold = True, spacing = 1.5, col = tone + ".lead")
    return s

def barre(c, x, y, w, part, tone, texte, sous):
    s = c.rect(x, y, w, 40, fill = "box", stroke = tone, width = 1.5, rx = 8)
    s += c.rect(x, y, int(w * part), 40, fill = tone + ".pill0", rx = 8)
    s += c.text(x + 14, y + 29, texte, size = c.fs(24), bold = True, col = "white")
    s += c.text(x, y + 72, sous, size = c.fs(22), col = "muted")
    return s

def render(step, c):
    s, y = c.head(titre = "Appris sur Rego, le transformer conteste une partie de sa faille",
                  repere = "Jev en une page", tone = "bleu")
    gap = 44
    w = (c.right - c.left - 3 * gap) // 4
    ya = y + 24
    ha = 440
    xs = [c.left + i * (w + gap) for i in range(4)]
    tones = ["violet", "bleu", "bleu", "rouge"]
    for i in range(3):
        s += c.arrow(xs[i] + w + 6, ya + ha // 2, xs[i + 1] - 8, ya + ha // 2, col = tones[i + 1], width = 3, head = 10)

    # 1 · la règle
    x = xs[0]
    g = panneau(c, x, ya, w, ha, "violet", "1", "LA RÈGLE")
    g += c.text(x + 24, ya + 110, ["policy.rego, 5 règles,", "exécutées par OPA"], size = c.body, col = "text", lh = 36)
    g += c.text(x + 24, ya + 210, "Règles 2 et 3", size = c.fs(24), bold = True, col = "violet.lead")
    g += c.text(x + 24, ya + 246, ["employee : même", "département exigé"], size = c.fs(24), col = "text", lh = 32)
    g += c.rect(x + 16, ya + 308, w - 32, 116, fill = "rouge.box", stroke = "rouge", width = 2, rx = 12)
    g += c.text(x + 32, ya + 346, "Règle 5 ⚠ la faille", size = c.fs(24), bold = True, col = "rouge.lead")
    g += c.text(x + 32, ya + 380, ["contractor lit tout internal,", "sans vérifier le département"], size = c.fs(22), col = "text", lh = 28)
    s += c.group(g, box = [x, ya, w, ha])

    # 2 · le dataset
    x = xs[1]
    g = panneau(c, x, ya, w, ha, "bleu", "2", "LE DATASET")
    g += c.text(x + w // 2, ya + 160, "1 152", size = c.fs(72), bold = True, col = "bleu.lead", anchor = "middle")
    g += c.text(x + w // 2, ya + 202, "requêtes énumérées", size = c.body, col = "text", anchor = "middle")
    g += c.text(x + w // 2, ya + 236, "étiquetées par OPA", size = c.fs(24), col = "muted", anchor = "middle")
    g += c.rect(x + 24, ya + 268, w - 48, 64, fill = "teal.box", stroke = "teal", width = 2, rx = 10)
    g += c.text(x + 44, ya + 310, "1 128 apprises", size = c.body, bold = True, col = "teal.lead")
    g += c.rect(x + 24, ya + 348, w - 48, 64, fill = "rouge.box", stroke = "rouge", width = 2, rx = 10)
    g += c.text(x + 44, ya + 390, "24 cas-faille cachés", size = c.body, bold = True, col = "rouge.lead")
    s += c.group(g, box = [x, ya, w, ha])

    # 3 · le modèle
    x = xs[2]
    g = panneau(c, x, ya, w, ha, "bleu", "3", "LE MODÈLE")
    etapes = [
        ("6 tokens champ=valeur", "text"),
        ("embeddings × 16", "text"),
        ("self-attention", "magenta.lead"),
        ("moyenne + linéaire", "text"),
        ("softmax : 2 probabilités", "text"),
    ]
    yy = ya + 84
    for i, (t, col) in enumerate(etapes):
        hot = col != "text"
        g += c.rect(x + 24, yy, w - 48, 50, fill = "magenta.box" if hot else "box",
                    stroke = "magenta" if hot else "bleu", width = 2 if hot else 1.2, rx = 10)
        g += c.text(x + w // 2, yy + 34, t, size = c.fs(24), bold = hot, col = col, anchor = "middle")
        if i < len(etapes) - 1:
            g += c.arrow(x + w // 2, yy + 52, x + w // 2, yy + 68, col = "bleu", width = 2, head = 7)
        yy += 70
    s += c.group(g, box = [x, ya, w, ha])

    # 4 · le verdict
    x = xs[3]
    g = panneau(c, x, ya, w, ha, "rouge", "4", "LE CAS JAMAIS VU")
    g += c.text(x + 24, ya + 108, ["contractor · read · internal", "doc RH ← demandeur marketing"], size = c.fs(22), col = "text", lh = 30)
    bw = w - 48
    g += barre(c, x + 24, ya + 172, bw, 1.0, "violet", "ALLOW", "Rego : règle 5, mécanique")
    g += barre(c, x + 24, ya + 262, bw, 0.952, "bleu", "DENY 95,2 %", "Jev : avec attention")
    g += barre(c, x + 24, ya + 352, bw, 0.858, "teal", "DENY 85,8 %", "sac de mots : règle 5 ratée")
    s += c.group(g, box = [x, ya, w, ha])

    # rangée du bas
    yb = ya + ha + 40
    hb = c.bottom - 40 - yb

    # la perte
    lw = xs[1] + w - c.left
    g = c.text(c.left, yb + 28, "LA PERTE S’EFFONDRE", size = c.label, bold = True, spacing = 1.5, col = "teal.lead")
    gx0 = c.left + 70
    gx1 = c.left + lw - 30
    gy0 = yb + 70
    gy1 = yb + hb - 40
    lmax = math.log(400.0)

    def px(e):
        return gx0 + (gx1 - gx0) * math.log(float(e)) / lmax

    def py(l):
        return gy1 - (gy1 - gy0) * l / 0.69

    g += c.line(gx0, gy1, gx1, gy1, col = "axis", width = 2)
    g += c.line(gx0, gy0 - 10, gx0, gy1, col = "axis", width = 2)
    g += c.text(gx0 - 12, gy0 + 8, "0,69", size = c.fs(22), col = "muted", anchor = "end")
    g += c.text(gx0 - 12, gy1 + 8, "0", size = c.fs(22), col = "muted", anchor = "end")
    g += c.text(gx1, gy1 + 34, "époques (log)", size = c.fs(22), col = "muted", anchor = "end")
    noms = [("ép. 1 : hasard", 20, 8, "start"), ("ép. 5 : imite Rego", 20, -24, "start"), ("ép. 400 : 0,0002", 0, -28, "end")]
    for i, (e, l) in enumerate(PTS):
        if i > 0:
            pe, pl = PTS[i - 1]
            g += c.line(px(pe), py(pl), px(e), py(l), col = "teal", width = 4, cap = "round")
    for i, (e, l) in enumerate(PTS):
        g += c.circle(px(e), py(l), 9, fill = "bg", stroke = "teal", width = 4)
        t, dx, dy, a = noms[i]
        g += c.text(px(e) + dx, py(l) + dy, t, size = c.fs(22), bold = True, col = "teal.lead", anchor = a)
    s += c.group(g, box = [c.left, yb, lw, hb])

    # la mesure honnête
    x = xs[2]
    g = c.text(x, yb + 28, "SUR LES 24 CAS-FAILLE", size = c.label, bold = True, spacing = 1.5, col = "magenta.lead")
    chiffres = [
        ("11 / 24", "refusés par le modèle", "teal"),
        ("8 / 8", "cas légitimes gardés", "bleu"),
        ("0 à 21", "selon la graine (40)", "magenta"),
    ]
    for i, (lead, lab, tone) in enumerate(chiffres):
        yy = yb + 84 + i * 70
        g += c.text(x, yy, lead, size = c.fs(40), bold = True, col = tone + ".lead")
        g += c.text(x + 170, yy - 4, lab, size = c.fs(24), col = "text")
    s += c.group(g, box = [x, yb, w, hb])

    # la leçon
    x = xs[3]
    g = c.rect(x, yb, w, hb, fill = "teal.box", stroke = "teal", width = 2, rx = 18)
    g += c.text(x + 24, yb + 44, "LA LEÇON", size = c.label, bold = True, spacing = 1.5, col = "teal.lead")
    g += c.text(x + 24, yb + 92, ["Rego définit la politique.", "Le modèle donne", "un indice à vérifier."], size = c.body, bold = True, col = "text", lh = 36)
    g += c.text(x + 24, yb + 222, ["go run ./training", "go run ./ask -demo"], size = c.fs(22), col = "muted", lh = 30)
    s += c.group(g, box = [x, yb, w, hb])

    return s + c.footer(["go run ./training, ./ask (lr = 0,02, seed 1337) · 40 graines"], prov = "interne")
```

> **À dire** Une page pour tout le projet. En haut, l'histoire : la politique Rego et
> sa faille volontaire (règle 5) ; le moteur OPA étiquette les 1 152 requêtes
> possibles, dont 24 cas-faille jamais montrés au modèle ; un mini-transformer écrit à
> la main apprend sur les 1 128 autres ; sur un cas-faille, Rego dit ALLOW et le
> modèle DENY à 95,2 %.

En bas, la preuve et ses limites : la perte passe de 0,69 à 0,14 en cinq époques ; sur
les 24 cas-faille, 11 sont refusés, la règle 5 légitime est conservée sur 8/8, et le
résultat varie selon la graine. Sur 40 graines (1 à 40), le nombre de refus va de 0 à 21, médiane 9 ; la graine 1337 en donne 11, un peu au-dessus de la médiane. REFERENCE.md annonçait « 0 à 19 » : non reproduit tel quel. La graine 6 refuse 24/24 mais rate la règle 5 (2/8), et 7 graines sur 40 n'atteignent pas 100 % sur l'entraînement. Le sac de mots, sans attention, dit aussi DENY sur ce cas (85,8 %), mais ce n'est pas une généralisation : il n'a pas appris la règle 5 (0/8 sur les cas contractor même département, DENY à 86,1 % sur contractor hr/hr) et refuse les 24 cas-faille, comme tout contractor en lecture interne. Mesuré : go run ./ask, et un réentraînement à seed 1337. Attention, 95,2 % vaut pour une seule requête : la même avec MFA oui donne ALLOW (1,3 %), et sur les 24 cas-faille le modèle tranche presque toujours à plus de 95 % dans un sens ou dans l'autre. Rego reste la source de vérité ; le modèle donne un indice à vérifier, pas un détecteur.

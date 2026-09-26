// Commande ask : pose une requête d'accès au modèle entraîné et confronte sa
// prédiction à la réponse littérale de Rego (policy.Request.Decide).
//
// C'est le banc de test visuel du dépôt : là où Rego exécute mécaniquement
// ses règles et ne peut ni douter ni signaler sa propre faille, le modèle
// statistique généralise à partir des exemples qu'il a vus. Le CLI rend cette
// différence visible, requête par requête.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/owulveryck/jevgo/model"
	"github.com/owulveryck/jevgo/policy"
)

const (
	attnWeightsPath = "weights/policy-attention.json"
	baseWeightsPath = "weights/policy-baseline.json"
)

func main() {
	var (
		demo    = flag.Bool("demo", false, "joue une série de requêtes pré-choisies en tableau")
		role    = flag.String("role", "", "rôle du demandeur ("+strings.Join(policy.Roles, "|")+")")
		action  = flag.String("action", "", "action demandée ("+strings.Join(policy.Actions, "|")+")")
		classif = flag.String("classification", "", "classification du document ("+strings.Join(policy.Classifications, "|")+")")
		resDept = flag.String("resource-dept", "", "département du document ("+strings.Join(policy.Departments, "|")+")")
		reqDept = flag.String("requester-dept", "", "département du demandeur ("+strings.Join(policy.Departments, "|")+")")
		mfa     = flag.Bool("mfa", false, "le demandeur est authentifié en MFA")
	)
	flag.Parse()

	attn, err := model.Load(attnWeightsPath)
	if err != nil {
		log.Fatalf("impossible de charger le modèle avec attention : %v\n(lancez d'abord 'go run ./training')", err)
	}
	baseline, err := policy.LoadBagOfEmbeddings(baseWeightsPath)
	if err != nil {
		log.Fatalf("impossible de charger le modèle de référence : %v\n(lancez d'abord 'go run ./training')", err)
	}

	if *demo {
		runDemo(attn, baseline)
		return
	}

	if *role == "" || *action == "" || *classif == "" || *resDept == "" || *reqDept == "" {
		fmt.Fprintln(os.Stderr, "Il manque des champs pour décrire la requête.")
		fmt.Fprintln(os.Stderr, "Exemple :")
		fmt.Fprintln(os.Stderr, "  go run ./ask -role=contractor -action=read -classification=internal \\")
		fmt.Fprintln(os.Stderr, "               -resource-dept=hr -requester-dept=marketing")
		fmt.Fprintln(os.Stderr, "  go run ./ask -demo")
		os.Exit(2)
	}

	req := policy.Request{
		Role:                   *role,
		Action:                 *action,
		ResourceClassification: *classif,
		ResourceDepartment:     *resDept,
		RequesterDepartment:    *reqDept,
		MFA:                    *mfa,
	}
	printDetail(evaluate(attn, baseline, req))
}

type result struct {
	req        policy.Request
	rego       bool
	attnClass  int
	attnProbs  []float64
	baseClass  int
	baseProbs  []float64
	unknown    []string
	divergence bool
}

func evaluate(attn *model.Model, baseline *policy.BagOfEmbeddings, req policy.Request) result {
	attnClass, attnProbs := attn.Forward(attn.Vocab.Encode(req.Tokens()))
	baseClass, baseProbs := baseline.Forward(baseline.Vocab.Encode(req.Tokens()))
	rego := req.Decide()
	return result{
		req:        req,
		rego:       rego,
		attnClass:  attnClass,
		attnProbs:  attnProbs,
		baseClass:  baseClass,
		baseProbs:  baseProbs,
		unknown:    unknownFields(req),
		divergence: attnClass != boolToClass(rego),
	}
}

// --- Mode requête unique (détail) -----------------------------------------

func printDetail(r result) {
	fmt.Printf("Requête : %s\n\n", describe(r.req))

	fmt.Println("Résultat")
	fmt.Printf("  Rego (règle littérale)        : %s\n", label(boolToClass(r.rego)))
	fmt.Printf("  Jev (avec attention)          : %s%s\n", label(r.attnClass), conf(r.attnProbs, r.attnClass))
	fmt.Printf("  Sac de mots (sans attention)  : %s%s\n", label(r.baseClass), conf(r.baseProbs, r.baseClass))

	if r.divergence {
		fmt.Printf("\n  ⚠ Le modèle avec attention diverge de Rego.\n")
	}
	fmt.Printf("\nAnalyse\n%s\n", analysis(r))
}

// --- Mode démo (tableau) --------------------------------------------------

func runDemo(attn *model.Model, baseline *policy.BagOfEmbeddings) {
	presets := []policy.Request{
		{Role: "employee", Action: "read", ResourceClassification: "public",
			ResourceDepartment: "engineering", RequesterDepartment: "engineering", MFA: false},
		{Role: "guest", Action: "delete", ResourceClassification: "confidential",
			ResourceDepartment: "finance", RequesterDepartment: "marketing", MFA: false},
		{Role: "contractor", Action: "read", ResourceClassification: "internal",
			ResourceDepartment: "hr", RequesterDepartment: "hr", MFA: false},
		{Role: "contractor", Action: "read", ResourceClassification: "internal",
			ResourceDepartment: "hr", RequesterDepartment: "marketing", MFA: false},
		{Role: "intern", Action: "read", ResourceClassification: "confidential",
			ResourceDepartment: "hr", RequesterDepartment: "engineering", MFA: false},
	}

	fmt.Println("--- Démo : Rego (règle littérale) vs modèles entraînés ---")
	fmt.Println("    (les lignes 4 et 5 portent sur des cas tenus à l'écart ou hors vocabulaire)")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "REQUETE\tREGO\tJEV (attention)\tSAC DE MOTS\tNOTE")
	divergences, unknownSeen := 0, 0
	for _, req := range presets {
		res := evaluate(attn, baseline, req)
		if res.divergence {
			divergences++
		}
		if len(res.unknown) > 0 {
			unknownSeen++
		}
		fmt.Fprintf(w, "%s\t%s\t%s%s\t%s%s\t%s\n",
			describe(req), label(boolToClass(res.rego)),
			label(res.attnClass), conf(res.attnProbs, res.attnClass),
			label(res.baseClass), conf(res.baseProbs, res.baseClass),
			note(res))
	}
	w.Flush()

	fmt.Printf("\n%d/%d requêtes où le transformer diverge de la règle littérale", divergences, len(presets))
	if unknownSeen > 0 {
		fmt.Printf(" — %d avec une valeur hors vocabulaire (token UNK)", unknownSeen)
	}
	fmt.Println(".")
	fmt.Println()
	fmt.Println("Lecture :")
	fmt.Println("  - Lignes 1-2 : cas ordinaires, Rego et le transformer s'accordent.")
	fmt.Println("  - Ligne 3 : contractor + lecture + interne de SON département. La règle 5")
	fmt.Println("    est présente à l'entraînement ; le transformer l'apprend (ALLOW), alors")
	fmt.Println("    que le sac de mots, qui ne mélange pas les positions entre tokens, ne la")
	fmt.Println("    capte pas.")
	fmt.Println("  - Ligne 4 : même requête, mais lue dans un AUTRE département. Ces 24 cas")
	fmt.Println("    ont été retirés du dataset. Rego applique mécaniquement la règle 5 et dit")
	fmt.Println("    ALLOW ; le transformer généralise la correspondance de département vue")
	fmt.Println("    ailleurs et prédit DENY. C'est la réponse que Rego ne peut PAS donner :")
	fmt.Println("    il ne sait pas signaler que sa propre règle 5 est suspecte.")
	fmt.Println("  - Ligne 5 : rôle inconnu (intern -> token UNK). Rego retombe sur son cas")
	fmt.Println("    par défaut (DENY), le modèle continue avec le reste des champs.")
	fmt.Println("Lancez une requête précise avec -role=... pour le détail et les probabilités.")
}

// --- Aides ----------------------------------------------------------------

func describe(r policy.Request) string {
	return fmt.Sprintf("%s %s %s %s/%s mfa=%v",
		r.Role, r.Action, r.ResourceClassification,
		r.ResourceDepartment, r.RequesterDepartment, r.MFA)
}

func note(r result) string {
	var notes []string
	if r.req.IsLoophole() {
		notes = append(notes, "faille regle 5")
	}
	if len(r.unknown) > 0 {
		notes = append(notes, "hors vocabulaire: "+strings.Join(r.unknown, ","))
	}
	if len(notes) == 0 {
		return "-"
	}
	return strings.Join(notes, " ; ")
}

func analysis(r result) string {
	var b strings.Builder

	switch {
	case r.req.IsLoophole() && r.attnClass == model.LabelDeny:
		b.WriteString("  Requête-faille : la règle 5 autorise ce contractor à lire un document\n")
		b.WriteString("  interne d'un autre département, sans vérifier la correspondance — c'est\n")
		b.WriteString("  le cas volontairement retiré de l'entraînement.\n")
		b.WriteString("  Rego applique la règle telle qu'écrite et répond ALLOW. Le transformer,\n")
		b.WriteString("  lui, a appris la règle 5 sur les cas même-département restés dans le\n")
		b.WriteString("  dataset, et a vu par ailleurs que les règles 2 et 3 exigent la\n")
		b.WriteString("  correspondance de département : il généralise et répond DENY. C'est\n")
		b.WriteString("  exactement ce que Rego ne sait pas faire — douter de sa propre règle.\n")
	case r.req.IsLoophole():
		b.WriteString("  Requête-faille : Rego répond ALLOW (règle 5, sans contrôle de\n")
		b.WriteString("  département). Ici le transformer ne diverge pas — sur ce cas, il n'a\n")
		b.WriteString("  pas généralisé la contrainte de département. Relancez l'entraînement\n")
		b.WriteString("  ou comparez avec le sac de mots pour voir l'effet de l'attention.\n")
	case len(r.unknown) > 0:
		b.WriteString("  Valeur(s) hors vocabulaire : " + strings.Join(r.unknown, ", ") + ".\n")
		b.WriteString("  Rego ne connaît que ses règles : toute valeur non prévue tombe dans le\n")
		b.WriteString("  cas par défaut (DENY). Le modèle, lui, encode ces tokens comme UNK et\n")
		b.WriteString("  continue de raisonner sur les autres champs — d'où une probabilité\n")
		b.WriteString("  parfois surprenante là où Rego est catégorique.\n")
	case r.divergence:
		b.WriteString("  Le modèle diverge de Rego. Sur une requête ordinaire, c'est le signe\n")
		b.WriteString("  qu'il interpole à partir de ce qu'il a vu plutôt que d'exécuter la\n")
		b.WriteString("  règle exacte. La confiance affichée indique à quel point il hésite.\n")
	default:
		b.WriteString("  Les deux modèles s'accordent avec la règle littérale. Aucune\n")
		b.WriteString("  généralisation notable sur cette requête.\n")
	}
	return b.String()
}

func conf(probs []float64, class int) string {
	if len(probs) <= class {
		return ""
	}
	return fmt.Sprintf(" (%.1f%%)", probs[class]*100)
}

func label(class int) string {
	if class == model.LabelAllow {
		return "ALLOW"
	}
	return "DENY"
}

func boolToClass(b bool) int {
	if b {
		return model.LabelAllow
	}
	return model.LabelDeny
}

func unknownFields(r policy.Request) []string {
	var out []string
	if !contains(policy.Roles, r.Role) {
		out = append(out, "role="+r.Role)
	}
	if !contains(policy.Actions, r.Action) {
		out = append(out, "action="+r.Action)
	}
	if !contains(policy.Classifications, r.ResourceClassification) {
		out = append(out, "classification="+r.ResourceClassification)
	}
	if !contains(policy.Departments, r.ResourceDepartment) {
		out = append(out, "resource_dept="+r.ResourceDepartment)
	}
	if !contains(policy.Departments, r.RequesterDepartment) {
		out = append(out, "requester_dept="+r.RequesterDepartment)
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// Package policy fournit le domaine d'exemple « contrôle d'accès aux
// documents » : une structure de requête, l'évaluateur qui reproduit très
// exactement policy.rego, et la sérialisation en tokens consommables par
// jevgo/model.
package policy

import "strconv"

// Domaines de valeurs possibles pour chaque champ — utilisés à la fois
// pour énumérer le dataset et pour construire le vocabulaire.
var (
	Roles           = []string{"admin", "employee", "contractor", "guest"}
	Actions         = []string{"read", "write", "delete"}
	Classifications = []string{"public", "internal", "confidential"}
	Departments     = []string{"engineering", "hr", "finance", "marketing"}
)

// Request représente une demande d'accès, telle qu'elle serait envoyée en
// entrée ("input") à la politique Rego équivalente.
type Request struct {
	Role                   string
	Action                 string
	ResourceClassification string
	ResourceDepartment     string
	RequesterDepartment    string
	MFA                    bool
}

// Decide reproduit exactement les règles "allow" de policy.rego. C'est
// l'oracle qui fournit les étiquettes du dataset — dans un système réel,
// on appellerait ici le vrai moteur OPA (CLI `opa eval` ou le SDK Go
// github.com/open-policy-agent/opa/rego) plutôt que de dupliquer la
// logique à la main ; on la duplique ici pour que l'exemple tourne sans
// dépendance ni réseau. Gardez les deux fichiers synchronisés si vous
// modifiez l'un des deux.
func (r Request) Decide() bool {
	if r.Role == "admin" {
		return true // règle 1
	}
	if r.Role == "employee" && r.Action == "read" && r.RequesterDepartment == r.ResourceDepartment {
		return true // règle 2
	}
	if r.Role == "employee" && r.Action == "write" && r.RequesterDepartment == r.ResourceDepartment && r.MFA {
		return true // règle 3
	}
	if r.Action == "read" && r.ResourceClassification == "public" {
		return true // règle 4
	}
	if r.Role == "contractor" && r.Action == "read" && r.ResourceClassification == "internal" {
		return true // règle 5 — la faille
	}
	return false
}

// IsLoophole signale les requêtes qui n'empruntent QUE la règle 5, avec un
// département demandeur différent du département de la ressource — le cas
// qu'un relecteur humain jugerait suspect, même si Decide() répond "allow".
func (r Request) IsLoophole() bool {
	return r.Role == "contractor" &&
		r.Action == "read" &&
		r.ResourceClassification == "internal" &&
		r.RequesterDepartment != r.ResourceDepartment
}

// Tokens sérialise la requête en une séquence de tokens "champ=valeur".
// Chaque token est une unité opaque à part entière : "resource_dept=hr" et
// "requester_dept=hr" sont deux tokens distincts, avec des embeddings
// indépendants — précisément pour empêcher le modèle de détecter une
// correspondance de département en comptant simplement des occurrences
// partagées entre champs, plutôt qu'en apprenant une vraie relation entre
// les deux positions.
func (r Request) Tokens() []string {
	return []string{
		"role=" + r.Role,
		"action=" + r.Action,
		"classification=" + r.ResourceClassification,
		"resource_dept=" + r.ResourceDepartment,
		"requester_dept=" + r.RequesterDepartment,
		"mfa=" + strconv.FormatBool(r.MFA),
	}
}

package policy

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
)

// policyRego embarque policy.rego dans le binaire : la politique est ainsi
// l'unique source de vérité, sans dépendance à un chemin de fichier relatif
// au répertoire d'exécution.
//
//go:embed policy.rego
var policyRego string

// Evaluator exécute la politique Rego (data.access.allow) via le moteur OPA.
// La requête préparée est réutilisée pour toutes les évaluations.
type Evaluator struct {
	pq rego.PreparedEvalQuery
}

// NewEvaluator compile policy.rego une fois pour toutes.
func NewEvaluator(ctx context.Context) (*Evaluator, error) {
	r := rego.New(
		rego.Query("data.access.allow"),
		rego.Module("policy.rego", policyRego),
		rego.SetRegoVersion(ast.RegoV1),
	)
	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("préparation de la politique OPA : %w", err)
	}
	return &Evaluator{pq: pq}, nil
}

// Eval demande à OPA si la requête est autorisée. C'est l'oracle qui
// étiquette le dataset d'entraînement et la référence affichée par le CLI.
func (e *Evaluator) Eval(ctx context.Context, req Request) (bool, error) {
	input := map[string]any{
		"role":                    req.Role,
		"action":                  req.Action,
		"resource_classification": req.ResourceClassification,
		"resource_department":     req.ResourceDepartment,
		"requester_department":    req.RequesterDepartment,
		"mfa":                     req.MFA,
	}
	rs, err := e.pq.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("évaluation OPA : %w", err)
	}
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return false, nil
	}
	allow, ok := rs[0].Expressions[0].Value.(bool)
	if !ok {
		return false, fmt.Errorf("résultat OPA inattendu : %v", rs[0].Expressions[0].Value)
	}
	return allow, nil
}

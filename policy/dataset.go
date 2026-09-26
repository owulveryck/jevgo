package policy

// All énumère l'intégralité de l'espace des requêtes
// (4 × 3 × 3 × 4 × 4 × 2 = 1152 combinaisons) — chaque combinaison de
// role/action/classification/départements/mfa possible exactement une fois.
func All() []Request {
	var out []Request
	for _, role := range Roles {
		for _, action := range Actions {
			for _, classification := range Classifications {
				for _, resourceDept := range Departments {
					for _, requesterDept := range Departments {
						for _, mfa := range []bool{true, false} {
							out = append(out, Request{
								Role:                   role,
								Action:                 action,
								ResourceClassification: classification,
								ResourceDepartment:     resourceDept,
								RequesterDepartment:    requesterDept,
								MFA:                    mfa,
							})
						}
					}
				}
			}
		}
	}
	return out
}

// Split sépare le dataset complet en un ensemble d'entraînement et un
// ensemble « faille » mis de côté : les requêtes où un contractor lit un
// document interne d'un AUTRE département que le sien (24 cas sur 1152).
// Elles sont autorisées par policy.rego (règle 5), mais volontairement
// absentes de l'entraînement, pour observer ce que le modèle prédit sur ce
// cas précis une fois entraîné sans jamais l'avoir vu.
func Split() (train []Request, loophole []Request) {
	for _, req := range All() {
		if req.IsLoophole() {
			loophole = append(loophole, req)
			continue
		}
		train = append(train, req)
	}
	return train, loophole
}

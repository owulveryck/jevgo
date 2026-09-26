package access

default allow = false

# 1. Les administrateurs ont accès à tout.
allow {
	input.role == "admin"
}

# 2. Un employé peut lire les documents de son propre département.
allow {
	input.role == "employee"
	input.action == "read"
	input.requester_department == input.resource_department
}

# 3. Un employé peut écrire dans son propre département, à condition
#    d'être authentifié en MFA.
allow {
	input.role == "employee"
	input.action == "write"
	input.requester_department == input.resource_department
	input.mfa == true
}

# 4. N'importe qui peut lire un document public.
allow {
	input.action == "read"
	input.resource_classification == "public"
}

# 5. FAILLE VOLONTAIRE : un prestataire (contractor) peut lire n'importe
#    quel document "internal", sans aucune vérification de département.
#    Un contractor en mission pour le marketing peut ainsi lire un
#    document RH interne — ce que la règle autorise au sens strict, mais
#    qu'un relecteur humain jugerait suspect.
allow {
	input.role == "contractor"
	input.action == "read"
	input.resource_classification == "internal"
}

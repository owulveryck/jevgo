package main

import (
	"fmt"
	"log"

	"github.com/owulveryck/jevgo/model"
	"github.com/owulveryck/jevgo/policy"
)

const (
	lr              = 0.02
	epochs          = 400
	attnWeightsPath = "weights/policy-attention.json"
	baseWeightsPath = "weights/policy-baseline.json"
)

func main() {
	// 1. Dataset : l'intégralité de l'espace des requêtes, moins les 24
	// requêtes-faille mises de côté. L'étiquette de chaque requête est
	// fournie par policy.Request.Decide(), la traduction fidèle de
	// policy.rego : Rego est l'oracle qui fabrique le dataset supervisé.
	train, loophole := policy.Split()
	fmt.Printf("Exemples d'entraînement : %d — mis de côté (faille) : %d\n", len(train), len(loophole))

	tokenized := make([][]string, len(train))
	labels := make([]int, len(train))
	for i, req := range train {
		tokenized[i] = req.Tokens()
		if req.Decide() {
			labels[i] = model.LabelAllow
		} else {
			labels[i] = model.LabelDeny
		}
	}

	// 2. Vocabulaire reconstruit à partir du corpus de requêtes.
	vocab := model.NewVocab(tokenized)
	fmt.Printf("Vocabulaire : %d tokens\n", vocab.Size())

	// 3. Deux modèles entraînés sur le MÊME dataset : le transformer du
	// package model (embeddings + self-attention) et un sac d'embeddings
	// sans attention, pour comparer ce que l'attention apporte.
	attn := model.New(vocab, 1337)
	baseline := policy.NewBagOfEmbeddings(vocab, model.DModel, 1337)

	for epoch := 1; epoch <= epochs; epoch++ {
		attnLoss, baseLoss := 0.0, 0.0
		for i := range train {
			ids := vocab.Encode(tokenized[i])
			attnLoss += attn.TrainStep(ids, labels[i], lr)
			baseLoss += baseline.TrainStep(ids, labels[i], lr)
		}
		if epoch == 1 || epoch%5 == 0 {
			fmt.Printf("Époque %2d — perte attention : %.4f — perte sans attention : %.4f\n",
				epoch, attnLoss/float64(len(train)), baseLoss/float64(len(train)))
		}
	}

	if err := attn.Save(attnWeightsPath); err != nil {
		log.Fatalf("sauvegarde (attention) : %v", err)
	}
	if err := baseline.Save(baseWeightsPath); err != nil {
		log.Fatalf("sauvegarde (baseline) : %v", err)
	}
	fmt.Printf("\nPoids sauvegardés dans %s et %s\n", attnWeightsPath, baseWeightsPath)
	fmt.Println("Interrogez-les avec : go run ./ask -demo")
}

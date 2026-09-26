package main

import (
	"fmt"
	"log"

	"github.com/owulveryck/jevgo/model"
)

const (
	lr          = 0.1
	epochs      = 300
	weightsPath = "weights/router.json"
)

type sample struct {
	code  string
	label int
}

// Dataset volontairement varié : le motif à apprendre n'est plus la simple
// présence d'un caractère, mais un vrai signal syntaxique (mots-clés,
// opérateurs, structure) qui distingue effectivement Go de TypeScript.
var dataset = []sample{
	{`package main
func main() {
	fmt.Println("hello")
}`, model.LabelGo},
	{`func Add(a int, b int) int {
	return a + b
}`, model.LabelGo},
	{`type Point struct {
	X int
	Y int
}`, model.LabelGo},
	{`for i := 0; i < 10; i++ {
	fmt.Println(i)
}`, model.LabelGo},
	{`if err != nil {
	return nil, err
}`, model.LabelGo},
	{`go func() {
	ch <- compute()
}()`, model.LabelGo},
	{`defer file.Close()`, model.LabelGo},
	{`var wg sync.WaitGroup
wg.Add(1)`, model.LabelGo},
	{`type Reader interface {
	Read(p []byte) (n int, err error)
}`, model.LabelGo},
	{`m := map[string]int{"a": 1}`, model.LabelGo},
	{`package model

import "fmt"

const weightsPath = "weights/router.json"`, model.LabelGo},
	{`switch x := v.(type) {
case int:
	fmt.Println(x)
}`, model.LabelGo},

	{`function add(a: number, b: number): number {
	return a + b;
}`, model.LabelTypeScript},
	{`const greet = (name: string): void => {
	console.log("hello " + name);
};`, model.LabelTypeScript},
	{`interface Point {
	x: number;
	y: number;
}`, model.LabelTypeScript},
	{`for (let i = 0; i < 10; i++) {
	console.log(i);
}`, model.LabelTypeScript},
	{`export class Service {
	constructor(private readonly repo: Repo) {}
}`, model.LabelTypeScript},
	{`async function fetchData(): Promise<void> {
	await fetch("/api");
}`, model.LabelTypeScript},
	{`import { useState } from "react";
const [count, setCount] = useState(0);`, model.LabelTypeScript},
	{`type Result<T> = { ok: true; value: T } | { ok: false; error: string };`, model.LabelTypeScript},
	{`export default function App() {
	return null;
}`, model.LabelTypeScript},
	{`const m: Map<string, number> = new Map();`, model.LabelTypeScript},
	{`try {
	doSomething();
} catch (e) {
	console.error(e);
}`, model.LabelTypeScript},
	{`enum Status {
	Active,
	Inactive,
}`, model.LabelTypeScript},
}

func main() {
	// 1. Tokenisation de tout le corpus, pour construire le vocabulaire.
	tokenized := make([][]string, len(dataset))
	for i, s := range dataset {
		tokenized[i] = model.Tokenize(s.code)
	}
	vocab := model.NewVocab(tokenized)
	fmt.Printf("Vocabulaire : %d tokens\n", vocab.Size())

	// 2. Entraînement du transformer minimal, exemple par exemple (pas de
	// batch ni de padding : on garde la simplicité pédagogique des scripts
	// d'origine, tout en rétropropageant réellement le gradient).
	m := model.New(vocab, 1337)

	for epoch := 1; epoch <= epochs; epoch++ {
		totalLoss := 0.0
		for i, s := range dataset {
			ids := vocab.Encode(tokenized[i])
			totalLoss += m.TrainStep(ids, s.label, lr)
		}
		if epoch == 1 || epoch%50 == 0 {
			fmt.Printf("Époque %3d — perte moyenne : %.4f\n", epoch, totalLoss/float64(len(dataset)))
		}
	}

	if err := m.Save(weightsPath); err != nil {
		log.Fatalf("sauvegarde des poids : %v", err)
	}
	fmt.Printf("\nPoids et vocabulaire sauvegardés dans %s\n", weightsPath)
}

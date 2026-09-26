package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/owulveryck/jevgo/model"
)

const weightsPath = "weights/router.json"

// Extraits de démonstration : à remplacer par n'importe quel code que l'on
// veut faire classifier.
var demoSnippets = []string{
	`func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}`,
	`export function divide(a: number, b: number): number {
	if (b === 0) {
		throw new Error("division by zero");
	}
	return a / b;
}`,
}

func main() {
	m, err := model.Load(weightsPath)
	if err != nil {
		log.Fatalf("impossible de charger les poids : %v\n(lancez d'abord 'go run ./training')", err)
	}

	fmt.Println("--- Classification Go vs TypeScript ---")
	for _, code := range demoSnippets {
		classify(m, code)
	}

	// Mode interactif optionnel : lire du code depuis l'entrée standard,
	// terminé par une ligne "###".
	if len(os.Args) > 1 && os.Args[1] == "-stdin" {
		fmt.Println("\nCollez un extrait de code, terminez par une ligne '###' :")
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "###" {
				break
			}
			lines = append(lines, line)
		}
		classify(m, strings.Join(lines, "\n"))
	}
}

func classify(m *model.Model, code string) {
	tokens := model.Tokenize(code)
	ids := m.Vocab.Encode(tokens)
	class, probs := m.Forward(ids)

	label := "Go"
	if class == model.LabelTypeScript {
		label = "TypeScript"
	}
	fmt.Printf("\n%s\n-> %s (confiance %.1f%%, distribution [Go: %.3f, TS: %.3f])\n",
		preview(code), label, probs[class]*100, probs[0], probs[1])
}

func preview(code string) string {
	line := strings.SplitN(strings.TrimSpace(code), "\n", 2)[0]
	if len(line) > 50 {
		line = line[:50] + "..."
	}
	return "« " + line + " » ..."
}

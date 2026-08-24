import type { SandboxLanguage } from "@/types/api"

export const STARTER_TEMPLATES: Record<SandboxLanguage, string> = {
  go: `package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Solve implements your algorithmic solution
func Solve(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	// TODO: implement logic here
	return strings.Join(lines, " ")
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	fmt.Println(Solve(lines))
}
`,
  python: `import sys

def solve():
    """Implement your algorithmic solution here"""
    lines = sys.stdin.read().strip().split()
    if not lines:
        return
    # TODO: implement logic here
    print(" ".join(lines))

if __name__ == "__main__":
    solve()
`,
  typescript: `function solve(input: string): string {
    // TODO: implement your solution
    return input.trim();
}

const readline = require("readline");
const rl = readline.createInterface({ input: process.stdin });
let lines: string[] = [];

rl.on("line", (line: string) => lines.push(line));
rl.on("close", () => {
    console.log(solve(lines.join("\\n")));
});
`,
  javascript: `function solve(input) {
    // TODO: implement your solution
    return input.trim();
}

const readline = require("readline");
const rl = readline.createInterface({ input: process.stdin });
let lines = [];
rl.on("line", (line) => lines.push(line));
rl.on("close", () => {
    console.log(solve(lines.join("\\n")));
});
`,
}

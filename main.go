package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	celext "github.com/google/cel-go/ext"
)

func main() {
	c := make(chan struct{}, 0)

	js.Global().Set("cel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) != 2 {
			return ""
		}

		req := args[0].String()
		expression := args[1].String()

		r, body, err := readHTTP(req)
		if err != nil {
			return fmt.Errorf("error reading HTTP file: %s", err)
		}

		evalContext, err := makeEvalContext(body, r.Header, r.URL.String())
		if err != nil {
			return fmt.Sprintf("error making eval context: %s", err)
		}

		mapStrDyn := decls.NewMapType(decls.String, decls.Dyn)
		env, err := cel.NewEnv(
			celext.Strings(),
			celext.Encoders(),
			cel.Declarations(
				decls.NewVar("body", mapStrDyn),
				decls.NewVar("header", mapStrDyn),
				decls.NewVar("requestURL", decls.String),
			))
		if err != nil {
			log.Fatal(err)
		}

		parsed, issues := env.Parse(expression)
		if issues != nil && issues.Err() != nil {
			return fmt.Sprintf("failed to parse expression %#v: %s", expression, issues.Err())
		}

		checked, issues := env.Check(parsed)
		if issues != nil && issues.Err() != nil {
			return fmt.Sprintf("expression %#v check failed: %s", expression, issues.Err())
		}

		prg, err := env.Program(checked)
		if err != nil {
			return fmt.Sprintf("expression %#v failed to create a Program: %s", expression, err)
		}

		out, _, err := prg.Eval(evalContext)
		if err != nil {
			return fmt.Sprintf("expression %#v failed to evaluate: %s", expression, err)
		}

		return fmt.Sprintf("%v", out)
	}))

	<-c
}

func readHTTP(input string) (*http.Request, []byte, error) {
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		return nil, nil, fmt.Errorf("error reading request: %s", err)
	}
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading HTTP body: %s", err)
	}
	return req, body, nil
}

func makeEvalContext(body []byte, h http.Header, url string) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the body as JSON: %s", err)
	}
	return map[string]interface{}{
		"body":       jsonMap,
		"header":     h,
		"requestURL": url,
	}, nil
}

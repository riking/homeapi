package factoid

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
)

type OutputFlags struct {
	NoReply     bool
	Say         bool
	Pre         bool
	SideEffects bool
}

// RunFactoid
//
// Returns the following errors (make sure to use errors.Cause()):
//
//   factoidName, ErrNoSuchFactoid - Factoid not found
//   ErrUser - Something was wrong with the input. Not enough args, recursion limit reached.
func (mod *FactoidModule) RunFactoid(ctx context.Context, line []string, of *OutputFlags, source marvin.ActionSource) (result string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = util.PCall(func() error {
		result, err = mod.exec_alias(ctx, line, of, source)
		return err
	})
	return
}

func (mod *FactoidModule) exec_alias(ctx context.Context, origLine []string, of *OutputFlags, actionSource marvin.ActionSource) (string, error) {
	// Handle alias recursion

	var recursionCheck []string
	line := origLine
	for {
		factoid := line[0]
		args := line[1:]

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		info, err := mod.GetFactoidBare(factoid, actionSource.ChannelID())
		if err == ErrNoSuchFactoid {
			return factoid, err
		}

		if strings.HasPrefix(info.RawSource, "{alias}") {
			_, tokens := info.Tokens()
			str, err := mod.exec_processTokens(tokens, args, actionSource)
			if err != nil {
				return "", err
			}
			for _, v := range recursionCheck {
				if v == str {
					return "", ErrUser{errors.Errorf("Recursion limit reached. Factoid: [%s]", str)}
				}
			}
			recursionCheck = append(recursionCheck, str)
			line = strings.Split(str, " ")
			continue
		}
		return mod.exec_parse(ctx, info, info.RawSource, args, of, actionSource)
	}
}

func (mod *FactoidModule) exec_parse(ctx context.Context, f *Factoid, raw string, args []string, of *OutputFlags, actionSource marvin.ActionSource) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	var directives []DirectiveToken
	var tokens []Token
	if f != nil {
		directives, tokens = f.Tokens()
	} else {
		var remainder string
		directives, remainder = Directives(raw)
		tokens = mod.Tokenize(remainder)
	}

directives_loop:
	for _, v := range directives {
		switch v.Directive {
		case "say":
			of.Say = true
		case "pre":
			of.Pre = true
			break directives_loop
		case "noreply":
			of.NoReply = true
			return "", nil
		case "skip":
			break directives_loop
		case "lua":
			luaSource, err := mod.exec_processTokens(tokens, args, actionSource)
			if err != nil {
				return "", err
			}
			result, err := RunFactoidLua(ctx, mod, luaSource, args, of, actionSource)
			if err != nil {
				return "", err
			}
			tokens := mod.Tokenize(result)
			return mod.exec_processTokens(tokens, args, actionSource)
		}
	}
	return mod.exec_processTokens(tokens, args, actionSource)
}

func (mod *FactoidModule) exec_processTokens(tokens []Token, args []string, actionSource marvin.ActionSource) (string, error) {
	var buf bytes.Buffer
	for _, v := range tokens {
		str, err := v.Run(mod, args, actionSource)
		if err != nil {
			return "", errors.Wrapf(err, "tokens")
		}
		buf.WriteString(str)
	}
	return buf.String(), nil
}

func Directives(source string) ([]DirectiveToken, string) {
	var directives []DirectiveToken

	// Get all directives
	// Directives are anchored to beginning of factoid
	m := DirectiveTokenRgx.FindStringSubmatchIndex(source)
	for m != nil {
		directive := source[m[2]:m[3]]
		directives = append(directives, DirectiveToken{Directive: directive})
		source = source[m[1]:]
		m = DirectiveTokenRgx.FindStringSubmatchIndex(source)
	}
	return directives, source
}

// Tokens can panic, make sure it gets called with PCall() before saving a factoid to the database.
func (fi *Factoid) Tokens() ([]DirectiveToken, []Token) {
	directives, source := Directives(fi.RawSource)

	fi.tokenize.Do(func() {
		tokens := fi.mod.collectTokenize(source)
		fi.tokens = tokens
	})
	return directives, fi.tokens
}

func (mod *FactoidModule) Tokenize(source string) []Token {
	return mod.collectTokenize(source)
}

func (mod *FactoidModule) collectTokenize(source string) []Token {
	var tokens []Token
	tokenCh := make(chan Token)

	go mod.tokenize(source, false, tokenCh)
	for v := range tokenCh {
		tokens = append(tokens, v)
	}
	return tokens
}

func (mod *FactoidModule) tokenize(source string, recursed bool, tokenCh chan<- Token) {
	// Function directives
	m := FunctionTokenRgx.FindStringSubmatchIndex(source)
	for m != nil {
		fmt.Println(m)
		if source[m[2]:m[3]] != "" {
			m[0]++
		}
		mod.tokenize(source[:m[0]], true, tokenCh)
		funcName := source[m[4]:m[5]]
		funcInfo, ok := mod.functions[funcName]
		if !ok {
			tokenCh <- TextToken{Text: "$" + funcName}
			source = source[m[5]:]
		} else {
			start := m[5]
			end := -999
			nesting := 0
			for i := start; i < len(source); i++ {
				var b byte = source[i]
				if b == '\\' {
					i++
					continue
				} else if b == '(' {
					nesting++
				} else if b == ')' {
					nesting--
				}
				if nesting == 0 {
					end = i
					break
				}
			}
			if end == -999 {
				panic(ErrSource{errors.Errorf("Unclosed function named '%s'", funcName)})
			}
			params := mod.collectTokenize(source[start+1 : end])
			if funcInfo.MultiArg {
				tokenCh <- FunctionToken{FactoidFunction: funcInfo, params: mod.splitParams(params)}
			} else {
				tokenCh <- FunctionToken{FactoidFunction: funcInfo, params: [][]Token{params}}
			}
			source = source[end+1:]
		}
		m = FunctionTokenRgx.FindStringSubmatchIndex(source)
	}
	// Parameter directives
	m = ParameterTokenRgx.FindStringSubmatchIndex(source)
	for m != nil {
		var opStr, startStr, rangeStr, endStr string
		if m[2] != -1 {
			opStr = source[m[2]:m[3]]
		}
		if m[4] != -1 {
			startStr = source[m[4]:m[5]]
		}
		if m[6] != -1 {
			rangeStr = source[m[6]:m[7]]
		}
		if m[8] != -1 {
			endStr = source[m[8]:m[9]]
		}
		t := NewParameterToken(source[m[0]:m[1]], opStr, startStr, rangeStr, endStr)
		mod.tokenize(source[:m[0]], true, tokenCh)
		tokenCh <- t
		source = source[m[1]:]
		m = ParameterTokenRgx.FindStringSubmatchIndex(source)
	}
	tokenCh <- TextToken{Text: source}
	if !recursed {
		close(tokenCh)
	}
}

func (mod *FactoidModule) splitParams(pTokens []Token) [][]Token {
	var fArgs [][]Token
	var cur []Token

	for _, v := range pTokens {
		if tt, ok := v.(TextToken); ok {
			var buf bytes.Buffer
			o := 0
			for i := 0; i < len(tt.Text); i++ {
				if tt.Text[i] == '\\' {
					i++
				} else if tt.Text[i] == ',' {
					if i > o {
						cur = append(cur, TextToken{
							Text: buf.String()})
					}
					fArgs = append(fArgs, cur)
					cur = nil
					buf.Reset()
					o = i + 1
				} else {
					buf.WriteByte(tt.Text[i])
				}
			}
			if buf.Len() > 0 {
				cur = append(cur, TextToken{Text: buf.String()})
			}
		} else {
			cur = append(cur, v)
		}
	}
	fArgs = append(fArgs, cur)
	return fArgs
}
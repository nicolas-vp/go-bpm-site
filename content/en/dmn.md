# DMN decisions

DMN tables compile into Go functions over the typed data model. Conditions become Go comparisons; there is no runtime FEEL interpreter. Unsupported expressions fail during generation or compilation.

## Connect a decision

1. Create a DMN decision table in the modeler and save it under `process/`.
2. Add a business rule task with its decision ID and result variable.
3. Add model fields for the inputs and outputs, then regenerate.

```xml
<bpmn:businessRuleTask id="CalcOffer" name="Calculate discount">
  <bpmn:extensionElements>
    <zeebe:calledDecision decisionId="Offer" resultVariable="offer" />
  </bpmn:extensionElements>
</bpmn:businessRuleTask>
```

Decision IDs are unique across the input directory. A decision can be reused by several tasks and processes. The generator creates a decision function in each calling process package and a method that records matched rule IDs and writes the result.

## Example: discount

A table named `Offer` with `FIRST` hit policy:

| Rule | `amount` (number) | `category` (string) | `discount` (number) | `reason` (string) |
|---|---|---|---|---|
| VipBig | `>= 10000` | `"vip"` | `15` | `"large VIP order"` |
| Big | `>= 10000` | `-` | `10` | `"large order"` |
| Status | `[1000..10000[` | `"vip","gold"` | `5` | `"customer status"` |

For multiple outputs, the result variable is a struct with fields corresponding to output **names**, not labels:

```go
type DataModel struct {
    Amount   float64 `json:"amount"`
    Category string  `json:"category"`
    Offer    Offer   `json:"offer"`
}

type Offer struct {
    Discount float64 `json:"discount"`
    Reason   string  `json:"reason"`
}
```

## Inputs

The Input Expression names a variable or expression: `amount` maps to `data.Amount`, `client.age` to `data.Client.Age`. JSON tags can also contain spaces. Expressions are translated in the same way as gateway conditions.

Each input is evaluated once. Types come from `typeRef`, model fields or literals: `number`/`double` map to `float64`, `integer`/`long` to `int`, plus `string` and `boolean`. Inputs with only wildcard tests do not need a model field.

## Input cells

| Cell | Meaning |
|---|---|
| Empty or `-` | Match any value |
| `"vip"`, `5`, `true` | Equality |
| `limit` | Equality with a model field |
| `< 5`, `<= 5`, `> 5`, `>= 5` | Comparison |
| `[1..10]` | Inclusive range |
| `]1..10[` or `(1..10)` | Exclusive range |
| `[1..10[` or `(1..10]` | Mixed bounds |
| `"vip","gold"` | Any listed value |
| `not("vip")`, `not(< 5)` | Negation |
| `? > a + 1` | Expression using the current input |
| `null` | Nil input field |

`null` requires a nil-capable field: a pointer, map, slice or `any`. A value field such as `float64` cannot distinguish a missing variable from zero. Pointer comparisons use the pointed-to value, with nil treated as the zero value for non-null tests.

## Outputs

One output writes directly into the result field. Multiple outputs write named fields of a result struct; nil struct pointers are allocated. List policies use slices, or slices of structs for multiple outputs. Empty output cells use the type's zero value.

## Hit policies

| Policy | Result | No match |
|---|---|---|
| `UNIQUE`, default | First matching row | Leave result unchanged |
| `FIRST` | First matching row | Leave unchanged |
| `ANY` | First matching row | Leave unchanged |
| `RULE ORDER` | All matches in table order | Empty slice |
| `COLLECT` | All matching results | Empty slice |
| `COLLECT SUM`, `MIN`, `MAX` | Numeric aggregation, one output | Leave unchanged |
| `COLLECT COUNT` | Number of matching rules | Zero |
| `PRIORITY`, `OUTPUT ORDER` | Unsupported | Generation error |

**Overlapping rules are not checked for `UNIQUE` or `ANY`.** Both return the first match. Choose `FIRST` explicitly when ordering matters. In first-match policies, rules after an unconditional wildcard row are unreachable and are not generated.

For example, a `COLLECT SUM` fee table can combine a base fee of 100, a large-order fee of 200 and an urgent-order fee of 300. If all three rows match, the result is 600 and history includes all three rule IDs.

## Rule history

Enable `Config.RulesHistory` to record decision ID, matched rule IDs and result in logs or `bpm_rules_history`. Records commit with the process step; failed or interrupted steps do not leave partial rule history.

In instance history, open a business rule task and select the matched-rules action. The console shows the decision table with matching rows highlighted. For repeated tasks, switch between calls. Input values are in element history variables, rather than a separate rule-history input record.

![Matched decision rules in the Russian console](images/history-rules.png)

## Limitations

Literal-expression decisions, `requiredDecision` dependencies, `PRIORITY`, `OUTPUT ORDER` and worker-based business rule tasks are rejected. Every business rule task needs a decision ID and result variable.

General FEEL functions, dates, durations, contexts and FEEL lists are not translated. Prepare complex inputs in a Go method before evaluating the decision; for example, compute `isWorkingDay` before the DMN task.

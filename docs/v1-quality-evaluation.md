# V1 explanation and quiz quality evaluation

Generate all fixtures with `python3 tests/fixtures/create_gemini_scenarios.py /tmp/vibe-quality-fixtures`. Each fixture is a fictional standalone Git repository with a committed before-state and a working-tree change. Review only the evidence sent by the app. The fixture generator does not run the sample code.

## Independent reference analyses

1. **business-logic (Go).** Discount now returns an error for negative cents, zero and nil below 100, and integer division by ten at and above 100. At 99 the discount is 0, at 100 it is 10, at 109 it is 10. Its signature changes, so callers may require adjustment. Callers and monetary units outside the comment are not supplied. Test the boundary and error return rather than assume all callers handle it.
2. **react-state (TypeScript).** The old updater spreads the captured items argument and the new updater appends to the previous state through a functional setter. Back-to-back updates can preserve each previous update when the state library evaluates the callbacks. No React import or component, timing, or initialization is shown. Test closure staleness, append order and the need for array-compatible state.
3. **authentication (client and Python API).** Client now sends JSON username/password and parses the response body. The server only checks presence and returns the username. Empty credentials return 400; any nonempty pair takes the success path. There is no password verification, routing, or caller shown. A matching payload does not prove an operational connection.
4. **mixed-checkout (client and Python API).** Empty or absent items produce 400, [0] passes, and SAVE10 applies integer floor division to the sum. The client sends JSON and returns the parsed body without checking status. The legacy module is deleted and README support hours are unrelated. Routing, callers, accepted item types and currency units are not shown. Test guard ordering, rounding and caller-visible response changes.
5. **react-search.** Search trims and lowercases the query and lowercases names before substring matching. Empty query matches every name. A missing query or non-string name may fail; the supplied code does not establish inputs. Test case/whitespace, empty query and substring behavior.
6. **cursor-pagination (Python API).** The new version filters rows whose id is strictly greater than after, slices the first limit, and returns the last batch id only when more matching rows exist than limit. With limit zero and at least one match it indexes an empty list. It does not sort, so correctness depends on input order; database and callers are absent. Test exclusivity, exact limit, zero limit and unsorted inputs.
7. **zero-stock.** A stored zero survives the new nullish default where the old logical OR substituted five. Missing/null still become five; false and empty string also differ. Inventory construction and callers are not shown. Test zero separately from absence.
8. **tenant-cache.** Cache keys use only user, while loading takes tenant and user. Two tenants with the same user key can receive the first tenant's cached profile. It has no invalidation in this snippet. Whether user identifiers are globally unique and whether this is used for access control are unknown. Test tenant collision and load count.
9. **webhook-retry.** On TimeoutError, the changed loop records the failed message then breaks and returns the list. Later messages were not attempted and are not included in the returned list. Other exception types propagate. The surrounding retry system is absent. Test first/middle failures and distinguish unattempted from failed.
10. **render-lookups.** Each order now scans the product array for the first matching ID, falling back to Unknown if missing. The old version indexed a map and would fail for a missing entry. Repeated linear scans can cost O(orders × products). Data size and index construction costs are unknown. Test duplicates and missing IDs.
11. **shared-normalization.** Both import and lookup now call a shared trim-and-lowercase helper. Import behavior is unchanged for ordinary strings, while lookup now trims. The import path and unsupported non-string inputs are not established. Test spaces and case and do not call the entire change behavior-preserving.
12. **java-validation.** Nonpositive quantities now throw IllegalArgumentException; positive quantities use multiplyExact, which throws on integer overflow. Price is not constrained. Callers and HTTP handling are absent. Test zero, negative, valid and overflowing values.
13. **sql-inventory.** The update now applies only to matching rows with positive stock and returns resulting stock. With no qualifying row, no row is returned. Schema, database dialect, transactions, uniqueness and concurrency guarantees are not shown. Test zero and one and avoid assuming a particular locking or affected-row policy.
14. **async-gather.** Previously fetches were awaited sequentially and an exception interrupted the loop. Now gather starts awaitables concurrently, preserves input order in its result, and returns exceptions as values. Fetch implementation, cancellation and external side effects are absent. Test ordering, exceptions and empty input.
15. **pure-refactor.** For ordinary numeric values with normal comparison and arithmetic, negative amounts yield zero and nonnegative amounts are doubled in both versions. The operations differ for types with custom comparison/multiplication, NaN, or unusual return types; no input contract is shown. Avoid claiming equivalence for all supported inputs without that contract.

## Method

For each real result, independently compare all findings and each option with the corresponding reference and cited excerpt. Citations establish line presence, not semantic correctness. Mark explanation accuracy separately from coverage. A question has a correct, unambiguous key only if exactly one option follows from visible evidence. Distractor quality requires three distinct, plausible alternatives. Failed or omitted responses remain in denominators.

## Real inference results

There were **22 real inference requests** in this evaluation: 17 API-driven VibeCodeFolks requests, two direct Codex baseline requests, and three dashboard-triggered VibeCodeFolks requests. The latter included one failure, one successful explanation with no displayed quiz, and one successful revised run with a question. They cover five distinct scenarios, not fifteen: the remaining fixtures are reference-only. The browser-only explanations are not included in the factual-accuracy numerator.

Runs used the installed Codex CLI signed in through its existing account and default model configuration. The exact configured model ID was not captured, so it cannot be stated. No paid API provider was used. Results below are manual assessments, not automated semantic checks.

| Run set | Fixtures attempted | Accepted explanations | Displayed questions | Correct, unambiguous keys | All distractors plausible and distinct |
| --- | ---: | ---: | ---: | ---: | ---: |
| Original prompt | 5 | 3/5 | 9 | 9/9 | 6/9 |
| More restrictive prompt | 5 | 5/5 | 0 | — | — |
| Restrictive prompt capture repeat | 1 | 1/1 | 0 | — | — |
| Reverted prompt, same three | 3 | 3/3 | 0 | — | — |
| Revised trace instruction | 3 | 3/3 | 1 | 1/1 | 1/1 |

The first pass's tenant-cache and webhook-retry requests returned HTTP 502 with a generic Codex failure after 4.6 and 4.7 seconds. The same two fixtures succeeded in the restrictive-prompt pass. Neither successful response produced a displayed quiz. The underlying failures are unknown and remain part of the results. The additional capture run used the same restrictive prompt and recorded the model's raw JSON for fictional business-logic evidence. Its quiz field was null, establishing model omission in that run. Other empty quizzes could be model omissions or validation rejection. The displayed results alone do not distinguish them.

The restrictive prompt added instructions about likely misconceptions and realistic uses. After observing zero quizzes in the first two samples it was reverted. The subsequent three results still had zero displayed quizzes. A narrower instruction then explicitly requested at least one concrete trace only when the excerpt supports it, while retaining null for insufficient evidence. One of the three revised requests produced one question. This is a small, non-deterministic sample and does not establish that the instruction caused the change. The final instruction is the narrower version.

The 15 accepted API explanations were manually checked against the reference. **15/15 had no factual error found** in the findings reviewed. This does not count coverage as correctness or establish accuracy on unseen code. The revised Go explanation omitted the useful 109 example and the React outputs did not show concrete queued A/B behavior. The pagination results identified zero-limit failure and unsorted cursor skipping. Tenant-cache identified the conditional cross-tenant leak and the retry result correctly distinguished unattempted messages from failures.

### Individual question review

Original prompt:

1. Go negative input. The key, zero with an error, follows the first branch. The distractors distinguish old division, zero without an error, and division combined with an error. Distinct and plausible.
2. Go threshold 99 and 100. The key is 0 and 10. Alternatives represent the old calculation, excluding the boundary, and rounding the lower value. Distinct and plausible.
3. Go single-value assignment. The key is a compile-time assignment mismatch. Alternatives reflect implicit error disposal, runtime-only checking and packaging multiple results. Distinct and plausible for a developer learning Go.
4. React queued A/B updates. Under the explicitly stated setter assumption the key is [A, B]. Alternatives represent losing the first update, reversing order and losing the second. Distinct and plausible.
5. React recording setter. The key is the updater function. The empty-array and bare-item choices are weak because neither is constructed or passed by the code. Correct key; distractors fail the quality gate.
6. React omitted old third argument. The key is a spread error before the setter is called. An empty array and an updater function are weak options for this old implementation. Correct key; distractors fail the quality gate.
7. Pagination exact fit. The key is [3, 4] and None. Two wrong answers both retain the excluded boundary row, differing only in cursor, so the set has repetitive misunderstandings. Correct key; distractors fail the quality gate.
8. Pagination zero limit. The key is IndexError. The alternatives reflect returning no cursor, ignoring the limit, and introducing explicit validation. Distinct and plausible.
9. Pagination cursor reuse on unsorted IDs. The key is an empty second batch and None. Alternatives reflect sorting implicitly, accepting equality, and preserving a cursor without remaining matches. Distinct and plausible.

Revised trace instruction:

10. Go 99 and 100. The key is (0, nil) and (10, nil). The alternatives reflect old behavior, an exclusive threshold and incorrectly applying the negative-input error. Distinct and plausible.

The revised browser run generated one additional Go threshold question. Its key and distractors matched question 10 above. Overall, **11/11 displayed questions** had a correct and unambiguous answer key, and **8/11** passed the conservative distractor assessment. Counts cover questions actually displayed in accepted responses, not a denominator of expected questions. The missing quizzes and failed runs are reported separately above. The first and revised Go questions overlap in concept because they are separate runs, not one quiz.

## Single-request baseline

Two baseline requests used the exact evidence objects captured from the Go and React app preparations. Each used the installed Codex CLI with the same default model configuration and asked for changes, flow, edge cases, caller effects, risks and evidence limits in one request. The baseline did not receive the app's detailed system instruction, structured response schema or interactive quiz. Both baseline explanations were factually correct against the references.

- **Go:** Both descriptions correctly covered negative inputs, the threshold, return signature and unseen callers. The baseline was more complete on integer truncation, including 109, and noted function-value compatibility and error identity. VibeCodeFolks added individually validated source links and a three-question quiz in the original run. One revised run contained a single threshold question.
- **React:** Both correctly described the functional updater and conditional setter semantics. The baseline additionally covered shallow copies, duplicate items, non-iterable previous state and TypeScript versus JavaScript extra-argument behavior. The original VibeCodeFolks response supplied three questions, though two had weak distractors. The later app runs displayed no quiz. In these samples the baseline explanation was more useful in detail; interactive citation and the strongest quiz question added learning value when available.

Two comparisons cannot establish a general ranking. The baseline has readable line references but they are not mechanically validated or interactive. No response can guarantee semantic correctness solely through its citations.

## Browser and automated verification

The initial browser request failed with the generic Codex error. A later browser run with the restrictive prompt produced an explanation but no quiz and showed the existing insufficient-or-invalid message. The revised browser request returned an explanation and one Go boundary question. Selecting an incorrect answer displayed the correct answer and reasoning for each alternative; the score showed 0/1. The explanation from this browser-only run was not included in the 15 reviewed API explanations. Prior dashboard runs in this project exercised generated quiz interactions, but they are not substitutes for this evaluation's missing-quiz observations.

Formatting, Go vet, Go tests, race tests, build, JavaScript syntax checks, and all 13 dashboard tests passed after the narrow instruction change. These tests exercise structure and interactions, not factual or distractor quality.

Release blockers remain: quiz availability is inconsistent on small suitable diffs, unavailable output is not distinguished from rejected output, and the observed generic Codex failures give insufficient diagnosis. Semantic equivalence of distractors and factual correctness still require manual review. The five-scenario sample and two baseline comparisons do not justify a v1 quality claim.

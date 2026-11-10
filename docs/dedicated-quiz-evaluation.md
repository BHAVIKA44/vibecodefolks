# Dedicated quiz evaluation

## Method

The same fictional fixtures and reference analyses in v1-quality-evaluation.md were used. The app made one explanation request followed by one explicitly initiated dedicated quiz request per scenario, with the installed signed-in Codex CLI and its existing default model. No model override or paid API provider was used. Each question was manually assessed for correctness, distinct plausible alternatives, references and feedback. Structural validation cannot establish semantic correctness.

The previous combined-request evaluation had 0 quizzes in 10 successful requests over five eligible scenarios (two runs each). The dedicated evaluation returned quizzes on 5/5 requests over those same five scenarios, one run each. These are different, small samples on different occasions, so the result supports testing the new approach but does not establish a population-wide reliability rate.

| Fixture | Explanation | Dedicated quiz | Questions | Correct keys | Strong distractor sets |
| --- | --- | --- | ---: | ---: | ---: |
| business-logic | success | available | 2 | 2 | 2 |
| react-state | success | available | 1 | 1 | 1 |
| cursor-pagination | success | available | 3 | 3 | 3 |
| tenant-cache | success | available | 2 | 2 | 2 |
| webhook-retry | success | available | 3 | 3 | 3 |
| mixed-checkout | success | available | 3 | 3 | 2 |
| java-validation | success | available | 2 | 2 | 0 |
| sql-inventory | success | available | 2 | 2 | 0 |
| async-gather | success | available | 2 | 2 | 2 |
| API total | 9/9 | 9/9 | 20 | 20 | 15 |
| Browser mixed-checkout | success | available | 3 | 3 | 2 |
| Overall | 10/10 | 10/10 | 23 | 23 | 17 |

These are actual inference requests. The browser run is a separate request against the mixed-checkout fixture. One additional live browser quiz request was deliberately canceled, giving 10 successful quiz requests out of 11 initiated, including the cancellation. Each result was accepted by the structural validator; no generated quiz was rejected in this sample. A correct key means exactly one answer was defensible under the stated assumptions and the cited code. The distractor assessment is conservative and subjective.

## Question review

1. Go threshold at 99/100 returns 0/10 after the change. Alternatives cover prior calculation, an exclusive boundary and rounding. Correct and plausible.
2. Go negative -10 returns 0 and an error. Alternatives misread guard order, old division or return amount. Correct and plausible.
3. React queued updaters preserve [A, B, C] while two old stale replacements leave [A, C], given the stated setter semantics. Alternatives confuse snapshot and queue ordering. Correct and plausible.
4. Pagination on unsorted IDs [5, 2, 4] yields [5, 2] and cursor 2, then [5, 4] and None. Alternatives assume sorting, remembered rows or a different filter. Correct and a strong multi-step trace.
5. Pagination with one match and limit zero raises IndexError. Alternatives assume a guarded empty page, cursor from matches, or no limit. Correct and plausible.
6. Pagination with exactly two matches and limit two returns both and no cursor. Alternatives misread > versus >=, exclusivity or slicing. Correct and plausible.
7. Tenant cache collision returns A's profile on B's call and loads once. Alternatives assume tenant-aware keys, reload or rejection. Correct and exposes a concrete risk.
8. A cached None is reused, while the old implementation reloads and returns the second value. Alternatives confuse membership, caching and None handling. Correct and plausible.
9. A webhook timeout at B yields [B] and never attempts C. Alternatives confuse break, failure with unattempted work, or propagation. Correct and useful.
10. Successful webhook sends produce an empty failure list, whereas the old function returns None. Alternatives confuse success values with failures and implicit return. Correct and plausible.
11. ValueError propagates and prevents later sends. Alternatives overgeneralize the TimeoutError handler. Correct and plausible.
12. Empty checkout changed from total zero/200 to error/400. Alternatives describe wrong old or new behavior. Correct and plausible.
13. SAVE10 on [5] gives 4; lowercase gives 5. One distractor returns empty-cart errors for a nonempty list and is weak. Correct key; distractor gate fails.
14. An HTTP 400 with a JSON body resolves to the parsed body. Alternatives confuse rejection, old Response return and extraction of an absent total. Correct and plausible.
15. Java quantity zero changes from return 0 to an exception. One distractor says both return 0 and another describes a path that also returns 0, so the two overlap semantically. Correct key; distractor gate fails.
16. Java negative price and positive quantity reaches multiplyExact unchanged. Rejecting a positive quantity is an obviously weak distractor. Correct key; distractor gate fails.
17. SQL stock 0 changes from decrement to -1 to no update. The new statement increasing stock is an obviously weak distractor. Correct key; distractor gate fails.
18. SQL stock 2 becomes and returns 1. A distractor positing an unexplained repeated decrement to 0 is weak. Correct under the stated SQL behavior; distractor gate fails.
19. Async gather permits overlap and returns values in input order. Alternatives confuse completion order, sequential execution and first-completed semantics. Correct and plausible.
20. Async gather with return_exceptions retains ValueError in its corresponding slot while the old sequential loop propagates. Alternatives confuse dropping errors and early returns. Correct and plausible.
21. Browser empty-cart comparison has correct key, but two wrong choices predict the same old and new results. Distractor gate fails.
22. Browser coupon [5, 6] returns 9 with status 200. Alternatives represent float division, rounding up and no discount. Correct and plausible.
23. Browser wrapper returns parsed JSON on a stubbed 400 response, whereas the old wrapper returns the Response. Alternatives confuse error status and parsing. Correct and plausible.

The browser flow reviewed a fictional diff, generated an explanation, explicitly requested a Codex quiz, answered one incorrectly, read option-specific feedback, completed the score, retried the same questions and opened an explanation citation. Fixture contents remained the expected fictional changes. Remote consent and retry without another request were covered by dashboard interaction tests, not a real remote-provider call. A second real browser session generated an explanation and then canceled its quiz request, returning to a retryable state. That canceled request is excluded from the completed-request quality table. Stale snapshot rejection was covered by automated interaction and server tests, not by a real browser in this evaluation.

## Assessment

A final regression check also covered cancellation while the post-generation freshness check is pending and automatic cleanup of expired in-memory quiz sessions.

The dedicated path addresses the observed combined response omission in this small sample. It does not make semantic validation possible. Six questions still contained weak or overlapping alternatives, and several fixtures produced only one or two questions. The app does not independently prove correctness or silently repair weak questions. A second provider request introduces latency, cost for paid providers, and another opportunity for failure; the dashboard discloses this and asks for renewed consent before remote transmission. These results are insufficient to claim v1 readiness.

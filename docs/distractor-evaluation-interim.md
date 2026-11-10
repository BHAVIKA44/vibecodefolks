# Distractor evaluation — interim results

The targeted distractor pass was interrupted before its full independent review and before a complete updated browser journey was recorded. These results do not establish that the 90% release target was met.

Fourteen fictional scenarios were attempted through the local Codex workflow. Five explanations and their quiz requests succeeded and returned nine questions. One explanation succeeded but quiz generation failed with HTTP 502. Eight explanation requests failed with HTTP 502 before a quiz request. The recorded results contain no assessment of whether those provider failures were caused by quotas or another issue.

| Scenario | Explanation | Quiz | Questions |
| --- | --- | --- | ---: |
| business-logic | success | success | 2 |
| java-validation | success | success | 2 |
| react-state | success | success | 2 |
| sql-inventory | success | success | 1 |
| async-gather | success | success | 2 |
| cursor-pagination | success | HTTP 502 | 0 |
| authentication, mixed-checkout, pure-refactor, react-search, shared-normalization, tenant-cache, webhook-retry, zero-stock | HTTP 502 | not requested | 0 |

Independent review was completed for four questions, the two business-logic and two java-validation questions. All four had correct keys and were judged to have distinct, plausible distractors. The remaining five generated questions have not received a complete independent review, so no overall answer-key or distractor pass rate is asserted.

The six prior failures and the recurring causes are documented in quiz-distractor-root-causes.md. The focused changes request private outcome-by-outcome construction and comparison, reject identical option feedback, and preserve case-sensitive differences in output alternatives. These checks do not establish semantic distinctness. The earlier 17/23 distractor-quality baseline and detailed prior evaluation are in dedicated-quiz-evaluation.md.

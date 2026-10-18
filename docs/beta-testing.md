# Manual beta testing

Date: 2026-10-18. Tested on macOS using a compiled local executable, the browser dashboard, and disposable fictional Git repositories. This is a focused manual test, not exhaustive certification.

| Scenario | Result |
| --- | --- |
| Changes in login, Go business logic, and mixed JavaScript/Python checkout repositories | Changed files and counts displayed; real Gemini explanations and quizzes returned. See [Gemini evaluation](gemini-evaluation.md). |
| Mixed change with a deleted file and unrelated README edit | Both appeared in the change list and generated graph; the inferred connection was labeled. |
| Consent | Sending without checking consent was blocked; canceling did not send evidence. |
| Quiz | Correct and intentionally incorrect answers received feedback; score reflected the answers. |
| Repository temporarily unavailable, then restored | Error appeared and explain remained disabled; refresh recovered. This uncovered a bug fixed in this change. |
| Clean repository with an empty initial commit | No changes displayed; explain remains disabled after the fix in this change. |
| Missing AI configuration | A clear configuration error appeared without contacting a provider. |

The three real provider requests used fictional code and the user's reported free quota; billing was not independently checked. This session did not test real provider outages, quota exhaustion, repeatability, large repositories, or manual use on Windows or Linux. The automated suite covers additional cases using mocks; it is not a substitute for real-provider and usability testing. AI claims, links, filtering of secrets, and prompt-injection defenses require independent judgment and are not guarantees.

The current evidence supports a small supervised trial with nonsensitive repositories, not a claim of general accuracy or readiness for a broad beta. Gather feedback from outside developers and test larger and more varied repositories before expanding access.

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

## Follow-up: Linux and provider failures

On 2026-10-18 a Linux ARM64 binary was run in a disposable Docker container with Git and no external network. The dashboard returned HTTP 200; a staged modification and an untracked file appeared in the working-tree view; after restoring a clean tree the last commit appeared; and missing AI configuration produced a clear error. This checked the Linux server and API through commands inside the container, not a Linux desktop browser.

In the macOS browser, a controlled loopback provider returned HTTP 429, HTTP 500, and malformed JSON. Each produced an error and left the explain action available; a subsequent valid response succeeded. These were simulations and do not demonstrate real quota exhaustion. A scripted local provider also returned an explanation, graph, and quiz through the updated dashboard.

Separately, a fictional 250-file repository with 120 changed files was tested against Gemini. The dashboard listed all 120, prepared evidence for 20, and disclosed 100 exclusions. A real request returned HTTP 503; two subsequent requests succeeded and produced similar correct summaries of the selected repetitive changes. The second generated quiz's three answer keys matched the sample. This is a narrow repeatability check, not a general reliability measurement. The app was run manually on macOS; manual Windows and Linux desktop-browser testing remains outstanding. GitHub CI has run the automated suite on all three operating systems.

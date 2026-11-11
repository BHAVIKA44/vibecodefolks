let quizState = null;
let quizVersion = 0;
let quizContext = null;
let quizRequestController = null;
function clearQuiz() {
  quizVersion++;
  quizRequestController?.abort();
  quizRequestController = null;
  quizContext = null;
  quizState = null;
  element('quiz-consent-panel').hidden = true;
  element('quiz-cancel-request').hidden = true;
  element('quiz-generate').disabled = false;
  element('quiz-section').hidden = true;
  element('quiz-start').disabled = false;
  element('quiz-content').replaceChildren();
  element('quiz-status').textContent = '';
}
function prepareQuiz(result) {
  clearQuiz();
  const quiz = result.explanation.quiz;
  element('quiz-section').hidden = false;
  element('quiz-generate').hidden = !result.quiz_session;
  if (result.quiz_session) {
    quizContext = {session: result.quiz_session, snapshot: result.snapshot, evidence: result.evidence,
      local: result.quiz_local, provider: result.quiz_provider, model: result.quiz_model};
    element('quiz-start').hidden = true;
    element('quiz-status').textContent = 'Generate a quiz using the evidence captured for this explanation.';
    return;
  }
  element('quiz-start').hidden = !quiz || !quiz.questions.length;
  if ('quiz_session' in result && !result.quiz_session) {
    element('quiz-status').textContent = 'Quiz generation is temporarily unavailable. Explain the changes again to create a session.';
    return;
  }
  const messages = {
    missing: 'The provider did not include a quiz. Try explaining the changes again.',
    model_null: 'The provider returned no quiz. Try explaining the changes again if the diff contains testable behavior.',
    empty: 'The provider returned no questions for this evidence. Review the diff or try again.',
    invalid_structure: 'The provider returned a quiz in an invalid format. The explanation is available; try again.',
    invalid_reference: 'The provider cited code outside the reviewed evidence, so the quiz was rejected. The explanation is available; try again.'
  };
  element('quiz-status').textContent = quiz && quiz.questions.length ? 'Questions are AI-generated interpretations of this evidence snapshot, not verified runtime behavior.' : (messages[result.explanation.quiz_diagnostic] || 'No quiz is available for this evidence. Try explaining the changes again.');
  if (quiz && quiz.questions.length) quizState = {questions: quiz.questions, evidence: result.evidence, snapshot: result.snapshot, index: 0, score: 0, answered: false};
}
function requestQuiz() {
  if (!quizContext || quizRequestController) return;
  if (quizContext.local) {
    generateQuiz(false);
    return;
  }
  element('quiz-consent').checked = false;
  element('quiz-privacy-info').textContent = `This sends the previously reviewed evidence to ${quizContext.provider} using ${quizContext.model || 'the configured model'} in another request. The provider may retain it and the request may incur costs.`;
  element('quiz-consent-panel').hidden = false;
  element('quiz-status').textContent = 'Review the destination and consent before generating a quiz.';
}
async function generateQuiz(consent) {
  if (!quizContext || quizRequestController) return;
  if (!quizContext.local && !consent) {
    element('quiz-status').textContent = 'Please consent before sending the evidence again.';
    return;
  }
  const context = quizContext;
  const version = quizVersion;
  const controller = new AbortController();
  quizRequestController = controller;
  element('quiz-consent-panel').hidden = true;
  element('quiz-generate').disabled = true;
  element('quiz-cancel-request').hidden = false;
  element('quiz-status').textContent = 'Generating questions from the reviewed evidence…';
  try {
    const result = await post('/api/quiz/generate', {session: context.session, snapshot: context.snapshot, consent}, controller.signal);
    if (version !== quizVersion || quizContext !== context) return;
    if (result.snapshot !== context.snapshot) {
      element('quiz-status').textContent = 'The quiz did not match the reviewed evidence. Refresh and explain the changes again.';
      return;
    }
    const quiz = result.quiz;
    if (!quiz || !Array.isArray(quiz.questions) || !quiz.questions.length) {
      const messages = {
        missing: 'The provider did not include a quiz.',
        model_null: 'The provider returned no quiz.',
        empty: 'The provider returned no questions for this evidence.',
        invalid_structure: 'The generated quiz had an invalid format.',
        invalid_reference: 'The generated quiz cited code outside the reviewed evidence.'
      };
      element('quiz-status').textContent = `${messages[result.quiz_diagnostic] || 'No supported questions were available.'} You can try generating again.`;
      return;
    }
    quizState = {questions: quiz.questions, evidence: context.evidence, snapshot: context.snapshot, index: 0, score: 0, answered: false};
    element('quiz-generate').hidden = true;
    await startQuiz(controller.signal);
  } catch (error) {
    if (version !== quizVersion || quizContext !== context) return;
    element('quiz-status').textContent = error.name === 'AbortError' ? 'Quiz request canceled. You can try again.' : error.message;
  } finally {
    if (version === quizVersion && quizContext === context) {
      quizRequestController = null;
      element('quiz-generate').disabled = false;
      element('quiz-cancel-request').hidden = true;
    }
  }
}
async function startQuiz(signal) {
  if (!quizState) return;
  const abortSignal = signal && typeof signal.aborted === 'boolean' ? signal : undefined;
  const version = quizVersion;
  const button = element('quiz-start');
  button.disabled = true;
  element('quiz-status').textContent = 'Checking the local evidence snapshot…';
  try {
    const current = await post('/api/quiz/fresh', {snapshot:quizState.snapshot}, abortSignal);
    if (abortSignal?.aborted) { quizState = null; element('quiz-generate').hidden = false; element('quiz-status').textContent = 'Quiz request canceled. You can try again.'; return; }
    if (version !== quizVersion || !quizState) return;
    if (!current.fresh) { clearQuiz(); element('quiz-section').hidden = false; element('quiz-start').hidden = true; element('quiz-status').textContent = 'The evidence has changed. Refresh and explain the changes again before taking the quiz.'; return; }
    quizState.index = 0; quizState.score = 0; quizState.answered = false; quizState.answers = [];
    button.hidden = true;
    element('quiz-status').textContent = 'Based on the captured evidence. AI answers may be wrong; runtime behavior is not verified.';
    renderQuestion();
  } catch (_) {
    if (version === quizVersion) {
      if (abortSignal?.aborted) {
        quizState = null;
        element('quiz-generate').hidden = false;
        element('quiz-status').textContent = 'Quiz request canceled. You can try again.';
      } else {
        element('quiz-start').hidden = false;
        element('quiz-status').textContent = 'Could not verify the local evidence snapshot. Try again or refresh.';
      }
    }
  } finally { if (version === quizVersion) button.disabled = false; }
}
function renderQuestion() {
  const s = quizState, q = s.questions[s.index], panel = element('quiz-content');
  panel.replaceChildren();
  panel.append(text('p', `QUESTION ${s.index + 1} OF ${s.questions.length} · ${q.category.toUpperCase()}`, 'label'), text('h3', q.text));
  const options = document.createElement('div'); options.className = 'quiz-options';
  const feedback = document.createElement('div'); feedback.className = 'quiz-feedback'; feedback.setAttribute('aria-live', 'polite');
  graphReferences(panel, q.references, s.evidence);
  q.options.forEach((option, index) => {
    const button = document.createElement('button'); button.type = 'button'; button.className = 'quiz-option'; button.textContent = `${String.fromCharCode(65 + index)}. ${option}`;
    button.addEventListener('click', () => {
      if (!quizState || s.answered) return;
      s.answered = true;
      for (const optionButton of options.children) optionButton.disabled = true;
      const correct = index === q.correct_index;
      button.className = `quiz-option ${correct ? 'selected-correct' : 'selected-wrong'}`;
      if (!correct && options.children[q.correct_index]) options.children[q.correct_index].className = 'quiz-option correct-answer';
      if (correct) s.score++;
      s.answers.push({question: q, selected: index, correct});
      feedback.append(text('p', correct ? 'Correct' : 'Incorrect', 'quiz-reaction'), text('h4', `Correct answer: ${String.fromCharCode(65 + q.correct_index)}. ${q.options[q.correct_index]}`), text('p', q.explanation, 'muted'), text('p', 'AI interpretation; check the cited evidence. Runtime behavior is unverified.', 'hint'));
      if (q.option_explanations?.[index]) feedback.append(text('p', `Your choice: ${q.option_explanations[index]}`, 'muted'));
      graphReferences(feedback, q.references, s.evidence);
      const next = document.createElement('button'); next.type = 'button'; next.className = 'quiz-next'; next.textContent = s.index + 1 === s.questions.length ? 'See score' : 'Next Question'; next.addEventListener('click', () => { if (!quizState) return; s.index++; s.answered = false; if (s.index === s.questions.length) renderScore(); else renderQuestion(); }); feedback.append(next);
      feedback.scrollIntoView?.({block: 'nearest'});
    });
    options.append(button);
  });
  panel.append(options, feedback);
}
function renderScore() {
  const s = quizState, panel = element('quiz-content'); panel.replaceChildren();
  panel.append(text('p', 'QUIZ COMPLETE', 'label'), text('h3', 'Review your understanding'), text('p', `${s.score} / ${s.questions.length} correct`, 'quiz-score'));
  for (const {question, selected, correct} of s.answers) {
    const review = document.createElement('details');
    review.append(text('summary', `${correct ? 'Correct' : 'Revisit'} · ${question.category}: ${question.text}`),
      text('p', `Your answer: ${question.options[selected]}`),
      text('p', `Correct answer: ${question.options[question.correct_index]}`),
      text('p', question.explanation));
    if (question.option_explanations?.[selected]) review.append(text('p', `Your choice: ${question.option_explanations[selected]}`, 'muted'));
    graphReferences(review, question.references, s.evidence);
    panel.append(review);
  }
  const retry = document.createElement('button'); retry.type = 'button'; retry.className = 'quiz-retry'; retry.textContent = 'Retry same questions'; retry.addEventListener('click', () => startQuiz()); panel.append(retry);
}

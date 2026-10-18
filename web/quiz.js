// Quiz messages are local UI copy, never model instructions.
const quizMessages = {
  correct: ['Bro, you actually understand your code. Respect! 😎', 'AI wrote it, but you understood it. Big brain energy. 🚀', 'Nice work! You followed the evidence. 🧠'],
  incorrect: ['No worries, bro. Let’s look at the evidence. 💙', 'The vibes were strong, but the answer was different. 💀', 'Good try! Here’s what the supplied code suggests.'],
  perfect: 'Certified Code Wizard 🏆', good: 'Bro actually knows what’s going on! 🔥', low: 'The vibes are immaculate. Let’s understand the code a little better. 💙'
};
let quizState = null;
let quizVersion = 0;
function clearQuiz() {
  quizVersion++;
  quizState = null;
  element('quiz-section').hidden = true;
  element('quiz-start').disabled = false;
  element('quiz-content').replaceChildren();
  element('quiz-status').textContent = '';
}
function prepareQuiz(result) {
  clearQuiz();
  const quiz = result.explanation.quiz;
  element('quiz-section').hidden = false;
  element('quiz-start').hidden = !quiz || !quiz.questions.length;
  element('quiz-status').textContent = quiz && quiz.questions.length ? 'Questions are AI-generated interpretations of this evidence snapshot, not verified runtime behavior.' : 'Not enough supported quiz questions were available for this snapshot.';
  if (quiz && quiz.questions.length) quizState = {questions: quiz.questions, evidence: result.evidence, snapshot: result.snapshot, index: 0, score: 0, answered: false};
}
async function startQuiz() {
  if (!quizState) return;
  const version = quizVersion;
  const button = element('quiz-start');
  button.disabled = true;
  element('quiz-status').textContent = 'Checking the local evidence snapshot…';
  try {
    const current = await post('/api/quiz/fresh', {snapshot:quizState.snapshot});
    if (version !== quizVersion || !quizState) return;
    if (!current.fresh) { clearQuiz(); element('quiz-section').hidden = false; element('quiz-start').hidden = true; element('quiz-status').textContent = 'The evidence has changed. Refresh and explain the changes again before taking the quiz.'; return; }
    quizState.index = 0; quizState.score = 0; quizState.answered = false;
    button.hidden = true;
    element('quiz-status').textContent = 'Based on the captured evidence. AI answers may be wrong; runtime behavior is not verified.';
    renderQuestion();
  } catch (_) {
    if (version === quizVersion) element('quiz-status').textContent = 'Could not verify the local evidence snapshot. Refresh and try again.';
  } finally { if (version === quizVersion) button.disabled = false; }
}
function renderQuestion() {
  const s = quizState, q = s.questions[s.index], panel = element('quiz-content');
  panel.replaceChildren();
  panel.append(text('p', `QUESTION ${s.index + 1} OF ${s.questions.length} · ${q.category.toUpperCase()}`, 'label'), text('h3', q.text));
  const options = document.createElement('div'); options.className = 'quiz-options';
  const feedback = document.createElement('div'); feedback.className = 'quiz-feedback'; feedback.setAttribute('aria-live', 'polite');
  q.options.forEach((option, index) => {
    const button = document.createElement('button'); button.type = 'button'; button.className = 'quiz-option'; button.textContent = `${String.fromCharCode(65 + index)}. ${option}`;
    button.addEventListener('click', () => {
      if (!quizState || s.answered) return;
      s.answered = true;
      for (const optionButton of options.children) optionButton.disabled = true;
      const correct = index === q.correct_index;
      if (correct) s.score++;
      const messages = correct ? quizMessages.correct : quizMessages.incorrect;
      feedback.append(text('p', messages[Math.floor(Math.random() * messages.length)], 'quiz-reaction'), text('h4', `Correct answer: ${String.fromCharCode(65 + q.correct_index)}. ${q.options[q.correct_index]}`), text('p', q.explanation, 'muted'), text('p', 'AI interpretation; check the cited evidence. Runtime behavior is unverified.', 'hint'));
      graphReferences(feedback, q.references, s.evidence);
      const next = document.createElement('button'); next.type = 'button'; next.textContent = s.index + 1 === s.questions.length ? 'See score' : 'Next Question'; next.addEventListener('click', () => { if (!quizState) return; s.index++; s.answered = false; if (s.index === s.questions.length) renderScore(); else renderQuestion(); }); feedback.append(next);
    });
    options.append(button);
  });
  panel.append(options, feedback);
}
function renderScore() {
  const s = quizState, panel = element('quiz-content'); panel.replaceChildren();
  const message = s.score === s.questions.length ? quizMessages.perfect : s.score / s.questions.length >= 0.6 ? quizMessages.good : quizMessages.low;
  const concepts = [...new Set(s.questions.map(q => q.category))].join(', ');
  panel.append(text('p', 'QUIZ COMPLETE', 'label'), text('h3', message), text('p', `${s.score} / ${s.questions.length} correct`, 'quiz-score'), text('p', `Topics covered: ${concepts}. This is a learning exercise, not a certification.`, 'muted'));
  const retry = document.createElement('button'); retry.type = 'button'; retry.textContent = 'Retry Quiz'; retry.addEventListener('click', startQuiz); panel.append(retry);
}

const SINGLE_CHOICE_TYPES = new Set(['classic', 'multiple_choice', 'single_select'])

const VALID_QUIZ_TYPES = new Set([
  'classic',
  'multiple_choice',
  'single_select',
  'multi_select',
  'true_false',
  'range',
  'number',
  'correct_picture',
  'wrong_pictures',
  'simple_text',
])

function createError(index, field, messageKey, params = {}) {
  return { index, field, messageKey, params }
}

function hasNonEmptyText(value) {
  return String(value ?? '').trim().length > 0
}

function parseFiniteNumber(value) {
  if (value === null || value === undefined || value === '' || typeof value === 'boolean') {
    return NaN
  }
  const num = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(num) ? num : NaN
}

function getQuestionPoints(question, type) {
  if (type === 'wrong_pictures') {
    return (
      question?.wrongPictureConfig?.maxPoints ??
      question?.scoringConfig?.maxPoints ??
      question?.points
    )
  }
  return question?.scoringConfig?.maxPoints ?? question?.points
}

function validatePoints(question, type, index, errors) {
  const points = parseFiniteNumber(getQuestionPoints(question, type))
  if (!Number.isFinite(points) || points <= 0) {
    const messageKey =
      type === 'simple_text'
        ? 'quiz_config.validation.simple_points'
        : 'quiz_config.validation.points_positive'
    errors.push(createError(index, 'points', messageKey))
  }
}

function validateChoiceAnswers(question, index, isMultiSelect, errors) {
  const answers = Array.isArray(question?.answers) ? question.answers : []
  const validAnswerIndices = new Set()

  answers.forEach((answer, answerIndex) => {
    if (hasNonEmptyText(answer)) {
      validAnswerIndices.add(answerIndex)
    }
  })

  if (validAnswerIndices.size < 2) {
    errors.push(createError(index, 'answers', 'quiz_config.validation.mc_min_answers', { min: 2 }))
  }

  if (isMultiSelect) {
    const rawIndices = Array.isArray(question?.correctIndices) ? question.correctIndices : []
    const validSelected = new Set()
    let hasInvalidSelection = false

    for (const idx of rawIndices) {
      if (Number.isInteger(idx) && validAnswerIndices.has(idx)) {
        validSelected.add(idx)
      } else {
        hasInvalidSelection = true
      }
    }

    if (validSelected.size < 2) {
      errors.push(
        createError(index, 'correctIndices', 'quiz_config.validation.multi_select_min_correct', {
          min: 2,
        })
      )
    } else if (hasInvalidSelection) {
      errors.push(
        createError(index, 'correctIndices', 'quiz_config.validation.correct_answer_required')
      )
    }
    return
  }

  if (!Number.isInteger(question?.correctIndex) || !validAnswerIndices.has(question.correctIndex)) {
    errors.push(
      createError(index, 'correctIndex', 'quiz_config.validation.correct_answer_required')
    )
  }
}

function validateTrueFalse(question, index, errors) {
  if (question?.answers !== undefined) {
    const answers = Array.isArray(question.answers) ? question.answers : []
    if (answers.length < 2 || answers.slice(0, 2).some((ans) => !hasNonEmptyText(ans))) {
      errors.push(createError(index, 'answers', 'quiz_config.validation.tf_options'))
    }
  }

  const { correctIndex, correctAnswer } = question || {}
  const hasValidIndex = correctIndex === 0 || correctIndex === 1 || typeof correctIndex === 'boolean'
  const hasValidBoolean = correctIndex === undefined && typeof correctAnswer === 'boolean'

  if (!hasValidIndex && !hasValidBoolean) {
    errors.push(createError(index, 'correctIndex', 'quiz_config.validation.tf_answer'))
  }
}

function validateCorrectPicture(question, index, errors) {
  const options = Array.isArray(question?.imageOptions) ? question.imageOptions : null

  if (!options || options.length < 2) {
    errors.push(createError(index, 'imageOptions', 'quiz_config.validation.cp_options', { min: 2 }))
  } else if (options.some((opt) => !hasNonEmptyText(opt?.imageId))) {
    errors.push(createError(index, 'imageOptions', 'quiz_config.validation.cp_images_required'))
  }

  const len = options ? options.length : 0
  if (!Number.isInteger(question?.correctIndex) || question.correctIndex < 0 || question.correctIndex >= len) {
    errors.push(
      createError(index, 'correctIndex', 'quiz_config.validation.correct_answer_required')
    )
  }
}

function validateRange(question, index, errors) {
  const config = question?.rangeConfig
  if (!config || typeof config !== 'object') {
    errors.push(createError(index, 'rangeConfig', 'quiz_config.validation.range_config'))
    return
  }

  const minValue = parseFiniteNumber(config.minValue)
  const maxValue = parseFiniteNumber(config.maxValue)
  const step = parseFiniteNumber(config.step)
  const correctValue = parseFiniteNumber(question?.correctValue)

  if (!Number.isFinite(minValue) || !Number.isFinite(maxValue) || minValue > maxValue) {
    errors.push(createError(index, 'rangeConfig', 'quiz_config.validation.range_min_max'))
  } else if (!Number.isFinite(correctValue) || correctValue < minValue || correctValue > maxValue) {
    errors.push(createError(index, 'correctValue', 'quiz_config.validation.range_target_bounds'))
  }

  if (!Number.isFinite(step) || step <= 0) {
    errors.push(createError(index, 'step', 'quiz_config.validation.range_step'))
  }

  if (question?.scoringConfig?.tolerance !== undefined) {
    const tolerance = parseFiniteNumber(question.scoringConfig.tolerance)
    if (!Number.isFinite(tolerance) || tolerance < 0) {
      errors.push(createError(index, 'tolerance', 'quiz_config.validation.range_tolerance'))
    }
  }
}

function validateNumberQuestion(question, index, errors) {
  const correctNumber = parseFiniteNumber(question?.correctNumber)
  if (!Number.isFinite(correctNumber)) {
    errors.push(createError(index, 'correctNumber', 'quiz_config.validation.number_required'))
  }

  if (question?.scoringConfig?.tolerance !== undefined) {
    const tolerance = parseFiniteNumber(question.scoringConfig.tolerance)
    if (!Number.isFinite(tolerance) || tolerance < 0) {
      errors.push(createError(index, 'tolerance', 'quiz_config.validation.range_tolerance'))
    }
  }
}

// A timeLimit of undefined means "not configured"; anything else must be a
// whole number of seconds >= 0 so the JSON schema (Go int) can round-trip it.
function validateTimeLimit(question, index, errors) {
  if (question?.timeLimit === undefined) return
  const limit = parseFiniteNumber(question.timeLimit)
  if (!Number.isFinite(limit) || !Number.isInteger(limit) || limit < 0) {
    errors.push(createError(index, 'timeLimit', 'quiz_config.validation.time_limit_invalid'))
  }
}

function validateWrongPicturesItem(item, index, textMessageKey, errors) {
  if (!hasNonEmptyText(item?.text)) {
    errors.push(createError(index, 'text', textMessageKey))
  }

  if (!hasNonEmptyText(item?.imageId)) {
    errors.push(createError(index, 'imageId', 'quiz_config.validation.wp_image_required'))
  }

  const targets = item?.wrongPictureConfig?.targets
  if (!Array.isArray(targets) || targets.length === 0) {
    errors.push(
      createError(index, 'targets', 'quiz_config.validation.wp_targets_required', { min: 1 })
    )
  }

  if (item?.wrongPictureConfig?.tolerancePercent !== undefined) {
    const tolerance = parseFiniteNumber(item.wrongPictureConfig.tolerancePercent)
    if (!Number.isFinite(tolerance) || tolerance < 0) {
      errors.push(createError(index, 'tolerancePercent', 'quiz_config.validation.wp_tolerance'))
    }
  }

  validateTimeLimit(item, index, errors)
  validatePoints(item, 'wrong_pictures', index, errors)
}

/**
 * Validate a single quiz or simple_quiz question and return structured error objects.
 *
 * @param {object} question
 * @param {number} [index=0]
 * @param {{ mode?: 'quiz' | 'simple_quiz' }} [options]
 * @returns {Array<{ index: number, field: string, messageKey: string, params: Record<string, any> }>}
 */
export function validateQuizQuestion(question, index = 0, options = {}) {
  const mode = options?.mode === 'simple_quiz' ? 'simple_quiz' : 'quiz'
  const type = mode === 'simple_quiz' ? 'simple_text' : question?.type || 'classic'
  const errors = []

  if (!VALID_QUIZ_TYPES.has(type)) {
    errors.push(createError(index, 'type', 'quiz_config.validation.unknown_type', { type }))
    if (!hasNonEmptyText(question?.text)) {
      errors.push(createError(index, 'text', 'quiz_config.validation.text_required'))
    }
    validatePoints(question, type, index, errors)
    return errors
  }

  if (type === 'wrong_pictures') {
    validateWrongPicturesItem(question, index, 'quiz_config.validation.text_required', errors)
    return errors
  }

  if (type === 'simple_text') {
    if (!hasNonEmptyText(question?.text) && !hasNonEmptyText(question?.imageId)) {
      errors.push(createError(index, 'text', 'quiz_config.validation.simple_text_required'))
    }
    validatePoints(question, type, index, errors)
    return errors
  }

  if (!hasNonEmptyText(question?.text)) {
    errors.push(createError(index, 'text', 'quiz_config.validation.text_required'))
  }

  validateTimeLimit(question, index, errors)
  validatePoints(question, type, index, errors)

  if (SINGLE_CHOICE_TYPES.has(type)) {
    validateChoiceAnswers(question, index, false, errors)
  } else if (type === 'multi_select') {
    validateChoiceAnswers(question, index, true, errors)
  } else if (type === 'true_false') {
    validateTrueFalse(question, index, errors)
  } else if (type === 'correct_picture') {
    validateCorrectPicture(question, index, errors)
  } else if (type === 'range') {
    validateRange(question, index, errors)
  } else if (type === 'number') {
    validateNumberQuestion(question, index, errors)
  }

  return errors
}

/**
 * Validate all questions for a quiz or simple_quiz module.
 *
 * @param {Array<object>} questions
 * @param {{ mode?: 'quiz' | 'simple_quiz' }} [options]
 * @returns {Array<{ index: number, field: string, messageKey: string, params: Record<string, any> }>}
 */
export function validateQuizQuestions(questions, options = {}) {
  if (!Array.isArray(questions) || questions.length === 0) {
    return [createError(-1, 'questions', 'quiz_config.validation.min_one')]
  }

  const errors = []
  questions.forEach((question, index) => {
    errors.push(...validateQuizQuestion(question, index, options))
  })
  return errors
}

/**
 * Validate a single wrong_pictures round and return structured error objects.
 *
 * @param {object} round
 * @param {number} [index=0]
 * @returns {Array<{ index: number, field: string, messageKey: string, params: Record<string, any> }>}
 */
export function validateWrongPicturesRound(round, index = 0) {
  const errors = []
  validateWrongPicturesItem(round, index, 'quiz_config.validation.wp_title_required', errors)
  return errors
}

/**
 * Validate all rounds for a wrong_pictures module.
 *
 * @param {Array<object>} rounds
 * @returns {Array<{ index: number, field: string, messageKey: string, params: Record<string, any> }>}
 */
export function validateWrongPicturesRounds(rounds) {
  if (!Array.isArray(rounds) || rounds.length === 0) {
    return [createError(-1, 'rounds', 'quiz_config.validation.wp_min_one')]
  }

  const errors = []
  rounds.forEach((round, index) => {
    errors.push(...validateWrongPicturesRound(round, index))
  })
  return errors
}

/**
 * Group structured validation errors by question/round index and field name.
 *
 * @param {Array<{ index: number, field: string, messageKey: string, params: Record<string, any> }>} errors
 * @returns {Record<number, Record<string, { index: number, field: string, messageKey: string, params: Record<string, any> }>>}
 */
export function groupErrorsByIndex(errors) {
  if (!Array.isArray(errors)) return {}
  const grouped = {}

  for (const err of errors) {
    if (!err || err.index === undefined || err.index === null) continue
    if (!grouped[err.index]) {
      grouped[err.index] = {}
    }
    if (err.field && !grouped[err.index][err.field]) {
      grouped[err.index][err.field] = err
    }
  }

  return grouped
}

import { describe, expect, it } from 'vitest'
import en from '../src/locales/en.json'
import ru from '../src/locales/ru.json'
import ro from '../src/locales/ro.json'
import {
  groupErrorsByIndex,
  validateQuizQuestion,
  validateQuizQuestions,
  validateWrongPicturesRound,
  validateWrongPicturesRounds,
} from '../src/utils/quizValidation'

const validScoring = (maxPoints = 50) => ({
  mode: 'symmetric',
  maxPoints,
  tolerance: 0,
  partialPercent: 0,
})

describe('quizValidation', () => {
  describe('collection-level checks', () => {
    it('requires at least one question in quiz and simple_quiz modes', () => {
      expect(validateQuizQuestions([])).toEqual([
        { index: -1, field: 'questions', messageKey: 'quiz_config.validation.min_one', params: {} },
      ])
      expect(validateQuizQuestions(null, { mode: 'simple_quiz' })).toEqual([
        { index: -1, field: 'questions', messageKey: 'quiz_config.validation.min_one', params: {} },
      ])
    })

    it('requires at least one round in validateWrongPicturesRounds', () => {
      expect(validateWrongPicturesRounds([])).toEqual([
        { index: -1, field: 'rounds', messageKey: 'quiz_config.validation.wp_min_one', params: {} },
      ])
      expect(validateWrongPicturesRounds(undefined)).toEqual([
        { index: -1, field: 'rounds', messageKey: 'quiz_config.validation.wp_min_one', params: {} },
      ])
    })
  })

  describe('points > 0 across all scored question types and rounds', () => {
    const scoredFixtures = [
      {
        name: 'classic',
        question: {
          type: 'classic',
          text: 'Capital of France?',
          answers: ['Paris', 'Lyon', '', ''],
          correctIndex: 0,
        },
      },
      {
        name: 'multiple_choice',
        question: {
          type: 'multiple_choice',
          text: 'Pick one',
          answers: ['A', 'B'],
          correctIndex: 1,
        },
      },
      {
        name: 'single_select',
        question: {
          type: 'single_select',
          text: 'Pick one',
          answers: ['A', 'B'],
          correctIndex: 0,
        },
      },
      {
        name: 'multi_select',
        question: {
          type: 'multi_select',
          text: 'Pick two',
          answers: ['A', 'B', 'C'],
          correctIndices: [0, 2],
        },
      },
      {
        name: 'true_false',
        question: {
          type: 'true_false',
          text: 'Is sky blue?',
          answers: ['True', 'False'],
          correctIndex: 0,
        },
      },
      {
        name: 'range',
        question: {
          type: 'range',
          text: 'Guess year',
          correctValue: 50,
          rangeConfig: { minValue: 0, maxValue: 100, step: 1 },
        },
      },
      {
        name: 'number',
        question: {
          type: 'number',
          text: 'How many?',
          correctNumber: 42,
        },
      },
      {
        name: 'correct_picture',
        question: {
          type: 'correct_picture',
          text: 'Pick image',
          imageOptions: [
            { id: '1', imageId: 'img-1' },
            { id: '2', imageId: 'img-2' },
          ],
          correctIndex: 1,
        },
      },
      {
        name: 'simple_text',
        question: {
          type: 'simple_text',
          text: 'Open question',
        },
      },
    ]

    it.each(scoredFixtures)('rejects $name when points <= 0 or missing', ({ question }) => {
      for (const badPoints of [0, -10, '', null, undefined, Number.NaN]) {
        const q = { ...question, scoringConfig: { ...validScoring(0), maxPoints: badPoints } }
        const errors = validateQuizQuestion(q, 0)
        expect(errors.some((e) => e.field === 'points')).toBe(true)
      }

      const validQ = { ...question, scoringConfig: validScoring(50) }
      expect(validateQuizQuestion(validQ, 0)).toEqual([])
    })

    it('rejects wrong_pictures questions and rounds when maxPoints <= 0', () => {
      const baseWp = {
        type: 'wrong_pictures',
        text: 'Spot the errors',
        imageId: 'img-wp',
        wrongPictureConfig: {
          targets: [{ id: 't1', xPercent: 20, yPercent: 30, radius: 5 }],
          tolerancePercent: 10,
          maxPoints: 0,
        },
      }

      expect(validateQuizQuestion(baseWp, 0).map((e) => e.field)).toContain('points')
      expect(validateWrongPicturesRound({ ...baseWp, wrongPictureConfig: { ...baseWp.wrongPictureConfig, maxPoints: -5 } }, 0).map((e) => e.field)).toContain('points')
    })

    it('supports top-level points property as fallback', () => {
      const errors = validateQuizQuestion(
        {
          type: 'classic',
          text: 'Question?',
          answers: ['Yes', 'No'],
          correctIndex: 0,
          points: 25,
        },
        0
      )
      expect(errors).toEqual([])
    })
  })

  describe('classic / multiple_choice / single_select', () => {
    it.each(['classic', 'multiple_choice', 'single_select'])(
      'validates %s with >= 2 non-empty answers and valid correctIndex',
      (type) => {
        expect(
          validateQuizQuestion(
            {
              type,
              text: 'Choose wisely',
              answers: ['Alpha', '   ', 'Beta', ''],
              correctIndex: 2,
              scoringConfig: validScoring(50),
            },
            0
          )
        ).toEqual([])
      }
    )

    it('rejects < 2 non-empty answers and invalid or empty-slot correctIndex', () => {
      const errors = validateQuizQuestion(
        {
          type: 'classic',
          text: '  ',
          answers: ['Only one', '   ', '', ''],
          correctIndex: 1,
          scoringConfig: validScoring(50),
        },
        2
      )

      expect(errors).toEqual([
        { index: 2, field: 'text', messageKey: 'quiz_config.validation.text_required', params: {} },
        { index: 2, field: 'answers', messageKey: 'quiz_config.validation.mc_min_answers', params: { min: 2 } },
        { index: 2, field: 'correctIndex', messageKey: 'quiz_config.validation.correct_answer_required', params: {} },
      ])
    })

    it('rejects out-of-range or non-integer correctIndex', () => {
      for (const badIndex of [-1, 4, 1.5, undefined, null, '0']) {
        const errors = validateQuizQuestion(
          {
            type: 'single_select',
            text: 'Pick one',
            answers: ['A', 'B'],
            correctIndex: badIndex,
            scoringConfig: validScoring(50),
          },
          0
        )
        expect(errors).toEqual([
          {
            index: 0,
            field: 'correctIndex',
            messageKey: 'quiz_config.validation.correct_answer_required',
            params: {},
          },
        ])
      }
    })
  })

  describe('multi_select', () => {
    it('passes when >= 2 non-empty answers and >= 2 valid correctIndices are selected', () => {
      const errors = validateQuizQuestion(
        {
          type: 'multi_select',
          text: 'Select primes',
          answers: ['2', '3', '4', ''],
          correctIndices: [0, 1],
          scoringConfig: validScoring(50),
        },
        0
      )
      expect(errors).toEqual([])
    })

    it('rejects multi_select with fewer than 2 valid correctIndices (including 0, 1, duplicates, or empty slots)', () => {
      for (const badIndices of [[], [0], [0, 0], [0, 3], [0, 99]]) {
        const errors = validateQuizQuestion(
          {
            type: 'multi_select',
            text: 'Select all correct',
            answers: ['A', 'B', 'C', ''],
            correctIndices: badIndices,
            scoringConfig: validScoring(50),
          },
          0
        )
        expect(errors).toEqual([
          {
            index: 0,
            field: 'correctIndices',
            messageKey: 'quiz_config.validation.multi_select_min_correct',
            params: { min: 2 },
          },
        ])
      }
    })

    it('rejects extra invalid indices alongside 2 valid correctIndices', () => {
      const errors = validateQuizQuestion(
        {
          type: 'multi_select',
          text: 'Select all correct',
          answers: ['A', 'B', ''],
          correctIndices: [0, 1, 2],
          scoringConfig: validScoring(50),
        },
        0
      )
      expect(errors).toEqual([
        {
          index: 0,
          field: 'correctIndices',
          messageKey: 'quiz_config.validation.correct_answer_required',
          params: {},
        },
      ])
    })

    it('rejects multi_select with fewer than 2 non-empty answers', () => {
      const errors = validateQuizQuestion(
        {
          type: 'multi_select',
          text: 'Select all',
          answers: ['Only A', '', '', ''],
          correctIndices: [0, 1],
          scoringConfig: validScoring(50),
        },
        0
      )
      expect(errors.map((e) => e.field)).toEqual(['answers', 'correctIndices'])
    })
  })

  describe('correct_picture', () => {
    it('passes when >= 2 options all have uploaded imageId and valid correctIndex', () => {
      const errors = validateQuizQuestion(
        {
          type: 'correct_picture',
          text: 'Which flag is Japan?',
          imageOptions: [
            { id: '1', imageId: 'img-1', label: 'A' },
            { id: '2', imageId: 'img-2', label: 'B' },
          ],
          correctIndex: 1,
          scoringConfig: validScoring(50),
        },
        0
      )
      expect(errors).toEqual([])
    })

    it('rejects when fewer than 2 options exist', () => {
      const errors = validateQuizQuestion(
        {
          type: 'correct_picture',
          text: 'Pick image',
          imageOptions: [{ id: '1', imageId: 'img-1' }],
          correctIndex: 0,
          scoringConfig: validScoring(50),
        },
        0
      )
      expect(errors).toEqual([
        {
          index: 0,
          field: 'imageOptions',
          messageKey: 'quiz_config.validation.cp_options',
          params: { min: 2 },
        },
      ])
    })

    it('rejects when >= 2 options exist but any option is missing imageId', () => {
      const errors = validateQuizQuestion(
        {
          type: 'correct_picture',
          text: 'Pick image',
          imageOptions: [
            { id: '1', imageId: 'img-1' },
            { id: '2', imageId: '   ' },
          ],
          correctIndex: 0,
          scoringConfig: validScoring(50),
        },
        1
      )
      expect(errors).toEqual([
        {
          index: 1,
          field: 'imageOptions',
          messageKey: 'quiz_config.validation.cp_images_required',
          params: {},
        },
      ])
    })

    it('rejects out-of-bounds correctIndex on correct_picture', () => {
      const errors = validateQuizQuestion(
        {
          type: 'correct_picture',
          text: 'Pick image',
          imageOptions: [
            { id: '1', imageId: 'img-1' },
            { id: '2', imageId: 'img-2' },
          ],
          correctIndex: 2,
          scoringConfig: validScoring(50),
        },
        0
      )
      expect(errors).toEqual([
        {
          index: 0,
          field: 'correctIndex',
          messageKey: 'quiz_config.validation.correct_answer_required',
          params: {},
        },
      ])
    })
  })

  describe('true_false', () => {
    it('accepts valid index (0 or 1) or boolean answer', () => {
      expect(
        validateQuizQuestion(
          {
            type: 'true_false',
            text: 'Water boils at 100C',
            answers: ['True', 'False'],
            correctIndex: 0,
            scoringConfig: validScoring(50),
          },
          0
        )
      ).toEqual([])

      expect(
        validateQuizQuestion(
          {
            type: 'true_false',
            text: 'The sun orbits the earth',
            correctAnswer: false,
            scoringConfig: validScoring(50),
          },
          0
        )
      ).toEqual([])
    })

    it('rejects empty T/F labels or invalid index/boolean answer', () => {
      const errors = validateQuizQuestion(
        {
          type: 'true_false',
          text: 'True or false?',
          answers: ['True', ''],
          correctIndex: 2,
          scoringConfig: validScoring(50),
        },
        0
      )
      expect(errors).toEqual([
        { index: 0, field: 'answers', messageKey: 'quiz_config.validation.tf_options', params: {} },
        { index: 0, field: 'correctIndex', messageKey: 'quiz_config.validation.tf_answer', params: {} },
      ])
    })
  })

  describe('range and number', () => {
    it('accepts valid range including fixed minValue === maxValue and tolerance === 0', () => {
      expect(
        validateQuizQuestion(
          {
            type: 'range',
            text: 'Exact slider',
            correctValue: 5,
            rangeConfig: { minValue: 5, maxValue: 5, step: 1 },
            scoringConfig: validScoring(100),
          },
          0
        )
      ).toEqual([])
    })

    it('rejects missing rangeConfig, inverted bounds, out-of-bounds target, non-positive step, and negative tolerance', () => {
      expect(
        validateQuizQuestion(
          { type: 'range', text: 'Missing config', scoringConfig: validScoring(100) },
          0
        )
      ).toEqual([
        { index: 0, field: 'rangeConfig', messageKey: 'quiz_config.validation.range_config', params: {} },
      ])

      const invertedErrors = validateQuizQuestion(
        {
          type: 'range',
          text: 'Bad range',
          correctValue: 50,
          rangeConfig: { minValue: 100, maxValue: 10, step: 0 },
          scoringConfig: { maxPoints: 100, tolerance: -5 },
        },
        0
      )
      expect(invertedErrors.map((e) => e.field)).toEqual(['rangeConfig', 'step', 'tolerance'])

      const targetOutErrors = validateQuizQuestion(
        {
          type: 'range',
          text: 'Out of bounds target',
          correctValue: 150,
          rangeConfig: { minValue: 0, maxValue: 100, step: 5 },
          scoringConfig: validScoring(100),
        },
        0
      )
      expect(targetOutErrors).toEqual([
        {
          index: 0,
          field: 'correctValue',
          messageKey: 'quiz_config.validation.range_target_bounds',
          params: {},
        },
      ])
    })

    it('validates number questions including 0 and rejects blank/NaN/negative tolerance', () => {
      expect(
        validateQuizQuestion(
          {
            type: 'number',
            text: 'Freezing point (C)',
            correctNumber: 0,
            scoringConfig: validScoring(50),
          },
          0
        )
      ).toEqual([])

      const errors = validateQuizQuestion(
        {
          type: 'number',
          text: 'How many?',
          correctNumber: '',
          scoringConfig: { maxPoints: 50, tolerance: -1 },
        },
        0
      )
      expect(errors).toEqual([
        { index: 0, field: 'correctNumber', messageKey: 'quiz_config.validation.number_required', params: {} },
        { index: 0, field: 'tolerance', messageKey: 'quiz_config.validation.range_tolerance', params: {} },
      ])
    })
  })

  describe('simple_quiz / simple_text', () => {
    it('validates simple_quiz mode questions without requiring explicit type', () => {
      const ok = validateQuizQuestions(
        [{ text: 'Who wrote Hamlet?', scoringConfig: validScoring(100) }],
        { mode: 'simple_quiz' }
      )
      expect(ok).toEqual([])

      const imageOnly = validateQuizQuestions(
        [{ text: '', imageId: 'img-only', scoringConfig: validScoring(100) }],
        { mode: 'simple_quiz' }
      )
      expect(imageOnly).toEqual([])
    })

    it('rejects simple_text with missing text/image and non-positive points', () => {
      const errors = validateQuizQuestions(
        [{ text: '   ', imageId: null, scoringConfig: validScoring(0) }],
        { mode: 'simple_quiz' }
      )
      expect(errors).toEqual([
        { index: 0, field: 'text', messageKey: 'quiz_config.validation.simple_text_required', params: {} },
        { index: 0, field: 'points', messageKey: 'quiz_config.validation.simple_points', params: {} },
      ])
    })
  })

  describe('wrong_pictures questions and rounds', () => {
    it('passes when title, imageId, >= 1 target pin, and points > 0 are provided', () => {
      const round = {
        type: 'wrong_pictures',
        text: 'Find the hidden mistake',
        imageId: 'img-1',
        wrongPictureConfig: {
          targets: [{ id: 't1', label: 'Clock', xPercent: 50, yPercent: 50, radius: 8 }],
          tolerancePercent: 5,
          maxPoints: 50,
        },
      }
      expect(validateWrongPicturesRounds([round])).toEqual([])
      expect(validateQuizQuestions([round])).toEqual([])
    })

    it('rejects missing title, missing imageId, empty targets, negative tolerance, and points <= 0', () => {
      const badRound = {
        type: 'wrong_pictures',
        text: '  ',
        imageId: '',
        wrongPictureConfig: {
          targets: [],
          tolerancePercent: -2,
          maxPoints: 0,
        },
      }

      const roundErrors = validateWrongPicturesRounds([badRound])
      expect(roundErrors.map((e) => e.field)).toEqual([
        'text',
        'imageId',
        'targets',
        'tolerancePercent',
        'points',
      ])
      expect(roundErrors[0].messageKey).toBe('quiz_config.validation.wp_title_required')

      const quizErrors = validateQuizQuestions([badRound])
      expect(quizErrors.map((e) => e.field)).toEqual([
        'text',
        'imageId',
        'targets',
        'tolerancePercent',
        'points',
      ])
      expect(quizErrors[0].messageKey).toBe('quiz_config.validation.text_required')
    })
  })

  describe('groupErrorsByIndex', () => {
    it('groups errors by question index and field, keeping first error per field', () => {
      const questions = [
        {
          type: 'classic',
          text: '',
          answers: ['Only one', '', '', ''],
          correctIndex: 0,
          scoringConfig: validScoring(0),
        },
        {
          type: 'multi_select',
          text: 'Valid title',
          answers: ['A', 'B', '', ''],
          correctIndices: [0],
          scoringConfig: validScoring(50),
        },
      ]

      const errors = validateQuizQuestions(questions)
      const grouped = groupErrorsByIndex(errors)

      expect(Object.keys(grouped)).toEqual(['0', '1'])
      expect(grouped[0].text.messageKey).toBe('quiz_config.validation.text_required')
      expect(grouped[0].points.messageKey).toBe('quiz_config.validation.points_positive')
      expect(grouped[0].answers.messageKey).toBe('quiz_config.validation.mc_min_answers')
      expect(grouped[1].correctIndices.messageKey).toBe('quiz_config.validation.multi_select_min_correct')
      expect(grouped[1].text).toBeUndefined()
    })

    it('keeps the first error when duplicate (index, field) entries are passed and handles invalid inputs', () => {
      const first = { index: 0, field: 'text', messageKey: 'first', params: {} }
      const second = { index: 0, field: 'text', messageKey: 'second', params: {} }
      expect(groupErrorsByIndex([first, second])).toEqual({ 0: { text: first } })
      expect(groupErrorsByIndex([])).toEqual({})
      expect(groupErrorsByIndex(null)).toEqual({})
    })
  })

  describe('locale coverage', () => {
    it('defines all quiz_config.validation keys across en, ru, and ro', () => {
      const expectedKeys = [
        'min_one',
        'text_required',
        'range_config',
        'range_min_max',
        'range_step',
        'tf_options',
        'cp_options',
        'number_required',
        'mc_options',
        'simple_text_required',
        'simple_points',
        'points_positive',
        'mc_min_answers',
        'correct_answer_required',
        'multi_select_min_correct',
        'cp_images_required',
        'tf_answer',
        'range_target_bounds',
        'range_tolerance',
        'wp_min_one',
        'wp_title_required',
        'wp_image_required',
        'wp_targets_required',
        'wp_tolerance',
        'unknown_type',
      ]

      for (const [localeName, messages] of [
        ['en', en],
        ['ru', ru],
        ['ro', ro],
      ]) {
        const validation = messages?.quiz_config?.validation
        expect(validation, `${localeName} missing quiz_config.validation`).toBeTruthy()
        for (const key of expectedKeys) {
          expect(typeof validation[key], `${localeName} missing ${key}`).toBe('string')
          expect(validation[key].trim().length, `${localeName} empty ${key}`).toBeGreaterThan(0)
        }
      }
    })
  })
})

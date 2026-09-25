<template>
  <section class="wrong-picture-answer-review" aria-labelledby="wrong-picture-review-title">
    <header class="wrong-picture-answer-review__header">
      <div>
        <p class="wrong-picture-answer-review__eyebrow">{{ $t('quiz.player.wrong_picture_review.eyebrow') }}</p>
        <h1 id="wrong-picture-review-title">{{ $t('quiz.player.wrong_picture_review.title') }}</h1>
      </div>
      <div class="wrong-picture-answer-review__total">
        <span>{{ $t('quiz.player.wrong_picture_review.total_score') }}</span>
        <strong>+{{ review.totalScore.toLocaleString() }}</strong>
      </div>
    </header>

    <div class="wrong-picture-answer-review__summary" aria-live="polite">
      <span class="summary-item summary-item--correct">
        <span class="summary-icon">✓</span>
        {{ $t('quiz.player.wrong_picture_review.correct_count', { count: review.correctCount }) }}
      </span>
      <span class="summary-item summary-item--incorrect">
        <span class="summary-icon">×</span>
        {{ $t('quiz.player.wrong_picture_review.incorrect_count', { count: review.incorrectCount }) }}
      </span>
    </div>

    <div v-if="review.guesses.length" class="wrong-picture-answer-review__grid">
      <article
        v-for="guess in review.guesses"
        :key="guess.number"
        class="wrong-picture-answer-card"
        :class="guess.isCorrect ? 'is-correct' : 'is-incorrect'"
      >
        <div class="wrong-picture-answer-card__header">
          <div>
            <span class="wrong-picture-answer-card__number">
              {{ $t('quiz.player.wrong_picture_review.answer', { number: guess.number }) }}
            </span>
            <span class="wrong-picture-answer-card__status">
              <span aria-hidden="true">{{ guess.isCorrect ? '✓' : '×' }}</span>
              {{ guess.isCorrect ? $t('quiz.player.correct') : $t('quiz.player.incorrect') }}
            </span>
          </div>
          <span class="wrong-picture-answer-card__points">
            {{ $t('quiz.player.wrong_picture_review.points', { points: guess.scoreEarned.toLocaleString() }) }}
          </span>
        </div>

        <WrongPictureTargetCrop
          :key="`${guess.number}-${showingCorrectTarget(guess) ? 'target' : 'guess'}`"
          class="wrong-picture-answer-card__crop"
          :image-url="review.imageUrl"
          :target="cropTarget(guess)"
          :target-number="guess.number"
          :marker-status="showingCorrectTarget(guess) ? 'correct' : (guess.isCorrect ? 'correct' : 'incorrect')"
          :label="cropLabel(guess)"
        />

        <p v-if="!guess.isCorrect && nearestTarget(guess)" class="wrong-picture-answer-card__nearest">
          {{ $t('quiz.player.wrong_picture_review.nearest_target', { label: nearestTarget(guess).label || $t('quiz.player.wrong_picture_review.target_number', { number: targetNumber(nearestTarget(guess)) }) }) }}
        </p>

        <button
          v-if="!guess.isCorrect && nearestTarget(guess)"
          class="wrong-picture-answer-card__location-button"
          type="button"
          @click="toggleCorrectLocation(guess)"
        >
          {{ showingCorrectTarget(guess)
            ? $t('quiz.player.wrong_picture_review.show_your_answer')
            : $t('quiz.player.wrong_picture_review.show_correct_location') }}
        </button>
      </article>
    </div>

    <div v-else class="wrong-picture-answer-review__empty">
      <span aria-hidden="true">◎</span>
      <p>{{ $t('quiz.player.wrong_picture_review.no_answers') }}</p>
    </div>
  </section>
</template>

<script setup>
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import WrongPictureTargetCrop from './WrongPictureTargetCrop.vue'

const props = defineProps({
  review: {
    type: Object,
    required: true
  }
})

const { t } = useI18n()
const revealedTargets = reactive({})

const nearestTarget = (guess) => {
  if (!guess.nearestTargetId) return null
  return props.review.targets?.find((target) => target.id === guess.nearestTargetId) || null
}

const targetNumber = (target) => {
  const index = props.review.targets?.findIndex((item) => item.id === target?.id) ?? -1
  return index >= 0 ? index + 1 : '?'
}

const showingCorrectTarget = (guess) => Boolean(!guess.isCorrect && revealedTargets[guess.number])

const toggleCorrectLocation = (guess) => {
  revealedTargets[guess.number] = !revealedTargets[guess.number]
}

const cropTarget = (guess) => {
  const target = nearestTarget(guess)
  if (showingCorrectTarget(guess) && target) return target

  return {
    xPercent: guess.xPercent,
    yPercent: guess.yPercent,
    radius: target?.radius || 5
  }
}

const cropLabel = (guess) => {
  if (showingCorrectTarget(guess)) {
    return nearestTarget(guess)?.label || t('quiz.player.wrong_picture_review.correct_location')
  }

  return guess.isCorrect
    ? nearestTarget(guess)?.label || t('quiz.player.wrong_picture_review.correct_location')
    : ''
}
</script>

<style scoped>
.wrong-picture-answer-review {
  width: min(100%, 1120px);
  height: 100%;
  min-height: 0;
  box-sizing: border-box;
  overflow: auto;
  padding: 8px clamp(16px, 4vw, 48px) 32px;
  color: #fff;
  text-align: left;
}

.wrong-picture-answer-review__header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  padding: 8px 0 20px;
}

.wrong-picture-answer-review__eyebrow {
  margin: 0 0 4px;
  color: rgb(255 255 255 / 65%);
  font-size: 0.8rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.wrong-picture-answer-review h1 {
  margin: 0;
  font-size: clamp(1.6rem, 4vw, 2.5rem);
  line-height: 1.1;
}

.wrong-picture-answer-review__total {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  color: rgb(255 255 255 / 68%);
  font-size: 0.8rem;
  font-weight: 700;
  white-space: nowrap;
}

.wrong-picture-answer-review__total strong {
  color: #32d74b;
  font-size: clamp(1.4rem, 3vw, 2.2rem);
  line-height: 1.1;
}

.wrong-picture-answer-review__summary {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 18px;
  font-size: 0.95rem;
  font-weight: 800;
}

.summary-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 13px;
  border: 1px solid rgb(255 255 255 / 16%);
  border-radius: 999px;
  background: rgb(0 0 0 / 18%);
}

.summary-item--correct {
  border-color: rgb(50 215 75 / 42%);
  color: #d8ffe0;
}

.summary-item--incorrect {
  border-color: rgb(255 99 99 / 42%);
  color: #ffe0e0;
}

.summary-icon {
  display: inline-flex;
  width: 20px;
  height: 20px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 0.85rem;
}

.summary-item--correct .summary-icon {
  background: #32d74b;
  color: #09220e;
}

.summary-item--incorrect .summary-icon {
  background: #ff5d5d;
  color: #2b0808;
}

.wrong-picture-answer-review__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 16px;
  padding-bottom: 12px;
}

.wrong-picture-answer-card {
  min-width: 0;
  padding: 12px;
  border: 2px solid rgb(255 255 255 / 12%);
  border-radius: 16px;
  background: rgb(9 9 20 / 56%);
  box-shadow: 0 12px 30px rgb(0 0 0 / 18%);
}

.wrong-picture-answer-card.is-correct {
  border-color: rgb(50 215 75 / 62%);
}

.wrong-picture-answer-card.is-incorrect {
  border-color: rgb(255 93 93 / 58%);
}

.wrong-picture-answer-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  min-height: 42px;
  padding: 0 2px 10px;
}

.wrong-picture-answer-card__number,
.wrong-picture-answer-card__status,
.wrong-picture-answer-card__points {
  display: block;
}

.wrong-picture-answer-card__number {
  font-size: 1rem;
  font-weight: 850;
}

.wrong-picture-answer-card__status {
  margin-top: 3px;
  color: rgb(255 255 255 / 66%);
  font-size: 0.78rem;
  font-weight: 750;
}

.is-correct .wrong-picture-answer-card__status {
  color: #8dff9f;
}

.is-incorrect .wrong-picture-answer-card__status {
  color: #ff9a9a;
}

.wrong-picture-answer-card__points {
  color: #fff;
  font-size: 0.78rem;
  font-weight: 800;
  text-align: right;
  white-space: nowrap;
}

.wrong-picture-answer-card__crop {
  display: block;
}

.wrong-picture-answer-card__nearest {
  min-height: 2.5em;
  margin: 10px 2px 0;
  color: rgb(255 255 255 / 70%);
  font-size: 0.8rem;
  line-height: 1.25;
}

.wrong-picture-answer-card__location-button {
  width: 100%;
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid rgb(255 255 255 / 24%);
  border-radius: 9px;
  background: rgb(255 255 255 / 10%);
  color: #fff;
  cursor: pointer;
  font: inherit;
  font-size: 0.82rem;
  font-weight: 800;
}

.wrong-picture-answer-card__location-button:hover,
.wrong-picture-answer-card__location-button:focus-visible {
  background: rgb(255 255 255 / 18%);
}

.wrong-picture-answer-review__empty {
  display: flex;
  min-height: 240px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 2px dashed rgb(255 255 255 / 18%);
  border-radius: 16px;
  color: rgb(255 255 255 / 70%);
  text-align: center;
}

.wrong-picture-answer-review__empty > span {
  color: #ff5d5d;
  font-size: 3rem;
}

.wrong-picture-answer-review__empty p {
  margin: 0;
  font-weight: 700;
}

@media (max-width: 600px) {
  .wrong-picture-answer-review {
    padding-right: 12px;
    padding-left: 12px;
  }

  .wrong-picture-answer-review__header {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }

  .wrong-picture-answer-review__total {
    align-items: flex-start;
    flex-direction: row;
    gap: 8px;
  }

  .wrong-picture-answer-review__grid {
    grid-template-columns: 1fr;
  }
}
</style>

<template>
  <div class="quiz-host-screen">
    <div class="bg-layer"></div>
    
    <!-- LOBBY STATE -->
    <div v-if="gameState === 'lobby'" class="state-container lobby-container">
      <div class="lobby-header">
        <h1 class="game-title">Join the Quiz!</h1>
        <div class="pin-display">
          <span class="pin-label">Game PIN:</span>
          <span class="pin-code">{{ pinCode }}</span>
        </div>
        <div class="qr-container">
           <!-- Placeholder for QR code if needed -->
           <div class="qr-placeholder">QR Code</div>
        </div>
      </div>
      
      <div class="players-area">
        <div class="players-count">
          <span>👥 {{ players.length }}</span> Players
        </div>
        <div class="players-grid">
          <div v-for="player in players" :key="player.id" class="player-chip">
            {{ player.nickname }}
          </div>
        </div>
      </div>

      <div class="lobby-footer">
        <button class="start-btn" @click="startQuiz" :disabled="players.length === 0">
          Start Quiz
        </button>
      </div>
    </div>

    <!-- GET READY STATE -->
    <div v-else-if="gameState === 'get-ready'" class="state-container get-ready-container">
      <div class="loading-content">
        <h1>Get Ready!</h1>
        <div class="loader"></div>
      </div>
    </div>

    <!-- QUESTION STATE -->
    <div v-else-if="gameState === 'question-active'" class="state-container question-container">
      <div class="question-header">
        <div class="timer" :class="{ 'timer-low': timeRemaining <= 5 }">
          {{ timeRemaining }}
        </div>
        <div class="question-counter">{{ currentQuestionIndex + 1 }} / {{ totalQuestions }}</div>
      </div>

      <div class="question-content">
        <div class="question-text-wrapper">
             <h2 class="question-text">{{ currentQuestion?.text }}</h2>
        </div>
        
        <!-- Answer Options Grid -->
        <div class="answers-grid" :class="{ 'true-false-grid': isTrueFalse }">
            <div 
                v-for="(answer, idx) in currentQuestion?.answers" 
                :key="idx" 
                class="answer-card"
                :class="getAnswerColorClass(idx)"
                v-show="!isTrueFalse || idx < 2"
            >
                <div class="answer-icon">{{ getShapeIcon(idx) }}</div>
                <div class="answer-text">{{ answer }}</div>
            </div>
        </div>

        <!-- Small Answer Counter in Corner -->
        <div class="answer-counter-widget">
            <div class="counter-value">{{ totalAnswered }}</div>
            <div class="counter-label">Answers</div>
        </div>
      </div>
      
      <div class="host-controls">
         <button v-if="timeRemaining > 0" class="control-btn" @click="endQuestionEarly">Skip Timer</button>
         <button v-else class="control-btn primary" @click="goToResults">Go to Results</button>
      </div>
    </div>

    <!-- RESULTS/LEADERBOARD STATE -->
    <div v-else-if="gameState === 'results' || gameState === 'leaderboard'" class="state-container results-container">
       <h2 class="results-title">{{ gameState === 'results' ? 'Question Results' : 'Leaderboard' }}</h2>
       
       <div v-if="gameState === 'results'" class="correct-answer-reveal">
          <p>Correct Answer:</p>
          <div class="correct-answer-display" :class="getAnswerColorClass(currentQuestion?.correctIndex)">
             {{ getShapeIcon(currentQuestion?.correctIndex) }} 
             {{ currentQuestion?.answers[currentQuestion?.correctIndex] }}
          </div>
       </div>

       <!-- Show Stats in Results -->
       <div v-if="gameState === 'results'" class="answers-stats results-mode">
          <div class="stats-grid">
             <div v-for="(count, idx) in (isTrueFalse ? 2 : 4)" :key="idx" class="stat-column">
                <div class="stat-bar-container">
                  <div 
                    class="stat-bar" 
                    :class="getAnswerColorClass(idx)"
                    :style="{ height: getBarHeight(idx) + '%' }"
                  ></div>
                </div>
                <div class="stat-count">{{ answerCounts[idx] || 0 }}</div>
                <div class="stat-icon">{{ getShapeIcon(idx) }}</div>
             </div>
          </div>
       </div>

       <div class="leaderboard-list">
          <div v-for="(player, idx) in topPlayers" :key="player.id" class="leaderboard-item">
            <div class="rank">{{ idx + 1 }}</div>
            <div class="name">{{ player.nickname }}</div>
            <div class="score">{{ player.score }} pts</div>
          </div>
       </div>

       <div class="host-controls">
         <button class="control-btn primary" @click="nextState">
           {{ gameState === 'results' ? 'Show Leaderboard' : (isLastQuestion ? 'End Quiz' : 'Next Question') }}
         </button>
       </div>
    </div>
    
    <!-- FINISHED STATE -->
    <div v-else-if="gameState === 'finished'" class="state-container finished-container">
       <h1>Quiz Finished!</h1>
       <div class="podium">
          <div class="podium-place second" v-if="topPlayers[1]">
              <div class="podium-bar">
                 <span class="podium-rank">2</span>
                 <span class="podium-name">{{ topPlayers[1].nickname }}</span>
                 <span class="podium-score">{{ topPlayers[1].score }}</span>
              </div>
          </div>
          <div class="podium-place first" v-if="topPlayers[0]">
              <div class="podium-bar">
                 <span class="podium-rank">1</span>
                 <span class="podium-name">{{ topPlayers[0].nickname }}</span>
                 <span class="podium-score">{{ topPlayers[0].score }}</span>
              </div>
          </div>
          <div class="podium-place third" v-if="topPlayers[2]">
               <div class="podium-bar">
                 <span class="podium-rank">3</span>
                 <span class="podium-name">{{ topPlayers[2].nickname }}</span>
                 <span class="podium-score">{{ topPlayers[2].score }}</span>
              </div>
          </div>
       </div>
       <button class="control-btn" @click="exitQuiz">Exit</button>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'

const route = useRoute()
const router = useRouter()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

// State
const gameState = ref('lobby') 
const pinCode = ref(eventId) // Using eventID as PIN for simplicity
const players = ref([])
const questions = ref([])
const currentQuestionIndex = ref(0)
const timeRemaining = ref(20)
const answerCounts = ref({ 0: 0, 1: 0, 2: 0, 3: 0 }) 
const totalAnswered = computed(() => Object.values(answerCounts.value).reduce((a, b) => a + b, 0)) 

const totalQuestions = computed(() => questions.value.length)
const currentQuestion = computed(() => questions.value[currentQuestionIndex.value])
const isLastQuestion = computed(() => currentQuestionIndex.value >= totalQuestions.value - 1)
const isTrueFalse = computed(() => currentQuestion.value?.type === 'true_false')

const topPlayers = computed(() => {
    return [...players.value].sort((a, b) => b.score - a.score).slice(0, 5)
})

const getBarHeight = (idx) => {
    const counts = Object.values(answerCounts.value)
    const max = Math.max(...counts, 1)
    const count = answerCounts.value[idx] || 0
    return (count / max) * 100
}

const getShapeIcon = (idx) => {
    if (isTrueFalse.value) {
        return idx === 0 ? '✔️' : '❌'
    }
    return ['▲', '◆', '●', '■'][idx]
}

const getAnswerColorClass = (idx) => {
    if (isTrueFalse.value) {
        return idx === 0 ? 'color-true' : 'color-false'
    }
    return 'color-' + idx
}

// Timer logic
let timerInterval
const startLocalTimer = (duration) => {
    clearInterval(timerInterval)
    timeRemaining.value = duration
    timerInterval = setInterval(() => {
        if (timeRemaining.value > 0) {
            timeRemaining.value--
        } else {
            clearInterval(timerInterval)
            // Auto-transition to Results when timer hits 0
            if (gameState.value === 'question-active') {
                goToResults()
            }
        }
    }, 1000)
}

// Actions
const startQuiz = () => {
    quizWS.send('host:start_quiz', { moduleId })
}

const endQuestionEarly = () => {
    // This is a host-side visual skip, ideally backend handles text_question
    // But for now, we can just jump to results if we want
    quizWS.send('host:show_results', { moduleId })
}

const goToResults = () => {
    quizWS.send('host:show_results', { moduleId })
}

const nextState = () => {
    if (gameState.value === 'results') {
        // Transition to Leaderboard
        quizWS.send('host:show_leaderboard', { moduleId })
    } else if (gameState.value === 'leaderboard') {
        if (isLastQuestion.value) {
            // End quiz
            quizWS.send('host:end_quiz', { moduleId })
        } else {
            quizWS.send('host:next_question', { moduleId })
        }
    }
}

const exitQuiz = () => {
    // Reset the quiz session before exiting
    quizWS.send('host:reset_quiz', { moduleId })
    quizWS.disconnect()
    router.push({ name: 'admin-event-details', params: { id: eventId } })
}

// WebSocket & Data Loading
onMounted(async () => {
    // 1. Fetch Questions
    try {
        const res = await fetch(`${API_BASE_URL}/events/${eventId}`)
        const data = await res.json()
        const mod = data.modules?.find(m => m.id === moduleId)
        if (mod && mod.data && mod.data.questions) {
            questions.value = mod.data.questions
        }
    } catch (e) {
        console.error('Failed to load questions', e)
    }

    // 2. Connect WS
    quizWS.connect(moduleId, '', true)

    // 3. Listen for updates
    quizWS.on('cmd:state', (payload) => {
        // Handle state transitions from server
        const prevState = gameState.value
        gameState.value = payload.state
        
        if (payload.currentIndex !== undefined) {
            currentQuestionIndex.value = payload.currentIndex
        }
        
        // Handle transitions
        if (payload.state === 'question-active') {
            // Restart timer when entering question state
            if (prevState !== 'question-active' || payload.currentIndex !== currentQuestionIndex.value) {
                startLocalTimer(currentQuestion.value?.timeLimit || 20)
            }
        } else if (payload.state === 'results') {
            clearInterval(timerInterval)
        
            // Reset answer counts for new question
            answerCounts.value = { 0: 0, 1: 0, 2: 0, 3: 0 }
        }
    })

    // Listen for minimal sync updates (answer counts, player info)
    quizWS.on('sync:minimal', (payload) => {
        // Update answer counts if provided
        if (payload.answerCounts) {
            // Convert string keys from JSON to numeric keys for proper reactivity
            answerCounts.value = {
                0: payload.answerCounts["0"] || 0,
                1: payload.answerCounts["1"] || 0,
                2: payload.answerCounts["2"] || 0,
                3: payload.answerCounts["3"] || 0,
            }
        }
    })

    // Listen for full state updates (legacy support - includes players)
    quizWS.on('state:update', (session) => {
        // Update players
        if (session.Players || session.players) {
            const playersMap = session.Players || session.players
            players.value = Object.values(playersMap)
        }

        // Update Answer Counts (convert string keys to numeric keys)
        if (session.AnswerCounts || session.answerCounts) {
             const counts = session.AnswerCounts || session.answerCounts
            answerCounts.value = {
                0: counts["0"] || 0,
                1: counts["1"] || 0,
                2: counts["2"] || 0,
                3: counts["3"] || 0,
            }
        }
    })
 })

onUnmounted(() => {
    clearInterval(timerInterval)
    quizWS.disconnect()
})
</script>

<style scoped>
.quiz-host-screen {
    width: 100vw;
    height: 100vh;
    background-color: #121212;
    color: white;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    font-family: 'Inter', sans-serif;
}

/* Dynamic background */
.bg-layer {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    background: radial-gradient(circle at center, #2c2c2e 0%, #000 100%);
    z-index: 0;
}

.state-container {
    z-index: 1;
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
    padding: 20px;
}

/* LOBBY */
.lobby-header {
    text-align: center;
    margin-top: 40px;
}

.game-title { font-size: 3rem; margin-bottom: 20px; text-shadow: 0 4px 10px rgba(0,0,0,0.5); }

.pin-display {
    background: white;
    color: black;
    padding: 10px 30px;
    border-radius: 8px;
    font-size: 2rem;
    font-weight: bold;
    display: flex;
    gap: 10px;
    justify-content: center;
    align-items: center;
}

.qr-container {
    margin-top: 20px;
    width: 150px;
    height: 150px;
    background: white;
    margin-left: auto;
    margin-right: auto;
    display: flex;
    align-items: center;
    justify-content: center;
    color: black;
}

.players-area {
    margin-top: 40px;
    width: 100%;
    max-width: 1000px;
}

.players-count {
    font-size: 1.5rem;
    margin-bottom: 20px;
    text-align: center;
}

.players-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    justify-content: center;
}

.player-chip {
    background-color: rgba(255,255,255,0.1);
    padding: 10px 20px;
    border-radius: 20px;
    font-weight: 600;
    animation: popIn 0.3s ease-out;
}

@keyframes popIn {
    0% { transform: scale(0); opacity: 0; }
    80% { transform: scale(1.1); }
    100% { transform: scale(1); opacity: 1; }
}

.lobby-footer {
    margin-top: auto;
    margin-bottom: 40px;
}

.start-btn {
    background-color: #32D74B;
    color: white;
    border: none;
    font-size: 1.5rem;
    padding: 15px 40px;
    border-radius: 8px;
    cursor: pointer;
    font-weight: bold;
    transition: transform 0.1s;
}
.start-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.start-btn:hover:not(:disabled) { transform: scale(1.05); }

/* QUESTION */
.question-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    padding: 2vw; /* Use viewport units for scaling */
    box-sizing: border-box;
}

.question-header {
    width: 100%;
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 10vh; /* Fixed header height */
}

.timer {
    font-size: 2.5rem;
    font-weight: 900;
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: rgba(255,255,255,0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    border: 4px solid #0A84FF;
}

.timer-low { border-color: #FF453A; color: #FF453A; animation: pulse 1s infinite; }

@keyframes pulse {
    0% { transform: scale(1); }
    50% { transform: scale(1.1); }
    100% { transform: scale(1); }
}

.question-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: space-between; /* Space between question and answers */
    width: 100%;
    margin-top: 20px;
    position: relative; /* For absolute positioning of counter */
}

.question-text-wrapper {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
}

.question-text {
    font-size: 4vw; /* Huge text responsive to screen width */
    text-align: center;
    font-weight: 800;
    line-height: 1.2;
    text-shadow: 0 4px 8px rgba(0,0,0,0.5);
}

/* Answers Grid */
.answers-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-template-rows: 1fr 1fr;
    gap: 20px;
    height: 40vh; /* Take up bottom 40% of screen */
    width: 100%;
}

.answer-card {
    display: flex;
    align-items: center;
    padding: 20px;
    border-radius: 8px;
    background: white; /* Fallback */
    box-shadow: 0 4px 10px rgba(0,0,0,0.3);
    transition: transform 0.1s;
    overflow: hidden;
}

.answer-icon {
    font-size: 4rem;
    margin-right: 20px;
    opacity: 0.8;
}

.answer-text {
    font-size: 2.5rem;
    font-weight: 700;
    color: white;
    text-shadow: 1px 1px 2px rgba(0,0,0,0.5);
    word-break: break-word; /* Wrap long text */
}

/* Answer Count Widget */
.answer-counter-widget {
    position: absolute;
    bottom: -10px; /* Slight adjust */
    right: -10px;
    background: #2C2C2E;
    padding: 15px 25px;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    align-items: center;
    box-shadow: 0 0 15px rgba(0,0,0,0.5);
    z-index: 10;
    border: 2px solid #555;
}

.counter-value {
    font-size: 2.5rem;
    font-weight: 900;
    line-height: 1;
}

.counter-label {
    font-size: 0.8rem;
    text-transform: uppercase;
    opacity: 0.7;
    margin-top: 5px;
}

.color-0 { background-color: #FF3B30; }
.color-1 { background-color: #0A84FF; }
.color-2 { background-color: #FFD60A; }
.color-3 { background-color: #32D74B; }

.color-true { background-color: #0A84FF; }
.color-false { background-color: #FF3B30; }

.true-false-grid {
    grid-template-rows: 1fr;
    grid-template-columns: 1fr 1fr;
}

/* GET READY */
.get-ready-container {
    justify-content: center;
    background-color: #FF3B30;
}
.loading-content {
    text-align: center;
}
.loader {
    width: 60px;
    height: 60px;
    border: 6px solid rgba(255,255,255,0.3);
    border-top: 6px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 30px auto;
}
@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

/* RESULTS/LEADERBOARD */
.results-container {
    padding-top: 60px;
}

.correct-answer-reveal {
    margin-bottom: 40px;
    text-align: center;
}
.correct-answer-display {
    font-size: 2rem;
    padding: 10px 20px;
    border-radius: 8px;
    margin-top: 10px;
    color: white; 
    text-shadow: 0 1px 2px rgba(0,0,0,0.5);
}

.leaderboard-list {
    width: 100%;
    max-width: 800px;
    margin-bottom: 40px;
}

.leaderboard-item {
    display: flex;
    align-items: center;
    background: #2C2C2E;
    margin-bottom: 10px;
    padding: 15px;
    border-radius: 8px;
    font-size: 1.2rem;
}

.rank { width: 40px; font-weight: bold; color: #FFD60A; }
.name { flex: 1; font-weight: 600; }
.score { font-weight: bold; }

.control-btn {
    padding: 12px 24px;
    border-radius: 8px;
    border: none;
    font-size: 1rem;
    cursor: pointer;
    background: #3A3A3C; 
    color: white;
}
.control-btn.primary { background: #0A84FF; }

/* ANSWERS STATS (Results Mode) */
.answers-stats {
    height: 300px;
    width: 100%;
}

.stats-grid {
    display: flex;
    height: 100%;
    align-items: flex-end;
    gap: 20px;
    padding: 0 40px;
}

.stat-column {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    height: 100%;
    justify-content: flex-end;
}

.stat-bar-container {
    flex: 1;
    width: 100%;
    display: flex;
    align-items: flex-end;
    justify-content: center;
}

.stat-bar {
    width: 60%;
    min-height: 10px;
    transition: height 0.5s ease-out;
}

.stat-count {
    font-size: 1.5rem;
    font-weight: bold;
    margin-top: 10px;
}

.stat-icon {
    font-size: 1.5rem;
    margin-top: 5px;
}

/* PODIUM */
.podium {
    display: flex;
    align-items: flex-end;
    height: 400px;
    gap: 20px;
    margin-top: 40px;
    margin-bottom: 40px;
}

.podium-place {
    width: 120px;
    display: flex;
    flex-direction: column;
    align-items: center;
}

.podium-bar {
    width: 100%;
    background: #3A3A3C;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
    padding-bottom: 20px;
}

.second .podium-bar { height: 200px; background: #C0C0C0; color: black; }
.first .podium-bar { height: 300px; background: #FFD700; color: black; box-shadow: 0 0 20px rgba(255, 215, 0, 0.5); }
.third .podium-bar { height: 150px; background: #CD7F32; color: black; }

.podium-rank { font-size: 2rem; font-weight: 900; margin-bottom: 10px; }
.podium-name { font-weight: bold; margin-bottom: 5px; }
</style>

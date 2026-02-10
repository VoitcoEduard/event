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

    <!-- QUESTION STATE -->
    <div v-else-if="gameState === 'question'" class="state-container question-container">
      <div class="question-header">
        <div class="question-counter">{{ currentQuestionIndex + 1 }} / {{ totalQuestions }}</div>
        <div class="timer" :class="{ 'timer-low': timeRemaining <= 5 }">
          {{ timeRemaining }}
        </div>
      </div>

      <div class="question-content">
        <h2 class="question-text">{{ currentQuestion?.text }}</h2>
        
        <!-- Answer Histogram for Host -->
        <div class="answers-stats">
          <div class="stats-grid">
             <div v-for="(count, idx) in 4" :key="idx" class="stat-column">
                <div class="stat-bar-container">
                  <div 
                    class="stat-bar" 
                    :class="'color-' + idx"
                    :style="{ height: getBarHeight(idx) + '%' }"
                  ></div>
                </div>
                <div class="stat-count">{{ answerCounts[idx] || 0 }}</div>
                <div class="stat-icon">{{ getShapeIcon(idx) }}</div>
             </div>
          </div>
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
          <div class="correct-answer-display" :class="'color-' + currentQuestion?.correctIndex">
             {{ getShapeIcon(currentQuestion?.correctIndex) }} 
             {{ currentQuestion?.answers[currentQuestion?.correctIndex] }}
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
           {{ isLastQuestion ? 'End Quiz' : 'Next Question' }}
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

const totalQuestions = computed(() => questions.value.length)
const currentQuestion = computed(() => questions.value[currentQuestionIndex.value])
const isLastQuestion = computed(() => currentQuestionIndex.value >= totalQuestions.value - 1)

const topPlayers = computed(() => {
    return [...players.value].sort((a, b) => b.score - a.score).slice(0, 5)
})

const getBarHeight = (idx) => {
    const counts = Object.values(answerCounts.value)
    const max = Math.max(...counts, 1)
    const count = answerCounts.value[idx] || 0
    return (count / max) * 100
}

const getShapeIcon = (idx) => ['▲', '◆', '●', '■'][idx]

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
            // Auto-transition? Maybe wait for host.
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
        gameState.value = 'leaderboard'
    } else if (gameState.value === 'leaderboard') {
        if (isLastQuestion.value) {
            // End quiz
            quizWS.send('host:end_quiz', { moduleId })
            gameState.value = 'finished'
        } else {
            quizWS.send('host:next_question', { moduleId })
        }
    }
}

const exitQuiz = () => {
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
    quizWS.on('state:update', (session) => {
        // Sync local state with server session
        
        // Update players
        if (session.players) {
            players.value = Object.values(session.players)
        }

        // Update Game State
        if (session.state) {
            const prevState = gameState.value
            gameState.value = session.state
            
            // Handle transitions
            if (session.state === 'question') {
                const prevIndex = currentQuestionIndex.value
                
                // New question started
                if (session.currentIndex !== undefined) {
                    currentQuestionIndex.value = session.currentIndex
                }
                
                // Restart timer ONLY if:
                // 1. We transitioned from a non-question state
                // 2. OR we moved to a new question index
                if (prevState !== 'question' || (session.currentIndex !== undefined && prevIndex !== session.currentIndex)) {
                     startLocalTimer(currentQuestion.value?.timeLimit || 20)
                }
            } else if (session.state === 'results') {
                clearInterval(timerInterval)
            }
        }

        // Update Answer Counts
        if (session.answers) {
            const counts = { 0: 0, 1: 0, 2: 0, 3: 0 }
            Object.values(session.answers).forEach(ansIdx => {
                counts[ansIdx] = (counts[ansIdx] || 0) + 1
            })
            answerCounts.value = counts
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
    justify-content: space-between;
    padding-bottom: 40px;
}

.question-header {
    width: 100%;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0 40px;
}

.timer {
    font-size: 3rem;
    font-weight: 900;
    width: 80px;
    height: 80px;
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
    justify-content: center;
    width: 100%;
    max-width: 1200px;
}

.question-text {
    font-size: 2.5rem;
    text-align: center;
    margin-bottom: 60px;
}

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

.color-0 { background-color: #FF3B30; }
.color-1 { background-color: #0A84FF; }
.color-2 { background-color: #FFD60A; }
.color-3 { background-color: #32D74B; }

/* LEADERBOARD/RESULTS */
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
    color: white; /* Depends on brightness, but usually white text on these colors is fine or we add shadow */
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

<template>
  <div class="quiz-player-screen">
    <div class="bg-layer"></div>

    <!-- NICKNAME ENTRY (Pre-Lobby) -->
    <div v-if="gameState === 'enter-nickname'" class="state-container name-entry-container">
      <h1 class="logo">QuizTime!</h1>
      <div class="card">
        <input 
          v-model="nickname" 
          placeholder="Enter Nickname" 
          class="nickname-input"
          @keyup.enter="joinLobby"
        />
        <button class="join-btn" @click="joinLobby" :disabled="!nickname.trim()">
          OK, Go!
        </button>
      </div>
    </div>

    <!-- LOBBY STATE -->
    <div v-else-if="gameState === 'lobby'" class="state-container lobby-container">
      <div class="lobby-content">
        <h2 class="waiting-text">You're in!</h2>
        <div class="nickname-display">{{ nickname }}</div>
        <p class="status-msg">See your name on screen?</p>
        <div class="loader"></div>
      </div>
    </div>

    <!-- GET READY STATE -->
    <div v-else-if="gameState === 'get-ready'" class="state-container get-ready-container">
      <div class="ready-content">
        <h1>Question {{ currentQuestionIndex + 1 }}</h1>
        <div class="countdown">{{ timeRemaining }}</div>
      </div>
    </div>

    <!-- QUESTION/ANSWER STATE -->
    <div v-else-if="gameState === 'question'" class="state-container question-container">
      <div class="question-info">
         <div class="q-number">{{ currentQuestionIndex + 1 }}</div>
         <!-- Timer bar? -->
      </div>
      
      <div class="answer-grid">
         <button 
            v-for="(shape, idx) in shapes" 
            :key="idx"
            class="answer-btn"
            :class="['color-' + idx, { 'selected': selectedAnswer === idx }]"
            @click="submitAnswer(idx)"
            :disabled="selectedAnswer !== null"
         >
            <span class="shape-icon">{{ shape }}</span>
         </button>
      </div>
    </div>

    <!-- ANSWER SUBMITTED (Waiting) -->
    <div v-else-if="gameState === 'answered'" class="state-container answered-container">
       <div class="answered-content">
          <div class="spinner"></div>
          <h2>Answer Submitted!</h2>
          <p>Waiting for others...</p>
       </div>
    </div>

    <!-- RESULT STATE -->
    <div v-else-if="gameState === 'result'" class="state-container result-container" :class="isCorrect ? 'bg-correct' : 'bg-wrong'">
       <div class="result-content">
          <div class="result-icon">{{ isCorrect ? '✅' : '❌' }}</div>
          <h1 class="result-title">{{ isCorrect ? 'Correct' : 'Incorrect' }}</h1>
          
          <div class="score-summary">
             <div class="score-delta" v-if="isCorrect">+ {{ currentPoints }}</div>
             <div class="streak-msg" v-if="streak > 2">🔥 Streak: {{ streak }}</div>
             <div class="rank-msg">You are in {{ rank }} place</div>
          </div>
       </div>
       <div class="footer-score">
          Total Score: {{ score }}
       </div>
    </div>

    <!-- FINISHED STATE -->
    <div v-else-if="gameState === 'finished'" class="state-container finished-container">
       <div class="finished-content">
          <h1>Quiz Over</h1>
          <div class="final-rank-badge">
             <span class="rank-num">{{ rank }}</span>
             <span class="rank-label">place</span>
          </div>
          <p class="final-score">Final Score: {{ score }}</p>
       </div>
       <button class="exit-btn" @click="goHome">Exit</button>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'

const router = useRouter()
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

// State
const nickname = ref('')
const gameState = ref('enter-nickname') 
const selectedAnswer = ref(null)
const isCorrect = ref(false)
const score = ref(0)
const currentPoints = ref(0)
const streak = ref(0)
const rank = ref(0)
const currentQuestionIndex = ref(0)
const timeRemaining = ref(5)
const questions = ref([])

const shapes = ['▲', '◆', '●', '■']

// Actions
const joinLobby = () => {
    if (!nickname.value.trim()) return
    gameState.value = 'lobby'
    quizWS.connect(moduleId, nickname.value, false)
}

const submitAnswer = (idx) => {
    selectedAnswer.value = idx
    gameState.value = 'answered'
    quizWS.send('player:answer', { answerIndex: idx })
}

const goHome = () => {
    quizWS.disconnect()
    router.push('/')
}

let timerInterval

// WS & Logic
onMounted(async () => {
    // 1. Fetch Questions logic
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

    // 2. Listen for WS updates
    quizWS.on('state:update', (session) => {
        // Update local player stats if available
        if (session.players && session.players[nickname.value]) {
            const myData = session.players[nickname.value]
            const oldScore = score.value
            score.value = myData.score || 0
            
            // Calculate rank
            const allPlayers = Object.values(session.players)
            allPlayers.sort((a, b) => b.score - a.score)
            rank.value = allPlayers.findIndex(p => p.nickname === nickname.value) + 1
        }

        if (session.state) {
            const prevState = gameState.value
            
            // Handle State Transitions
            if (session.state === 'question') {
                // If it's a NEW question or just reconnecting
                if (prevState !== 'question' && prevState !== 'answered') {
                    currentQuestionIndex.value = session.currentIndex
                    selectedAnswer.value = null
                    
                    // Simple "Get Ready" countdown if we were waiting
                    if (prevState === 'lobby' || prevState === 'results' || prevState === 'leaderboard') {
                         gameState.value = 'get-ready'
                         timeRemaining.value = 3
                         clearInterval(timerInterval)
                         timerInterval = setInterval(() => {
                             timeRemaining.value--
                             if (timeRemaining.value <= 0) {
                                 clearInterval(timerInterval)
                                 gameState.value = 'question'
                             }
                         }, 1000)
                    } else {
                        // Late join or reconnect
                        gameState.value = 'question'
                    }
                }
            } else if (session.state === 'results') {
                // Check if correct
                const currentQ = questions.value[currentQuestionIndex.value]
                if (currentQ) {
                    const correctIdx = currentQ.correctIndex
                    isCorrect.value = (selectedAnswer.value === correctIdx)
                    
                    if (isCorrect.value) {
                         // Points calculation is simplified here, ideally derived from score delta
                         currentPoints.value = 1000 
                         streak.value++
                    } else {
                         currentPoints.value = 0
                         streak.value = 0
                    }
                }
                gameState.value = 'result'
            } else if (session.state === 'finished') {
                gameState.value = 'finished'
            } else if (session.state === 'lobby') {
                if (gameState.value !== 'enter-nickname') {
                   gameState.value = 'lobby'
                }
            } else if (session.state === 'leaderboard') {
                // Determine if we show leaderboard or keep showing "result"
                // Usually players see leaderboard too
                // For MVP, maybe keep result screen or show simple rank
                // Let's stick to result screen because it shows rank
                gameState.value = 'result' 
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
.quiz-player-screen {
    width: 100vw;
    height: 100vh;
    background-color: #46178F; /* Standard Kahoot-like purple */
    color: white;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    font-family: 'Inter', sans-serif;
}

.bg-layer {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    background: radial-gradient(circle at center, rgba(0,0,0,0) 0%, rgba(0,0,0,0.2) 100%);
    z-index: 0;
}

.state-container {
    z-index: 1;
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
}

/* NAME ENTRY */
.name-entry-container {
    justify-content: center;
    padding: 20px;
}

.logo {
    font-size: 3rem;
    font-weight: 900;
    margin-bottom: 40px;
    text-shadow: 2px 2px 0px #000;
}

.card {
    background: white;
    padding: 20px;
    border-radius: 4px;
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
}

.nickname-input {
    padding: 15px;
    border: 2px solid #ccc;
    border-radius: 4px;
    font-size: 1.2rem;
    text-align: center;
    font-weight: bold;
}

.join-btn {
    background-color: #333;
    color: white;
    padding: 15px;
    border: none;
    border-radius: 4px;
    font-size: 1.2rem;
    font-weight: bold;
    cursor: pointer;
}
.join-btn:disabled { opacity: 0.5; }

/* LOBBY */
.lobby-container {
    justify-content: center;
    text-align: center;
}

.nickname-display {
    font-size: 2.5rem;
    font-weight: 900;
    margin: 20px 0;
}

.loader {
    width: 40px;
    height: 40px;
    border: 4px solid rgba(255,255,255,0.3);
    border-top: 4px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 20px auto;
}

@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

/* GET READY */
.get-ready-container {
    justify-content: center;
    background-color: #FF3B30; /* Red alert for ready */
}
.ready-content h1 { font-size: 2rem; margin-bottom: 20px; }
.countdown { font-size: 5rem; font-weight: 900; }

/* QUESTION */
.question-container {
    justify-content: flex-end; /* Buttons at bottom usually */
    padding-bottom: 20px;
}

.question-info {
    position: absolute;
    top: 20px;
    left: 20px;
    background: white;
    color: black;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
}

.answer-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    width: 100%;
    padding: 10px;
    height: 60%; /* Take up bottom half */
}

.answer-btn {
    border: none;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 4rem; /* Big icon */
    cursor: pointer;
    transition: transform 0.1s;
    box-shadow: 0 4px 0 rgba(0,0,0,0.2);
}
.answer-btn:active { transform: translateY(4px); box-shadow: none; }

.color-0 { background-color: #FF3B30; }
.color-1 { background-color: #0A84FF; }
.color-2 { background-color: #FFD60A; }
.color-3 { background-color: #32D74B; }

/* ANSWERED */
.answered-container {
    justify-content: center;
    text-align: center;
    background-color: #0A84FF;
}

/* RESULT */
.result-container {
    justify-content: center;
    text-align: center;
}
.bg-correct { background-color: #32D74B; }
.bg-wrong { background-color: #FF3B30; }

.result-icon { font-size: 5rem; margin-bottom: 20px; }
.result-title { font-size: 2.5rem; font-weight: 900; margin-bottom: 10px; }

.score-summary {
    background: rgba(0,0,0,0.2);
    padding: 20px;
    border-radius: 8px;
    min-width: 200px;
}

.score-delta { font-size: 2rem; font-weight: bold; margin-bottom: 10px; }

.footer-score {
    position: absolute;
    bottom: 20px;
    background: rgba(0,0,0,0.3);
    padding: 5px 15px;
    border-radius: 20px;
    font-weight: bold;
}

/* FINISHED */
.finished-container {
    justify-content: center;
    text-align: center;
}

.final-rank-badge {
    background: #FFD700; /* Gold */
    color: black;
    width: 150px;
    height: 150px;
    border-radius: 50%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    margin: 30px auto;
    box-shadow: 0 0 20px rgba(255, 215, 0, 0.5);
    animation: bounce 2s infinite;
}

@keyframes bounce {
    0%, 20%, 50%, 80%, 100% {transform: translateY(0);}
    40% {transform: translateY(-20px);}
    60% {transform: translateY(-10px);}
}

.rank-num { font-size: 4rem; font-weight: 900; line-height: 1; }
.rank-label { font-size: 1rem; font-weight: 600; text-transform: uppercase; }

.exit-btn {
    margin-top: 40px;
    background: rgba(255,255,255,0.2);
    border: 2px solid white;
    color: white;
    padding: 10px 30px;
    border-radius: 20px;
    cursor: pointer;
    font-weight: bold;
}

</style>

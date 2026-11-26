import { ref } from 'vue'
import { quizWS } from '../services/websocket'

const States = {
    LOBBY: 'lobby',
    GET_READY: 'get-ready',
    QUESTION_PRESENTING: 'question-presenting',
    QUESTION_ACTIVE: 'question-active',
    ANSWERED: 'answered',
    QUESTION_ENDED: 'question-ended',
    RESULTS: 'results',
    LEADERBOARD: 'leaderboard',
    FINISHED: 'finished',
    BUFFERING: 'buffering',
}

const transitions = {
    [States.LOBBY]: [States.GET_READY, States.QUESTION_PRESENTING, States.LOBBY],
    [States.GET_READY]: [States.QUESTION_PRESENTING, States.QUESTION_ACTIVE, States.BUFFERING],
    [States.QUESTION_PRESENTING]: [States.QUESTION_ACTIVE],
    [States.QUESTION_ACTIVE]: [States.ANSWERED, States.QUESTION_ENDED, States.RESULTS],
    [States.ANSWERED]: [States.QUESTION_ENDED, States.RESULTS, States.LEADERBOARD],
    [States.QUESTION_ENDED]: [States.RESULTS],
    [States.RESULTS]: [States.LEADERBOARD],
    [States.LEADERBOARD]: [States.GET_READY, States.QUESTION_PRESENTING, States.FINISHED, States.QUESTION_ACTIVE],
    [States.BUFFERING]: [States.GET_READY, States.QUESTION_PRESENTING],
}

export function useQuizStateMachine({ requestBatch = null } = {}) {
    const state = ref(States.LOBBY)
    const questionCache = ref(new Map())
    const currentQuestionIndex = ref(0)
    const score = ref(0)
    const streak = ref(0)

    function canTransition(to) {
        return transitions[state.value]?.includes(to) ?? false
    }

    function transition(to, data = null, { authoritative = false } = {}) {
        if (!Object.values(States).includes(to) || (!authoritative && !canTransition(to))) {
            console.warn(`Invalid transition: ${state.value} -> ${to}`)
            return
        }

        onExitState(state.value)

        state.value = to
        onEnterState(to, data)
    }

    function onExitState(oldState) {
    }

    function onEnterState(newState, data) {
        switch (newState) {
            case States.QUESTION_ACTIVE:
                startLocalTimer(data?.timeLimit || 20)
                break
            case States.ANSWERED:
                validateAndScore(data)
                break
            case States.BUFFERING:
                requestNextBatch()
                break
        }
    }

    function startLocalTimer(timeLimit) {
    }

    function validateAndScore(data) {
    }

    function requestNextBatch() {
        const batchIndex = Math.floor(currentQuestionIndex.value / 50)
        const request = requestBatch || ((index) => {
            quizWS.send('player:batch_request', { currentBatchIndex: index })
        })

        request(batchIndex)
    }

    return {
        state,
        transition,
        canTransition,
        requestNextBatch,
        questionCache,
        currentQuestionIndex,
        score,
        streak,
        States,
    }
}

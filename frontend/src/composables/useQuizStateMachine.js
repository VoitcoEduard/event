import { ref } from 'vue'

// Define possible states
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

// Define valid transitions
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

export function useQuizStateMachine() {
    const state = ref(States.LOBBY)
    const questionCache = ref(new Map())
    const currentQuestionIndex = ref(0)
    const score = ref(0)
    const streak = ref(0)

    function canTransition(to) {
        return transitions[state.value]?.includes(to) ?? false
    }

    function transition(to, data = null) {
        if (!canTransition(to)) {
            console.warn(`Invalid transition: ${state.value} -> ${to}`)
            return
        }

        // Exit current state
        onExitState(state.value)

        // Enter new state
        state.value = to
        onEnterState(to, data)
    }

    function onExitState(oldState) {
        // Clean up any state specific resources
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
        // Timer logic to be implemented
        console.log(`Question active for ${timeLimit} seconds`)
    }

    function validateAndScore(data) {
        // Answer validation is handled by the view component
        // console.log('State machine: validateAndScore triggered', data)
    }

    function requestNextBatch() {
        // Batch request logic to be implemented
        console.log('Requesting next batch')
    }

    return {
        state,
        transition,
        canTransition,
        questionCache,
        currentQuestionIndex,
        score,
        streak,
        States,
    }
}